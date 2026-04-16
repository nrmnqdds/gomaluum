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

var carryMarkTdPool = sync.Pool{
	New: func() any {
		return make([]string, 0, 6)
	},
}

func extractSession(scriptContent string) string {
	const prefix = `console.log("`
	idx := strings.Index(scriptContent, prefix)
	if idx == -1 {
		return ""
	}
	rest := scriptContent[idx+len(prefix):]
	end := strings.Index(rest, `"`)
	if end == -1 {
		return ""
	}
	return rest[:end]
}

// @Title CarryMarkHandler
// @Description Get continuous assessment marks from i-Ma'luum
// @Tags scraper
// @Produce json
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Failure 404 {object} errors.CustomError "No carry mark data found"
// @Router /api/carry-mark [get]
func (s *Server) CarryMarkHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger         = s.log.GetLogger()
		cookie         = r.Context().Value(ctxToken).(string)
		mu             sync.Mutex
		subjects       []dtos.CarryMarkSubject
		currentSubject *dtos.CarryMarkSubject
		session        string
	)

	cookieStr := "MOD_AUTH_CAS=" + cookie

	// NOTE: currentSubject pointer tracking relies on synchronous callback execution.
	// Do NOT add colly.Async() — it would invalidate the pointer after slice reallocation.
	c := colly.NewCollector()
	c.WithTransport(s.httpClient.Transport)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", cookieStr)
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML("script", func(e *colly.HTMLElement) {
		content := strings.TrimSpace(e.Text)
		if strings.Contains(content, "console.log") {
			mu.Lock()
			if session == "" {
				session = extractSession(content)
			}
			mu.Unlock()
		}
	})

	c.OnHTML("table.table.table-hover tbody tr", func(e *colly.HTMLElement) {
		cells := e.DOM.Find("td")
		if cells.Length() < 6 {
			return
		}

		tds := carryMarkTdPool.Get().([]string)
		tds = tds[:0]

		cells.Each(func(_ int, s *goquery.Selection) {
			tds = append(tds, strings.TrimSpace(s.Text()))
		})

		code := tds[0]
		name := tds[2]

		if code != "" {
			subject := dtos.CarryMarkSubject{
				ID:             fmt.Sprintf("gomaluum:carry-mark-subject:%s", cuid.Slug()),
				Code:           code,
				Section:        tds[1],
				Course:         tds[2],
				CreditHour:     tds[3],
				TotalCarryMark: tds[4],
				Components:     []dtos.CarryMarkComponent{},
			}
			mu.Lock()
			subjects = append(subjects, subject)
			currentSubject = &subjects[len(subjects)-1]
			mu.Unlock()
		} else if name != "" && currentSubject != nil {
			component := dtos.CarryMarkComponent{
				ID:           fmt.Sprintf("gomaluum:carry-mark-component:%s", cuid.Slug()),
				Name:         name,
				MarkingScore: tds[3],
				ActualScore:  tds[4],
			}
			mu.Lock()
			currentSubject.Components = append(currentSubject.Components, component)
			mu.Unlock()
		}

		carryMarkTdPool.Put(tds)
	})

	if err := c.Visit(constants.ImaluumCarryMarkPage); err != nil {
		logger.Sugar().Errorf("Failed to go to URL: %v", err)
		errors.Render(w, r, errors.ErrFailedToGoToURL)
		return
	}

	if len(subjects) == 0 {
		logger.Sugar().Error("Carry mark data is empty")
		errors.Render(w, r, errors.ErrNoCarryMark)
		return
	}

	carryMark := &dtos.CarryMark{
		ID:       fmt.Sprintf("gomaluum:carry-mark:%s", cuid.Slug()),
		Session:  session,
		Subjects: subjects,
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched carry marks",
		Data:    carryMark,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
