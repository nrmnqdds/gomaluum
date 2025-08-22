package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/bytedance/sonic"
	"github.com/gocolly/colly/v2"
	"github.com/lucsky/cuid"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/rung/go-safecast"
)

// Object pools for memory reuse
var programPool = sync.Pool{
	New: func() any {
		return &dtos.StarpointProgram{}
	},
}

var programTdStringSlicePool = sync.Pool{
	New: func() any {
		return make([]string, 0, 10)
	},
}

// Parse table row with object pooling
func parseProgramRows(tds []string, programs *[]dtos.StarpointProgram, mu *sync.Mutex) {
	if len(tds) == 0 {
		return
	}

	var program *dtos.StarpointProgram

	// Handle perfect cell (6 columns)
	if len(tds) == 6 {
		program = programPool.Get().(*dtos.StarpointProgram)
		*program = dtos.StarpointProgram{} // Reset

		section, err := safecast.Atoi8(strings.TrimSpace(tds[0]))
		if err != nil {
			programPool.Put(program)
			return
		}

		program.Semester = uint8(section)
		program.Session = strings.TrimSpace(tds[1])
		program.EventName = strings.TrimSpace(tds[2])
		program.Type = strings.TrimSpace(tds[3])
		program.Level = strings.TrimSpace(tds[4])

		points, err := strconv.ParseFloat(strings.TrimSpace(tds[5]), 32)
		if err != nil {
			programPool.Put(program)
			return
		}
		program.Points = float32(points)
	}

	if program != nil {
		program.ID = fmt.Sprintf("gomaluum:program:%s", cuid.Slug())

		mu.Lock()
		*programs = append(*programs, *program)
		mu.Unlock()

		programPool.Put(program)
	}
}

func getFloatFromString(s string) float64 {
	ca := strings.TrimSpace(strings.Split(s, ":")[1])

	points, err := strconv.ParseFloat(ca, 64)
	if err != nil {
		return 0
	}

	return points
}

// @Title StarpointHandler
// @Description Get co-curricular from i-Ma'luum
// @Tags scraper
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/starpoint [get]
func (s *Server) StarpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger    = s.log.GetLogger()
		cookie    = r.Context().Value(ctxToken).(string)
		mu        sync.Mutex
		programs  []dtos.StarpointProgram
		starpoint = &dtos.Starpoint{}
	)

	// Pre-build cookie string once
	cookieStr := "MOD_AUTH_CAS=" + cookie

	c := colly.NewCollector()
	c.WithTransport(s.httpClient.Transport)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", cookieStr)
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML("table.table.table-hover tbody tr", func(e *colly.HTMLElement) {
		// Get all text at once with efficient DOM traversal
		cells := e.DOM.Find("td")
		if cells.Length() == 0 {
			return
		}

		tds := programTdStringSlicePool.Get().([]string)
		tds = tds[:0] // Reset slice

		cells.Each(func(_ int, s *goquery.Selection) {
			if strings.TrimSpace(strings.Split(s.Text(), ":")[0]) == "Cummulative Average" {
				// Special case for Cummulative Average row
				if starpoint.CummulativeAverage != 0 {
					logger.Sugar().Warn("Cummulative Average already set, skipping duplicate")
					logger.Sugar().Debugf("Current value: %f, new value: %s", starpoint.CummulativeAverage, s.Text())
					return
				}
				starpoint.CummulativeAverage = getFloatFromString(s.Text())
				return
			}

			if strings.TrimSpace(strings.Split(s.Text(), ":")[0]) == "Total Point" {
				// Special case for Total Point row
				if starpoint.TotalPoints != 0 {
					logger.Sugar().Warn("Total Point already set, skipping duplicate")
					logger.Sugar().Debugf("Current value: %f, new value: %s", starpoint.CummulativeAverage, s.Text())
					return
				}
				starpoint.TotalPoints = getFloatFromString(s.Text())
				return
			}
			tds = append(tds, s.Text())
		})
		parseProgramRows(tds, &programs, &mu)

		programTdStringSlicePool.Put(tds)
	})

	if err := c.Visit(constants.ImaluumStarpointPage); err != nil {
		logger.Sugar().Errorf("Failed to go to URL: %v", err)
		errors.Render(w, r, errors.ErrFailedToGoToURL)
		return
	}

	if len(programs) == 0 {
		logger.Sugar().Error("Program is empty")
		errors.Render(w, r, errors.ErrNoStarpoint)
		return
	}

	// Set starpoint data
	starpoint.Programs = programs
	starpoint.ID = fmt.Sprintf("gomaluum:starpoint:%s", cuid.Slug())

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched starpoints programs",
		Data:    starpoint,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
