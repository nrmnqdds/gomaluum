package server

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bytedance/sonic"
	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/nrmnqdds/gomaluum/pkg/utils"
	"github.com/rung/go-safecast"
)

var UnwantedSessionQueries = [...]string{
	"?ses=1111/1111&sem=1",
	"?ses=0000/0000&sem=0",
}

// Pre-map day conversions for better performance
var dayMap = map[string][]string{
	"MTW":    {"M", "T", "W"},
	"TWTH":   {"T", "W", "TH"},
	"MTWTH":  {"M", "T", "W", "TH"},
	"MTWTHF": {"M", "T", "W", "TH", "F"},
}

// Pre-compiled regex for time parsing
var timePattern = regexp.MustCompile(`^\d{3,4}-\d{3,4}$`)

// Object pools for memory reuse
var subjectPool = sync.Pool{
	New: func() any {
		return &dtos.ScheduleSubject{}
	},
}

var weekTimeSlicePool = sync.Pool{
	New: func() any {
		return make([]dtos.WeekTime, 0, 5)
	},
}

var stringSlicePool = sync.Pool{
	New: func() any {
		return make([]string, 0, 10)
	},
}

// Worker pool structures
type scheduleJob struct {
	query string
	name  string
}

type scheduleResult struct {
	err      error
	schedule dtos.ScheduleResponse
}

// Fast day parsing using pre-built map
func parseDays(dayStr string) []string {
	cleaned := strings.ReplaceAll(dayStr, " ", "")
	if mapped, exists := dayMap[cleaned]; exists {
		return mapped
	}
	return strings.Split(cleaned, "-")
}

// Normalize time format efficiently
func normalizeTime(timeStr string) (string, *int64) {
	trimmed := strings.TrimSpace(timeStr)

	if len(trimmed) == 3 {
		trimmed = fmt.Sprintf("0%s", trimmed) // Pad single-digit times
	}

	now := time.Now()

	// KLTimezone, err := time.LoadLocation("Local")
	// if err != nil {
	// 	fmt.Println("Error parsing time:", err)
	// 	return trimmed, nil
	// }

	t, err := time.Parse("2006-01-02 1504", fmt.Sprintf("%04d-%02d-%02d %s", now.Year(), now.Month(), now.Day(), trimmed))
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return trimmed, nil
	}

	unixTimestamp := t.Unix()

	return trimmed, &unixTimestamp
}

// Parse table row with object pooling
func parseTableRow(tds []string, subjects *[]dtos.ScheduleSubject, mu *sync.Mutex) {
	if len(tds) == 0 {
		return
	}

	weekTimeSlice := weekTimeSlicePool.Get().([]dtos.WeekTime)
	weekTimeSlice = weekTimeSlice[:0] // Reset slice

	var subject *dtos.ScheduleSubject

	// Handle perfect cell (9 columns)
	if len(tds) == 9 {
		subject = subjectPool.Get().(*dtos.ScheduleSubject)
		*subject = dtos.ScheduleSubject{} // Reset

		subject.CourseCode = strings.TrimSpace(tds[0])
		subject.CourseName = strings.TrimSpace(tds[1])

		section, err := safecast.Atoi32(strings.TrimSpace(tds[2]))
		if err != nil {
			subjectPool.Put(subject)
			weekTimeSlicePool.Put(weekTimeSlice)
			return
		}
		subject.Section = uint32(section)

		chr, err := strconv.ParseFloat(strings.TrimSpace(tds[3]), 32)
		if err != nil {
			subjectPool.Put(subject)
			weekTimeSlicePool.Put(weekTimeSlice)
			return
		}
		subject.Chr = chr

		// Parse days and times
		days := parseDays(strings.TrimSpace(tds[5]))
		timeFullForm := strings.ReplaceAll(strings.TrimSpace(tds[6]), " ", "")

		if timeFullForm != constants.TimeSeparator && timePattern.MatchString(timeFullForm) {
			timeParts := strings.Split(timeFullForm, constants.TimeSeparator)
			if len(timeParts) == 2 {
				start, startUnix := normalizeTime(timeParts[0])
				end, endUnix := normalizeTime(timeParts[1])

				for _, day := range days {
					dayNum := utils.GetScheduleDays(day)
					weekTimeSlice = append(weekTimeSlice, dtos.WeekTime{
						Start:     start,
						StartUnix: *startUnix,
						End:       end,
						EndUnix:   *endUnix,
						Day:       dayNum,
					})
				}
			}
		}

		subject.Venue = strings.TrimSpace(tds[7])
		subject.Lecturer = strings.TrimSpace(tds[8])
	}

	// Handle merged cell (4 columns)
	if len(tds) == 4 {
		mu.Lock()
		if len(*subjects) == 0 {
			mu.Unlock()
			weekTimeSlicePool.Put(weekTimeSlice)
			return
		}
		lastSubject := (*subjects)[len(*subjects)-1]
		mu.Unlock()

		subject = subjectPool.Get().(*dtos.ScheduleSubject)
		*subject = dtos.ScheduleSubject{} // Reset

		subject.CourseCode = lastSubject.CourseCode
		subject.CourseName = lastSubject.CourseName
		subject.Section = lastSubject.Section
		subject.Chr = lastSubject.Chr

		// Parse days and times
		days := parseDays(strings.TrimSpace(tds[0]))
		timeFullForm := strings.ReplaceAll(strings.TrimSpace(tds[1]), " ", "")

		if timePattern.MatchString(timeFullForm) {
			timeParts := strings.Split(timeFullForm, "-")
			if len(timeParts) == 2 {
				start, startUnix := normalizeTime(timeParts[0])
				end, endUnix := normalizeTime(timeParts[1])

				for _, day := range days {
					dayNum := utils.GetScheduleDays(day)
					weekTimeSlice = append(weekTimeSlice, dtos.WeekTime{
						Start:     start,
						StartUnix: *startUnix,
						End:       end,
						EndUnix:   *endUnix,
						Day:       dayNum,
					})
				}
			}
		}

		subject.Venue = strings.TrimSpace(tds[2])
		subject.Lecturer = strings.TrimSpace(tds[3])
	}

	if subject != nil {
		// Copy weekTime slice to avoid pool contamination
		subject.Timestamps = make([]dtos.WeekTime, len(weekTimeSlice))
		copy(subject.Timestamps, weekTimeSlice)
		subject.ID = fmt.Sprintf("gomaluum:subject:%s", cuid.Slug())

		mu.Lock()
		*subjects = append(*subjects, *subject)
		mu.Unlock()

		subjectPool.Put(subject)
	}

	weekTimeSlicePool.Put(weekTimeSlice)
}

