package server

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

	_ "time/tzdata"
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

	KLTimezone, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return trimmed, nil
	}

	t, err := time.ParseInLocation("2006-01-02 1504", fmt.Sprintf("%04d-%02d-%02d %s", now.Year(), now.Month(), now.Day(), trimmed), KLTimezone)
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
func (s *Server) scheduleWorker(ctx context.Context, jobs <-chan scheduleJob, results chan<- scheduleResult, cookie string, stale *atomic.Bool) {
	for job := range jobs {
		func() {
			defer utils.CatchPanic("schedule worker")

			c := s.newImaluumCollector(ctx, cookie, stale)

			mu := sync.Mutex{}
			subjects := []dtos.ScheduleSubject{}

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
					err: classifyVisitError(err),
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
func (s *Server) processSchedulesWithWorkerPool(ctx context.Context, queries, names []string, cookie string, stale *atomic.Bool) ([]dtos.ScheduleResponse, error) {
	const maxWorkers = 5

	jobs := make(chan scheduleJob, len(queries))
	results := make(chan scheduleResult, len(queries))

	// Start workers
	for range maxWorkers {
		go s.scheduleWorker(ctx, jobs, results, cookie, stale)
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

// latestSessionQuery returns the query of the newest session in the dropdown.
// Past semesters are immutable, so only this one must be re-scraped on a cache
// hit. queries and names are index-aligned.
func latestSessionQuery(queries, names []string) string {
	latest := 0
	for i := 1; i < len(queries) && i < len(names); i++ {
		if utils.SortSessionNames(names[i], names[latest]) {
			latest = i
		}
	}
	return queries[latest]
}

// resolveSchedules returns the schedules for the given sessions, using the GEI
// cache to avoid re-scraping immutable past semesters. When the cache is
// disabled (indexer nil), forced (refresh), or empty, it scrapes everything.
// Otherwise it re-scrapes only the latest session plus any session missing from
// the cache, and serves the rest from cache. It never writes the cache — the
// handler persists the result only after a fully successful (non-stale) scrape.
func (s *Server) resolveSchedules(ctx context.Context, username string, queries, names []string, cookie string, stale *atomic.Bool, refresh bool) ([]dtos.ScheduleResponse, error) {
	if s.indexer == nil || refresh {
		return s.processSchedulesWithWorkerPool(ctx, queries, names, cookie, stale)
	}

	cached, found, err := s.indexer.GetSchedule(ctx, username)
	if err != nil {
		// A cache read failure must not break the request: fall back to a scrape.
		s.log.WarnContext(ctx, "GEI GetSchedule failed, scraping all sessions", "error", err)
		found = false
	}
	if !found || len(cached) == 0 {
		return s.processSchedulesWithWorkerPool(ctx, queries, names, cookie, stale)
	}

	cachedByQuery := make(map[string]dtos.ScheduleResponse, len(cached))
	for _, c := range cached {
		cachedByQuery[c.SessionQuery] = c
	}

	// Re-scrape the latest session (mutable) plus anything not yet cached.
	latest := latestSessionQuery(queries, names)
	var scrapeQueries, scrapeNames []string
	for i, q := range queries {
		if _, ok := cachedByQuery[q]; q == latest || !ok {
			scrapeQueries = append(scrapeQueries, q)
			scrapeNames = append(scrapeNames, names[i])
		}
	}

	var scraped []dtos.ScheduleResponse
	if len(scrapeQueries) > 0 {
		scraped, err = s.processSchedulesWithWorkerPool(ctx, scrapeQueries, scrapeNames, cookie, stale)
		if err != nil {
			return nil, err
		}
	}
	scrapedByQuery := make(map[string]dtos.ScheduleResponse, len(scraped))
	for _, sc := range scraped {
		scrapedByQuery[sc.SessionQuery] = sc
	}

	// Merge over the authoritative dropdown order: fresh scrape wins, else cache.
	// Iterating over queries drops cached sessions no longer offered by i-Ma'luum.
	merged := make([]dtos.ScheduleResponse, 0, len(queries))
	for _, q := range queries {
		if sc, ok := scrapedByQuery[q]; ok {
			merged = append(merged, sc)
		} else if c, ok := cachedByQuery[q]; ok {
			merged = append(merged, c)
		}
	}
	return merged, nil
}

// @Title ScheduleHandler
// @Description Get schedule from i-Ma'luum
// @Tags scraper
// @Produce json
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/schedule [get]
func (s *Server) ScheduleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger         = s.log
		cookie         = r.Context().Value(ctxToken).(string)
		sessionQueries []string
		sessionNames   []string
		schedules      []dtos.ScheduleResponse
	)

	// username keys the GEI schedule cache; ?refresh=1 (or ?refresh=true) forces
	// a full re-scrape that bypasses the cache read. Any other value is ignored,
	// so ?refresh=0 keeps using the cache.
	var username string
	if sess, ok := r.Context().Value(ctxSession).(*TokenPayload); ok && sess != nil {
		username = sess.username
	}
	refresh := isTruthyParam(r.URL.Query().Get("refresh"))

	// Return fake data for fake user
	if cookie == constants.DebugUserCookie {
		now := time.Now()
		KLTimezone, err := time.LoadLocation("Asia/Kuala_Lumpur")
		if err != nil {
			logger.ErrorContext(r.Context(), "Error loading timezone", "error", err)
			KLTimezone = time.UTC
		}

		morning9am, _ := time.ParseInLocation("2006-01-02 1504", fmt.Sprintf("%04d-%02d-%02d 0900", now.Year(), now.Month(), now.Day()), KLTimezone)
		morning11am, _ := time.ParseInLocation("2006-01-02 1504", fmt.Sprintf("%04d-%02d-%02d 1100", now.Year(), now.Month(), now.Day()), KLTimezone)
		afternoon2pm, _ := time.ParseInLocation("2006-01-02 1504", fmt.Sprintf("%04d-%02d-%02d 1400", now.Year(), now.Month(), now.Day()), KLTimezone)
		afternoon4pm, _ := time.ParseInLocation("2006-01-02 1504", fmt.Sprintf("%04d-%02d-%02d 1600", now.Year(), now.Month(), now.Day()), KLTimezone)
		evening5pm, _ := time.ParseInLocation("2006-01-02 1504", fmt.Sprintf("%04d-%02d-%02d 1700", now.Year(), now.Month(), now.Day()), KLTimezone)
		evening7pm, _ := time.ParseInLocation("2006-01-02 1504", fmt.Sprintf("%04d-%02d-%02d 1900", now.Year(), now.Month(), now.Day()), KLTimezone)

		fakeSchedule := []dtos.ScheduleResponse{
			{
				ID:           fmt.Sprintf("gomaluum:schedule:%s", cuid.Slug()),
				SessionName:  "2024/2025 Semester 1",
				SessionQuery: "?ses=2024/2025&sem=1",
				Schedule: []dtos.ScheduleSubject{
					{
						ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
						CourseCode: "INFO4335",
						CourseName: "Software Engineering",
						Venue:      "E1-LT4",
						Lecturer:   "Dr. Muhammad Ali bin Ahmad",
						Section:    1,
						Chr:        3.0,
						Timestamps: []dtos.WeekTime{
							{
								Start:     "0900",
								StartUnix: morning9am.Unix(),
								End:       "1100",
								EndUnix:   morning11am.Unix(),
								Day:       1,
							},
							{
								Start:     "0900",
								StartUnix: morning9am.Unix(),
								End:       "1100",
								EndUnix:   morning11am.Unix(),
								Day:       3,
							},
						},
					},
					{
						ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
						CourseCode: "INFO4327",
						CourseName: "Database Systems",
						Venue:      "E2-LT2",
						Lecturer:   "Prof. Dr. Siti Aminah binti Abdullah",
						Section:    2,
						Chr:        3.0,
						Timestamps: []dtos.WeekTime{
							{
								Start:     "1400",
								StartUnix: afternoon2pm.Unix(),
								End:       "1600",
								EndUnix:   afternoon4pm.Unix(),
								Day:       2,
							},
							{
								Start:     "1400",
								StartUnix: afternoon2pm.Unix(),
								End:       "1600",
								EndUnix:   afternoon4pm.Unix(),
								Day:       4,
							},
						},
					},
					{
						ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
						CourseCode: "INFO4501",
						CourseName: "Web Development",
						Venue:      "E3-LAB1",
						Lecturer:   "Dr. Ahmad bin Hassan",
						Section:    1,
						Chr:        4.0,
						Timestamps: []dtos.WeekTime{
							{
								Start:     "0900",
								StartUnix: morning9am.Unix(),
								End:       "1100",
								EndUnix:   morning11am.Unix(),
								Day:       2,
							},
							{
								Start:     "1400",
								StartUnix: afternoon2pm.Unix(),
								End:       "1600",
								EndUnix:   afternoon4pm.Unix(),
								Day:       5,
							},
						},
					},
					{
						ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
						CourseCode: "INFO4210",
						CourseName: "Mobile Application Development",
						Venue:      "E1-LAB2",
						Lecturer:   "Dr. Fatimah binti Ibrahim",
						Section:    3,
						Chr:        3.0,
						Timestamps: []dtos.WeekTime{
							{
								Start:     "1700",
								StartUnix: evening5pm.Unix(),
								End:       "1900",
								EndUnix:   evening7pm.Unix(),
								Day:       3,
							},
						},
					},
					{
						ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
						CourseCode: "UNGS2040",
						CourseName: "Tamadun Islam dan Tamadun Asia (TITAS)",
						Venue:      "KAED-LT1",
						Lecturer:   "Dr. Zainab binti Yusof",
						Section:    5,
						Chr:        2.0,
						Timestamps: []dtos.WeekTime{
							{
								Start:     "0900",
								StartUnix: morning9am.Unix(),
								End:       "1100",
								EndUnix:   morning11am.Unix(),
								Day:       5,
							},
						},
					},
				},
			},
			{
				ID:           fmt.Sprintf("gomaluum:schedule:%s", cuid.Slug()),
				SessionName:  "2023/2024 Semester 2",
				SessionQuery: "?ses=2023/2024&sem=2",
				Schedule: []dtos.ScheduleSubject{
					{
						ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
						CourseCode: "INFO3202",
						CourseName: "Data Structures and Algorithms",
						Venue:      "E2-LT3",
						Lecturer:   "Prof. Dr. Abdul Rahman bin Mohd",
						Section:    1,
						Chr:        3.0,
						Timestamps: []dtos.WeekTime{
							{
								Start:     "1400",
								StartUnix: afternoon2pm.Unix(),
								End:       "1600",
								EndUnix:   afternoon4pm.Unix(),
								Day:       1,
							},
							{
								Start:     "1400",
								StartUnix: afternoon2pm.Unix(),
								End:       "1600",
								EndUnix:   afternoon4pm.Unix(),
								Day:       3,
							},
						},
					},
					{
						ID:         fmt.Sprintf("gomaluum:subject:%s", cuid.Slug()),
						CourseCode: "INFO3150",
						CourseName: "Computer Networks",
						Venue:      "E3-LT1",
						Lecturer:   "Dr. Nurul Huda binti Hassan",
						Section:    2,
						Chr:        3.0,
						Timestamps: []dtos.WeekTime{
							{
								Start:     "0900",
								StartUnix: morning9am.Unix(),
								End:       "1100",
								EndUnix:   morning11am.Unix(),
								Day:       4,
							},
						},
					},
				},
			},
		}

		response := &dtos.ResponseDTO{
			Message: "Successfully fetched schedule",
			Data:    fakeSchedule,
		}

		if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
			logger.ErrorContext(r.Context(), "Failed to encode response", "error", err)
			errors.Render(w, r, errors.ErrFailedToEncodeResponse)
		}
		return
	}

	if err := s.scrapeWithRetry(r.Context(), func(cookie string) (bool, error) {
		var stale atomic.Bool
		sessionQueries = sessionQueries[:0]
		sessionNames = sessionNames[:0]

		c := s.newImaluumCollector(r.Context(), cookie, &stale)
		c.OnHTML(".box.box-primary .box-header.with-border .dropdown ul.dropdown-menu", func(e *colly.HTMLElement) {
			sessionQueries = e.ChildAttrs("li[style*='font-size:16px'] a", "href")
			sessionNames = e.ChildTexts("li[style*='font-size:16px'] a")
		})
		if err := c.Visit(constants.ImaluumSchedulePage); err != nil {
			return false, classifyVisitError(err)
		}
		if stale.Load() {
			return true, nil
		}

		filteredQueries := make([]string, 0, len(sessionQueries))
		filteredNames := make([]string, 0, len(sessionNames))
		for i := range sessionQueries {
			if !slices.Contains(UnwantedSessionQueries[:], sessionQueries[i]) {
				filteredQueries = append(filteredQueries, sessionQueries[i])
				filteredNames = append(filteredNames, sessionNames[i])
			}
		}
		if len(filteredQueries) == 0 {
			logger.ErrorContext(r.Context(), "No valid sessions found")
			return false, errors.ErrScheduleIsEmpty
		}

		result, err := s.resolveSchedules(r.Context(), username, filteredQueries, filteredNames, cookie, &stale, refresh)
		if err != nil {
			return false, err
		}
		if stale.Load() {
			return true, nil
		}
		schedules = result
		return false, nil
	}); err != nil {
		logger.ErrorContext(r.Context(), "Failed to scrape schedule", "error", err)
		errors.Render(w, r, err)
		return
	}

	if len(schedules) == 0 {
		logger.ErrorContext(r.Context(), "Schedule is empty")
		errors.Render(w, r, errors.ErrScheduleIsEmpty)
		return
	}

	// Sort schedules
	sort.Slice(schedules, func(i, j int) bool {
		return utils.SortSessionNames(schedules[i].SessionName, schedules[j].SessionName)
	})

	// Refresh the cache with the successful result (best-effort). Only reached on
	// a non-stale scrape, so the login page can never poison the cache.
	if s.indexer != nil && username != "" {
		if err := s.indexer.StoreSchedule(r.Context(), username, schedules); err != nil {
			logger.WarnContext(r.Context(), "Failed to cache schedule in GEI", "error", err)
		}
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched schedule",
		Data:    schedules,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(r.Context(), "Failed to encode response", "error", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
