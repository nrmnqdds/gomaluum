package server

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/nrmnqdds/gomaluum/pkg/utils"
	"github.com/rung/go-safecast"
	"go.uber.org/zap"
)

var UnwantedSessionQueries = [...]string{
	"?ses=1111/1111&sem=1",
	"?ses=0000/0000&sem=0",
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
		c              = colly.NewCollector()
		wg             sync.WaitGroup
		schedule       []dtos.ScheduleResponse
		sessionQueries []string
		sessionNames   []string
		stringBuilder  strings.Builder
	)

	stringBuilder.Grow(100)
	stringBuilder.WriteString("MOD_AUTH_CAS=")
	stringBuilder.WriteString(cookie)

	httpClient, err := CreateHTTPClient()
	if err != nil {
		logger.Sugar().Errorf("Failed to create HTTP client: %v", err)
		errors.Render(w, errors.ErrFailedToCreateHTTPClient)
		return
	}

	c.WithTransport(httpClient.Transport)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", stringBuilder.String())
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML(".box.box-primary .box-header.with-border .dropdown ul.dropdown-menu", func(e *colly.HTMLElement) {
		sessionQueries = e.ChildAttrs("li[style*='font-size:16px'] a", "href")
		sessionNames = e.ChildTexts("li[style*='font-size:16px'] a")
	})

	if err := c.Visit(constants.ImaluumSchedulePage); err != nil {
		logger.Sugar().Errorf("Failed to go to URL: %v", err)
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

	scheduleChan := make(chan dtos.ScheduleResponse, len(filteredQueries))
	errChan := make(chan error, len(filteredQueries))

	for i := range filteredQueries {
		wg.Add(1)

		clone := c.Clone()

		go func() {
			defer utils.CatchPanic("get schedule from session")
			defer wg.Done()

			response, err := getScheduleFromSession(clone, cookie, filteredQueries[i], filteredNames[i], logger)
			if err != nil {
				logger.Sugar().Errorf("Failed to get schedule from session: %v", err)

				errChan <- err
				return
			}

			scheduleChan <- *response
		}()
	}

	go func() {
		defer utils.CatchPanic("schedule close channel")
		wg.Wait()
		close(errChan)
		close(scheduleChan)
	}()

	for err := range errChan {
		if err != nil {
			logger.Sugar().Errorf("Failed to get schedule from session: %v", err)
			errors.Render(w, err)
			return
		}
	}

	for s := range scheduleChan {
		schedule = append(schedule, s)
	}

	if len(schedule) == 0 {
		logger.Sugar().Error("Schedule is empty")
		errors.Render(w, errors.ErrScheduleIsEmpty)
		return
	}

	sort.Slice(schedule, func(i, j int) bool {
		return utils.SortSessionNames(schedule[i].SessionName, schedule[j].SessionName)
	})

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched schedule",
		Data:    schedule,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, errors.ErrFailedToEncodeResponse)
	}
}