// Worker function for processing schedule sessions
func (s *Server) scheduleWorker(jobs <-chan scheduleJob, results chan<- scheduleResult, cookie string) {
	cookieStr := "MOD_AUTH_CAS=" + cookie

	for job := range jobs {
		func() {
			defer utils.CatchPanic("schedule worker")

			c := colly.NewCollector()
			c.WithTransport(s.httpClient.Transport)

			var (
				mu       sync.Mutex
				subjects []dtos.ScheduleSubject
			)

			c.OnRequest(func(r *colly.Request) {
				r.Headers.Set("Cookie", cookieStr)
				r.Headers.Set("User-Agent", cuid.New())
			})

			c.OnHTML("table.table-hover tbody tr", func(e *colly.HTMLElement) {
				// Get all text at once with efficient DOM traversal
				cells := e.DOM.Find("td")
				if cells.Length() == 0 {
					return
				}

				tds := stringSlicePool.Get().([]string)
				tds = tds[:0] // Reset slice

				cells.Each(func(_ int, s *goquery.Selection) {
					tds = append(tds, s.Text())
				})

				parseTableRow(tds, &subjects, &mu)
				stringSlicePool.Put(tds)
			})

			url := constants.ImaluumSchedulePage + job.query
			if err := c.Visit(url); err != nil {
				results <- scheduleResult{
					err: errors.ErrFailedToGoToURL,
				}
				return
			}

			response := dtos.ScheduleResponse{
				ID:           fmt.Sprintf("gomaluum:schedule:%s", cuid.Slug()),
				SessionName:  job.name,
				SessionQuery: job.query,
				Schedule:     subjects,
			}

			results <- scheduleResult{
				schedule: response,
				err:      nil,
			}
		}()
	}
}

// Process schedules using worker pool pattern
func (s *Server) processSchedulesWithWorkerPool(queries, names []string, cookie string) ([]dtos.ScheduleResponse, error) {
	const maxWorkers = 5

	jobs := make(chan scheduleJob, len(queries))
	results := make(chan scheduleResult, len(queries))

	// Start workers
	for range maxWorkers {
		go s.scheduleWorker(jobs, results, cookie)
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for i := range queries {
			jobs <- scheduleJob{
				query: queries[i],
				name:  names[i],
			}
		}
	}()

	// Collect results
	var schedules []dtos.ScheduleResponse
	var errors []error

	for range queries {
		result := <-results
		if result.err != nil {
			errors = append(errors, result.err)
		} else {
			schedules = append(schedules, result.schedule)
		}
	}

	if len(errors) > 0 {
		return nil, errors[0] // Return first error
	}

	return schedules, nil
}

// @Title ScheduleHandler
// @Description Get schedule from i-Ma'luum
// @Tags scraper
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/schedule [get]
func (s *Server) ScheduleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger         = s.log.GetLogger()
		cookie         = r.Context().Value(ctxToken).(string)
		sessionQueries []string
		sessionNames   []string
	)

	// Pre-build cookie string once
	cookieStr := "MOD_AUTH_CAS=" + cookie

	c := colly.NewCollector()
	c.WithTransport(s.httpClient.Transport)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", cookieStr)
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML(".box.box-primary .box-header.with-border .dropdown ul.dropdown-menu", func(e *colly.HTMLElement) {
		sessionQueries = e.ChildAttrs("li[style*='font-size:16px'] a", "href")
		sessionNames = e.ChildTexts("li[style*='font-size:16px'] a")
	})

	if err := c.Visit(constants.ImaluumSchedulePage); err != nil {
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
		errors.Render(w, r, errors.ErrScheduleIsEmpty)
		return
	}

	// Use worker pool for concurrent processing
	schedules, err := s.processSchedulesWithWorkerPool(filteredQueries, filteredNames, cookie)
	if err != nil {
		logger.Sugar().Errorf("Failed to process schedules: %v", err)
		errors.Render(w, r, err)
		return
	}

	if len(schedules) == 0 {
		logger.Sugar().Error("Schedule is empty")
		errors.Render(w, r, errors.ErrScheduleIsEmpty)
		return
	}

	// Sort schedules
	sort.Slice(schedules, func(i, j int) bool {
		return utils.SortSessionNames(schedules[i].SessionName, schedules[j].SessionName)
	})

	logger.Sugar().Infof("Schedule response %v: ", schedules)
	response := &dtos.ResponseDTO{
		Message: "Successfully fetched schedule",
		Data:    schedules,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
