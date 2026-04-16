package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/bytedance/sonic"
	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

var finalExamItemPool = sync.Pool{
	New: func() any {
		return &dtos.FinalExamItem{}
	},
}

var finalExamTdPool = sync.Pool{
	New: func() any {
		return make([]string, 0, 7)
	},
}

func parseFinalExamRow(tds []string, exams *[]dtos.FinalExamItem, mu *sync.Mutex) {
	if len(tds) < 7 {
		return
	}

	var trimmed [7]string
	for i := range 7 {
		trimmed[i] = strings.TrimSpace(tds[i])
	}

	if trimmed[0] == "" && trimmed[1] == "" {
		return
	}

	item := finalExamItemPool.Get().(*dtos.FinalExamItem)
	*item = dtos.FinalExamItem{
		ID:             fmt.Sprintf("gomaluum:exam:%s", cuid.Slug()),
		SubjectCode:    trimmed[0],
		SubjectName:    trimmed[1],
		SubjectSection: trimmed[2],
		Date:           trimmed[3],
		Time:           trimmed[4],
		Venue:          trimmed[5],
		Seat:           trimmed[6],
	}

	mu.Lock()
	*exams = append(*exams, *item)
	mu.Unlock()

	finalExamItemPool.Put(item)
}

// @Title FinalExamHandler
// @Description Get final exam timetable from i-Ma'luum
// @Tags scraper
// @Produce json
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Failure 404 {object} errors.CustomError "No final exam timetable found"
// @Router /api/exam-timetable [get]
func (s *Server) FinalExamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger = s.log.GetLogger()
		cookie = r.Context().Value(ctxToken).(string)
		mu     sync.Mutex
		exams  []dtos.FinalExamItem
	)

	cookieStr := "MOD_AUTH_CAS=" + cookie

	c := colly.NewCollector()
	c.WithTransport(s.httpClient.Transport)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", cookieStr)
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML("table.table.table-hover tbody tr", func(e *colly.HTMLElement) {
		cells := e.DOM.Find("td")
		if cells.Length() == 0 {
			return
		}

		tds := finalExamTdPool.Get().([]string)
		tds = tds[:0]

		cells.Each(func(_ int, s *goquery.Selection) {
			tds = append(tds, s.Text())
		})

		parseFinalExamRow(tds, &exams, &mu)

		finalExamTdPool.Put(tds)
	})

	if err := c.Visit(constants.ImaluumFinalExamPage); err != nil {
		logger.Sugar().Errorf("Failed to go to URL: %v", err)
		errors.Render(w, r, errors.ErrFailedToGoToURL)
		return
	}

	if len(exams) == 0 {
		logger.Sugar().Error("Final exam timetable is empty")
		errors.Render(w, r, errors.ErrNoFinalExam)
		return
	}

	finalExam := &dtos.FinalExam{
		ID:    fmt.Sprintf("gomaluum:final-exam:%s", cuid.Slug()),
		Exams: exams,
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched final exam timetable",
		Data:    finalExam,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
