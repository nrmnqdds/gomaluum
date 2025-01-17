package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/nrmnqdds/gomaluum/pkg/utils"
	"github.com/rung/go-safecast"
)

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

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", stringBuilder.String())
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML(".box.box-primary .box-header.with-border .dropdown ul.dropdown-menu", func(e *colly.HTMLElement) {
		sessionQueries = e.ChildAttrs("li[style*='font-size:16px'] a", "href")
		sessionNames = e.ChildTexts("li[style*='font-size:16px'] a")
	})

	scheduleChan := make(chan dtos.ScheduleResponse, len(sessionQueries))

	if err := c.Visit(constants.ImaluumSchedulePage); err != nil {
		logger.Sugar().Error("Failed to go to URL")
		errors.Render(w, errors.ErrFailedToGoToURL)
		return
	}

	for i := range sessionQueries {
		wg.Add(1)

		clone := c.Clone()

		go func() {
			defer wg.Done()
			response, err := getScheduleFromSession(clone, cookie, sessionQueries[i], sessionNames[i])
			if err != nil {
				logger.Sugar().Errorf("Failed to get schedule from session: %v", err)
				return
			}
			scheduleChan <- *response
		}()
	}

	go func() {
		wg.Wait()
		close(scheduleChan)
	}()

	for s := range scheduleChan {
		schedule = append(schedule, s)
	}

	if len(schedule) == 0 {
		logger.Error("Schedule is empty")
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

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, errors.ErrFailedToEncodeResponse)
	}
}

func getScheduleFromSession(c *colly.Collector, cookie string, sessionQuery string, sessionName string) (*dtos.ScheduleResponse, error) {
	var (
		mu            sync.Mutex
		subjects      = []dtos.ScheduleSubject{}
		stringBuilder strings.Builder
	)

	stringBuilder.Grow(20)
	stringBuilder.WriteString(constants.ImaluumSchedulePage)
	stringBuilder.WriteString(sessionQuery)

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
				ID:         cuid.New(),
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

	if err := c.Visit(stringBuilder.String()); err != nil {
		return nil, err
	}

	response := &dtos.ScheduleResponse{
		ID:           fmt.Sprintf("gomaluum:schedule:%s", cuid.Slug()),
		SessionName:  sessionName,
		SessionQuery: sessionQuery,
		Schedule:     subjects,
	}

	return response, nil
}
