package server

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/bytedance/sonic"
	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/nrmnqdds/gomaluum/pkg/utils"
)

// Object pools for result processing
var resultPool = sync.Pool{
	New: func() any {
		return &dtos.Result{}
	},
}

var resultStringSlicePool = sync.Pool{
	New: func() any {
		return make([]string, 0, 10)
	},
}

// Worker pool structures for results
type resultJob struct {
	query string
	name  string
}

type resultWorkerResult struct {
	result dtos.ResultResponse
	err    error
}

// Parse result table row with object pooling
func parseResultRow(tds []string, subjects *[]dtos.Result, gpaInfo *map[string]string, mu *sync.Mutex) {
	if len(tds) < 4 {
		return
	}

	courseCode := strings.TrimSpace(tds[0])
	courseName := strings.TrimSpace(tds[1])
	courseGrade := strings.TrimSpace(tds[2])
	courseCredit := strings.TrimSpace(tds[3])

	words := strings.Fields(courseCode)
	if len(words) == 0 {
		return
	}

	// Handle GPA information row
	if words[0] == "Total" {
		mu.Lock()
		gpaWords := strings.Fields(courseName)

		if len(gpaWords) > 1 {
			(*gpaInfo)["chr"] = strings.TrimSpace(gpaWords[1])
		}
		if len(gpaWords) > 2 {
			(*gpaInfo)["gpa"] = strings.TrimSpace(gpaWords[2])
		}
		if len(gpaWords) > 3 {
			(*gpaInfo)["status"] = strings.TrimSpace(gpaWords[3])
		}

		cgpaWords := strings.Fields(courseCredit)
		if len(cgpaWords) > 2 {
			(*gpaInfo)["cgpa"] = strings.TrimSpace(cgpaWords[2])
		}
		mu.Unlock()
		return
	}

	// Create result object
	result := resultPool.Get().(*dtos.Result)
	*result = dtos.Result{} // Reset

	result.ID = fmt.Sprintf("gomaluum:subject:%s", cuid.Slug())
	result.CourseCode = courseCode
	result.CourseName = courseName
	result.CourseGrade = courseGrade
	result.CourseCredit = courseCredit

	mu.Lock()
	*subjects = append(*subjects, *result)
	mu.Unlock()

	resultPool.Put(result)
}

// Worker function for processing result sessions
func (s *Server) resultWorker(jobs <-chan resultJob, results chan<- resultWorkerResult, cookie string) {
	cookieStr := "MOD_AUTH_CAS=" + cookie

	for job := range jobs {
		func() {
			defer utils.CatchPanic("result worker")

			c := colly.NewCollector(
				colly.Headers(
					map[string]string{
						"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
						"Accept-Language": "en-US,en;q=0.9",
						"Cookie":          cookieStr,
					},
				),
				colly.UserAgent(constants.UserAgent),
			)
			c.WithTransport(s.httpClient.Transport)

			var (
				mu       sync.Mutex
				subjects []dtos.Result
				gpaInfo  = map[string]string{
					"gpa":    "0",
					"cgpa":   "0",
					"chr":    "0",
					"status": "0",
				}
			)

			c.OnHTML("table.table-hover tbody tr", func(e *colly.HTMLElement) {
				cells := e.DOM.Find("td")
				if cells.Length() == 0 {
					return
				}

				tds := resultStringSlicePool.Get().([]string)
				tds = tds[:0] // Reset slice

				cells.Each(func(_ int, s *goquery.Selection) {
					tds = append(tds, s.Text())
				})

				parseResultRow(tds, &subjects, &gpaInfo, &mu)
				resultStringSlicePool.Put(tds)
			})

			url := constants.ImaluumResultPage + job.query
			if err := c.Visit(url); err != nil {
				results <- resultWorkerResult{
					err: errors.ErrFailedToGoToURL,
				}
				return
			}

			response := dtos.ResultResponse{
				ID:           fmt.Sprintf("gomaluum:result:%s", cuid.Slug()),
				SessionName:  job.name,
				SessionQuery: job.query,
				GpaValue:     gpaInfo["gpa"],
				CgpaValue:    gpaInfo["cgpa"],
				CreditHours:  gpaInfo["chr"],
				Status:       gpaInfo["status"],
				Result:       subjects,
			}

			results <- resultWorkerResult{
				result: response,
				err:    nil,
			}
		}()
	}
}

