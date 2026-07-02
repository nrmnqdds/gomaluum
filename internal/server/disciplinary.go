package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/PuerkitoBio/goquery"
	"github.com/bytedance/sonic"
	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

var disciplinaryTdPool = sync.Pool{
	New: func() any {
		return make([]string, 0, 9)
	},
}

func parseDisciplinaryRow(tds []string, actionCell *goquery.Selection, compounds *[]dtos.DisciplinaryCompound, mu *sync.Mutex) {
	if len(tds) < 8 {
		return
	}

	var trimmed [8]string
	for i := range 8 {
		trimmed[i] = strings.TrimSpace(tds[i])
	}

	// Skip empty rows
	if trimmed[0] == "" && trimmed[2] == "" {
		return
	}

	var links []dtos.DisciplinaryCompoundLink
	if actionCell != nil {
		actionCell.Find("a").Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			title := strings.TrimSpace(a.Text())
			if href != "" {
				links = append(links, dtos.DisciplinaryCompoundLink{
					Title: title,
					URL:   href,
				})
			}
		})
	}

	compound := dtos.DisciplinaryCompound{
		ID:          fmt.Sprintf("gomaluum:compound:%s", cuid.Slug()),
		Session:     trimmed[0],
		OffenceDate: trimmed[1],
		CompoundNo:  trimmed[2],
		Description: trimmed[3],
		Agency:      trimmed[4],
		Status:      trimmed[5],
		Fine:        trimmed[6],
		DueDate:     trimmed[7],
		Links:       links,
	}

	mu.Lock()
	*compounds = append(*compounds, compound)
	mu.Unlock()
}

// @Title DisciplinaryHandler
// @Description Get compound and traffic summon records from i-Ma'luum
// @Tags scraper
// @Produce json
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Failure 404 {object} errors.CustomError "No disciplinary or compound records found"
// @Router /api/disciplinary [get]
func (s *Server) DisciplinaryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger    = s.log
		mu        sync.Mutex
		compounds []dtos.DisciplinaryCompound
	)

	if err := s.scrapeWithRetry(r.Context(), func(cookie string) (bool, error) {
		// Reset accumulators so a retry starts clean.
		mu.Lock()
		compounds = compounds[:0]
		mu.Unlock()

		var stale atomic.Bool
		c := s.newImaluumCollector(r.Context(), cookie, &stale)

		c.OnHTML("table.table.table-hover tbody tr", func(e *colly.HTMLElement) {
			cells := e.DOM.Find("td")
			if cells.Length() < 8 {
				return
			}

			tds := disciplinaryTdPool.Get().([]string)
			tds = tds[:0]
			defer disciplinaryTdPool.Put(tds)

			var actionCell *goquery.Selection
			cells.Each(func(i int, s *goquery.Selection) {
				if i < 8 {
					tds = append(tds, s.Text())
				} else if i == 8 {
					actionCell = s
				}
			})

			parseDisciplinaryRow(tds, actionCell, &compounds, &mu)
		})

		if err := c.Visit(constants.ImaluumDisciplinaryPage); err != nil {
			return false, classifyVisitError(err)
		}
		return stale.Load(), nil
	}); err != nil {
		logger.ErrorContext(r.Context(), "Failed to scrape disciplinary records", "error", err)
		errors.Render(w, r, err)
		return
	}

	if len(compounds) == 0 {
		logger.ErrorContext(r.Context(), "No disciplinary or compound records found")
		errors.Render(w, r, errors.ErrNoDisciplinaryRecord)
		return
	}

	disciplinary := &dtos.Disciplinary{
		ID:        fmt.Sprintf("gomaluum:disciplinary:%s", cuid.Slug()),
		Compounds: compounds,
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched disciplinary records",
		Data:    disciplinary,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(r.Context(), "Failed to encode response", "error", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
