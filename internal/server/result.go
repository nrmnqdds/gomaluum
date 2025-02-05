package server

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/nrmnqdds/gomaluum/pkg/utils"
)

// @Title ResultHandler
// @Description Get result from i-Ma'luum
// @Tags scraper
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/result [get]
func (s *Server) ResultHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		cookie         = r.Context().Value(ctxToken).(string)
		c              = colly.NewCollector()
		logger         = s.log.GetLogger()
		wg             sync.WaitGroup
		result         []dtos.ResultResponse
		sessionQueries []string
		sessionNames   []string
		stringBuilder  strings.Builder
	)

	stringBuilder.Grow(100)
	stringBuilder.WriteString("MOD_AUTH_CAS=")
	stringBuilder.WriteString(cookie)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", stringBuilder.String())
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML(".box.box-primary .box-header.with-border .dropdown ul.dropdown-menu", func(e *colly.HTMLElement) {
		sessionQueries = e.ChildAttrs("li[style*='font-size:16px'] a", "href")
		sessionNames = e.ChildTexts("li[style*='font-size:16px'] a")
	})

	if err := c.Visit(constants.ImaluumResultPage); err != nil {
		logger.Error("Failed to go to URL")
		errors.Render(w, errors.ErrFailedToGoToURL)
		return
	}

	// Filter out unwanted session
	filteredQueries := make([]string, 0)
	filteredNames := make([]string, 0)
	for i := range sessionQueries {
		if !slices.Contains(UnwantedSessionQueries[:], sessionQueries[i]) {
			filteredQueries = append(filteredQueries, sessionQueries[i])
			filteredNames = append(filteredNames, sessionNames[i])
		}
	}

	resultChan := make(chan dtos.ResultResponse, len(filteredQueries))
	errChan := make(chan error, len(filteredQueries))

	for i := range filteredQueries {
		wg.Add(1)

		clone := c.Clone()

		go func() {
			defer wg.Done()
			response, err := getResultFromSession(clone, cookie, filteredQueries[i], filteredNames[i])
			if err != nil {
				logger.Sugar().Errorf("Failed to get result from session: %v", err)

				errChan <- err
				return
			}

			resultChan <- *response
		}()
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(resultChan)
	}()

	for err := range errChan {
		if err != nil {
			logger.Error("Failed to get result from session")
			errors.Render(w, err)
			return
		}
	}

	for s := range resultChan {
		result = append(result, s)
	}

	if len(result) == 0 {
		logger.Error("Result is empty")
		errors.Render(w, errors.ErrResultIsEmpty)
		return
	}

	sort.Slice(result, func(i, j int) bool {
		return utils.SortSessionNames(result[i].SessionName, result[j].SessionName)
	})

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched schedule",
		Data:    result,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, errors.ErrFailedToEncodeResponse)
	}
}

func getResultFromSession(c *colly.Collector, cookie string, sessionQuery string, sessionName string) (*dtos.ResultResponse, error) {
	url := constants.ImaluumResultPage + sessionQuery

	var (
		subjects     []dtos.Result
		mu           sync.Mutex
		courseCode   string
		courseName   string
		courseGrade  string
		courseCredit string
		gpa          string
		cgpa         string
		chr          string
		status       string
	)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", "MOD_AUTH_CAS="+cookie)
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML(".box-body table.table.table-hover tbody tr", func(e *colly.HTMLElement) {
		tds := e.ChildTexts("td")

		// Check if tds has enough elements
		if len(tds) < 4 {
			return
		}

		courseCode = strings.TrimSpace(tds[0])

		courseName = strings.TrimSpace(tds[1])

		courseGrade = strings.TrimSpace(tds[2])

		courseCredit = strings.TrimSpace(tds[3])

		words := strings.Fields(strings.TrimSpace(tds[0]))
		if len(words) == 0 {
			return
		}

		if words[0] == "Total" {

			gpaWord := strings.Fields(strings.TrimSpace(tds[1]))

			chr = "0"
			gpa = "0"
			status = "0"
			cgpa = "0"

			if len(gpaWord) > 1 {
				chr = strings.TrimSpace(gpaWord[1])
			}
			if len(gpaWord) > 2 {
				gpa = strings.TrimSpace(gpaWord[2])
			}
			if len(gpaWord) > 3 {
				status = strings.TrimSpace(gpaWord[3])
			}

			cgpaWord := strings.Fields(strings.TrimSpace(tds[3]))
			if len(cgpaWord) > 2 {
				cgpa = strings.TrimSpace(cgpaWord[2])
			}
			return
		}

		mu.Lock()
		subjects = append(subjects, dtos.Result{
			ID:           fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
			CourseCode:   courseCode,
			CourseName:   courseName,
			CourseGrade:  courseGrade,
			CourseCredit: courseCredit,
		})
		mu.Unlock()
	})

	if err := c.Visit(url); err != nil {
		return nil, errors.ErrFailedToGoToURL
	}

	response := &dtos.ResultResponse{
		ID:           fmt.Sprintf("gomaluum:result:%s", cuid.Slug()),
		SessionName:  sessionName,
		SessionQuery: sessionQuery,
		GpaValue:     gpa,
		CgpaValue:    cgpa,
		CreditHours:  chr,
		Status:       status,
		Result:       subjects,
	}

	return response, nil
}