func getScheduleFromSession(c *colly.Collector, cookie string, sessionQuery string, sessionName string, logger *zap.Logger) (*dtos.ScheduleResponse, error) {
	url := constants.ImaluumSchedulePage + sessionQuery

	var (
		mu       sync.Mutex
		subjects = []dtos.ScheduleSubject{}
	)

	httpClient, err := CreateHTTPClient()
	if err != nil {
		logger.Sugar().Errorf("Failed to create HTTP client: %v", err)
		return nil, errors.ErrFailedToCreateHTTPClient
	}

	c.WithTransport(httpClient.Transport)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", "MOD_AUTH_CAS="+cookie)
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML(".box-body table.table.table-hover tr", func(e *colly.HTMLElement) {
		tds := e.ChildTexts("td")

		weekTime := []dtos.WeekTime{}

		if len(tds) == 0 {
			// Skip the first row
			return
		}

		// Handles for perfect cell
		if len(tds) == 9 {
			courseCode := strings.TrimSpace(tds[0])
			courseName := strings.TrimSpace(tds[1])

			section, err := safecast.Atoi32(strings.TrimSpace(tds[2]))
			if err != nil {
				return
			}

			chr, err := strconv.ParseFloat(strings.TrimSpace(tds[3]), 32)
			if err != nil {
				return
			}

			// Split the days
			_days := strings.Split(strings.Replace(strings.TrimSpace(tds[5]), " ", "", -1), "-")

			// Handles weird ass day format
			switch _days[0] {
			case "MTW":
				_days = []string{"M", "T", "W"}
			case "TWTH":
				_days = []string{"T", "W", "TH"}
			case "MTWTH":
				_days = []string{"M", "T", "W", "TH"}
			case "MTWTHF":
				_days = []string{"M", "T", "W", "TH", "F"}
			}

			for _, day := range _days {
				dayNum := utils.GetScheduleDays(day)
				timeTemp := tds[6]

				// `timeFullForm` refers to schedule time from iMaluum
				// e.g.: 800-920 or 1000-1120
				timeFullForm := strings.Replace(strings.TrimSpace(timeTemp), " ", "", -1)

				// in some cases, iMaluum will return "-" as time
				// if `timeFullForm` equals `TimeSeparator`, then we skip this row
				if timeFullForm == constants.TimeSeparator {
					continue
				}

				// safely split time entry
				time := strings.Split(timeFullForm, constants.TimeSeparator)

				start := strings.TrimSpace(time[0])
				end := strings.TrimSpace(time[1])

				if len(start) == 3 {
					start = "0" + start
				}

				if len(end) == 3 {
					end = "0" + end
				}

				weekTime = append(weekTime, dtos.WeekTime{
					Start: start,
					End:   end,
					Day:   dayNum,
				})
			}

			venue := strings.TrimSpace(tds[7])
			lecturer := strings.TrimSpace(tds[8])

			mu.Lock()
			subjects = append(subjects, dtos.ScheduleSubject{
				ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
				CourseCode: courseCode,
				CourseName: courseName,
				Section:    uint32(section),
				Chr:        chr,
				Timestamps: weekTime,
				Venue:      venue,
				Lecturer:   lecturer,
			})
			mu.Unlock()

		}

		// Handles for merged cell usually at time or day or venue
		if len(tds) == 4 {
			mu.Lock()
			lastSubject := subjects[len(subjects)-1]
			mu.Unlock()
			courseCode := lastSubject.CourseCode
			courseName := lastSubject.CourseName
			section := lastSubject.Section
			chr := lastSubject.Chr

			// Split the days
			_days := strings.Split(strings.Replace(strings.TrimSpace(tds[0]), " ", "", -1), "-")

			// Handles weird ass day format
			switch _days[0] {
			case "MTW":
				_days = []string{"M", "T", "W"}
			case "TWTH":
				_days = []string{"T", "W", "TH"}
			case "MTWTH":
				_days = []string{"M", "T", "W", "TH"}
			case "MTWTHF":
				_days = []string{"M", "T", "W", "TH", "F"}
			}

			for _, day := range _days {
				dayNum := utils.GetScheduleDays(day)
				timeTemp := tds[1]
				time := strings.Split(strings.Replace(strings.TrimSpace(timeTemp), " ", "", -1), "-")

				if len(time) != 2 {
					continue
				}

				start := strings.TrimSpace(time[0])
				end := strings.TrimSpace(time[1])

				if len(start) == 3 {
					start = "0" + start
				}
				if len(end) == 3 {
					end = "0" + end
				}

				weekTime = append(weekTime, dtos.WeekTime{
					Start: start,
					End:   end,
					Day:   dayNum,
				})
			}

			venue := strings.TrimSpace(tds[2])
			lecturer := strings.TrimSpace(tds[3])

			mu.Lock()
			subjects = append(subjects, dtos.ScheduleSubject{
				ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
				CourseCode: courseCode,
				CourseName: courseName,
				Section:    section,
				Chr:        chr,
				Timestamps: weekTime,
				Venue:      venue,
				Lecturer:   lecturer,
			})
			mu.Unlock()
		}
	})

	if err := c.Visit(url); err != nil {
		logger.Sugar().Errorf("Failed to go to URL: %v", err)
		return nil, errors.ErrFailedToGoToURL
	}

	response := &dtos.ScheduleResponse{
		ID:           fmt.Sprintf("gomaluum:schedule:%s", cuid.Slug()),
		SessionName:  sessionName,
		SessionQuery: sessionQuery,
		Schedule:     subjects,
	}

	return response, nil
}
