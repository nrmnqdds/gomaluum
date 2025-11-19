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
	"go.uber.org/zap"
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
func parseProgramRows(tds []string, programs *[]dtos.StarpointProgram, mu *sync.Mutex, lastSession *string, logger *zap.Logger) {
	if len(tds) != 6 {
		return
	}

	var program *dtos.StarpointProgram

	// Trim all cells
	trimmedTds := make([]string, 6)
	for i, td := range tds {
		trimmedTds[i] = strings.TrimSpace(td)
	}

	// Skip empty rows or header rows
	if trimmedTds[2] == "" || strings.HasPrefix(trimmedTds[0], "PROGRAMMES") {
		return
	}

	logger.Sugar().Debugf("Processing row: %v", trimmedTds)

	program = programPool.Get().(*dtos.StarpointProgram)
	*program = dtos.StarpointProgram{} // Reset

	// Check if this is a new session row (has semester number and session)
	if trimmedTds[0] != "" && trimmedTds[1] != "" {
		// Full row with semester and session
		program.Session = trimmedTds[1]
		program.EventName = trimmedTds[2]
		if trimmedTds[3] != "" {
			typeVal := trimmedTds[3]
			program.Type = &typeVal
		} else {
			program.Type = nil
		}
		program.Level = trimmedTds[4]

		points, err := strconv.ParseFloat(trimmedTds[5], 32)
		if err != nil {
			logger.Sugar().Warnf("Failed to parse points '%s': %v", trimmedTds[5], err)
			programPool.Put(program)
			return
		}
		program.Points = float32(points)

		// Update last session for continuation rows
		*lastSession = program.Session
		logger.Sugar().Debugf("New session row: %s - %s", program.Session, program.EventName)
	} else if trimmedTds[1] != "" && trimmedTds[0] == "" {
		// Row with session but no semester (new session group, continuation)
		program.Session = trimmedTds[1]
		program.EventName = trimmedTds[2]
		if trimmedTds[3] != "" {
			typeVal := trimmedTds[3]
			program.Type = &typeVal
		} else {
			program.Type = nil
		}
		program.Level = trimmedTds[4]

		points, err := strconv.ParseFloat(trimmedTds[5], 32)
		if err != nil {
			logger.Sugar().Warnf("Failed to parse points '%s': %v", trimmedTds[5], err)
			programPool.Put(program)
			return
		}
		program.Points = float32(points)

		// Update last session
		*lastSession = program.Session
		logger.Sugar().Debugf("Session continuation row: %s - %s", program.Session, program.EventName)
	} else if *lastSession != "" {
		// Continuation row - no semester or session, use previous session
		program.Session = *lastSession
		program.EventName = trimmedTds[2]
		if trimmedTds[3] != "" {
			typeVal := trimmedTds[3]
			program.Type = &typeVal
		} else {
			program.Type = nil
		}
		program.Level = trimmedTds[4]

		points, err := strconv.ParseFloat(trimmedTds[5], 32)
		if err != nil {
			logger.Sugar().Warnf("Failed to parse points '%s': %v", trimmedTds[5], err)
			programPool.Put(program)
			return
		}
		program.Points = float32(points)
		logger.Sugar().Debugf("Continuation row (using %s): %s", *lastSession, program.EventName)
	} else {
		logger.Sugar().Warnf("Skipping row with no session context: %v", trimmedTds)
		programPool.Put(program)
		return
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
		logger      = s.log.GetLogger()
		cookie      = r.Context().Value(ctxToken).(string)
		mu          sync.Mutex
		programs    []dtos.StarpointProgram
		starpoint   = &dtos.Starpoint{}
		lastSession string
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

		// Check if this is a summary row (Cummulative Average or Total Point)
		firstCellText := cells.First().Text()
		if strings.Contains(firstCellText, "Cummulative Average") {
			if starpoint.CummulativeAverage == 0 {
				starpoint.CummulativeAverage = getFloatFromString(firstCellText)
			}
			return
		}

		if strings.Contains(firstCellText, "Total Point") {
			if starpoint.TotalPoints == 0 {
				starpoint.TotalPoints = getFloatFromString(firstCellText)
			}
			return
		}

		tds := programTdStringSlicePool.Get().([]string)
		tds = tds[:0] // Reset slice

		cells.Each(func(_ int, s *goquery.Selection) {
			tds = append(tds, s.Text())
		})

		parseProgramRows(tds, &programs, &mu, &lastSession, logger)

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