// Process results using worker pool pattern
func (s *Server) processResultsWithWorkerPool(queries, names []string, cookie string) ([]dtos.ResultResponse, error) {
	const maxWorkers = 5

	jobs := make(chan resultJob, len(queries))
	results := make(chan resultWorkerResult, len(queries))

	// Start workers
	for range maxWorkers {
		go s.resultWorker(jobs, results, cookie)
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for i := range queries {
			jobs <- resultJob{
				query: queries[i],
				name:  names[i],
			}
		}
	}()

	// Collect results
	var resultResponses []dtos.ResultResponse
	var errorList []error

	for range queries {
		result := <-results
		if result.err != nil {
			errorList = append(errorList, result.err)
		} else {
			resultResponses = append(resultResponses, result.result)
		}
	}

	if len(errorList) > 0 {
		return nil, errorList[0] // Return first error
	}

	return resultResponses, nil
}

// @Title ResultHandler
// @Description Get result from i-Ma'luum
// @Tags scraper
// @Produce json
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/result [get]
func (s *Server) ResultHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger         = s.log.GetLogger()
		cookie         = r.Context().Value(ctxToken).(string)
		sessionQueries []string
		sessionNames   []string
	)

	// Pre-build cookie string once
	cookieStr := "MOD_AUTH_CAS=" + cookie

	c := colly.NewCollector(
		colly.Headers(
			map[string]string{
				"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
				"Accept-Language": "en-US,en;q=0.9",
				"Cookie":          cookieStr,
			},
		),
		colly.UserAgent(constants.UserAgent),
	)
	c.WithTransport(s.httpClient.Transport)

	c.OnHTML(".box.box-primary .box-header.with-border .dropdown ul.dropdown-menu", func(e *colly.HTMLElement) {
		sessionQueries = e.ChildAttrs("li[style*='font-size:16px'] a", "href")
		sessionNames = e.ChildTexts("li[style*='font-size:16px'] a")
	})

	if err := c.Visit(constants.ImaluumResultPage); err != nil {
		logger.Sugar().Errorf("Failed to go to URL: %v", err)
		errors.Render(w, r, errors.ErrFailedToGoToURL)
		return
	}

	// Filter out unwanted sessions with pre-allocated slices
	filteredQueries := make([]string, 0, len(sessionQueries))
	filteredNames := make([]string, 0, len(sessionNames))

	for i := range sessionQueries {
		if !slices.Contains(UnwantedSessionQueries[:], sessionQueries[i]) {
			filteredQueries = append(filteredQueries, sessionQueries[i])
			filteredNames = append(filteredNames, sessionNames[i])
		}
	}

	if len(filteredQueries) == 0 {
		logger.Sugar().Error("No valid sessions found")
		errors.Render(w, r, errors.ErrResultIsEmpty)
		return
	}

	// Use worker pool for concurrent processing
	results, err := s.processResultsWithWorkerPool(filteredQueries, filteredNames, cookie)
	if err != nil {
		logger.Sugar().Errorf("Failed to process results: %v", err)
		errors.Render(w, r, err)
		return
	}

	if len(results) == 0 {
		logger.Sugar().Error("Result is empty")
		errors.Render(w, r, errors.ErrResultIsEmpty)
		return
	}

	// Sort results
	sort.Slice(results, func(i, j int) bool {
		return utils.SortSessionNames(results[i].SessionName, results[j].SessionName)
	})

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched results",
		Data:    results,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
