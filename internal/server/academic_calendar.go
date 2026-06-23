package server

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

//go:embed academic_calendar_data.json
var academicCalendarData []byte

type academicCalendarItem struct {
	Title         string `json:"title"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	StartDateUnix int64  `json:"start_date_unix"`
	EndDateUnix   int64  `json:"end_date_unix"`
}

type academicCalendarRaw struct {
	Title     string `json:"title"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// @Title AcademicCalendarHandler
// @Description Get IIUM academic calendar
// @Tags academic
// @Produce json
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/academic-calendar [get]
func (s *Server) AcademicCalendarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	logger := s.log

	var raw []academicCalendarRaw
	if err := json.Unmarshal(academicCalendarData, &raw); err != nil {
		logger.ErrorContext(r.Context(), "Failed to parse academic calendar data", "error", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
		return
	}

	items := make([]academicCalendarItem, 0, len(raw))
	for _, entry := range raw {
		startTime, err := time.Parse(time.RFC3339, entry.StartDate)
		if err != nil {
			logger.ErrorContext(r.Context(), "Failed to parse start_date", "start_date", entry.StartDate, "error", err)
			errors.Render(w, r, errors.ErrFailedToEncodeResponse)
			return
		}
		endTime, err := time.Parse(time.RFC3339, entry.EndDate)
		if err != nil {
			logger.ErrorContext(r.Context(), "Failed to parse end_date", "end_date", entry.EndDate, "error", err)
			errors.Render(w, r, errors.ErrFailedToEncodeResponse)
			return
		}
		items = append(items, academicCalendarItem{
			Title:         entry.Title,
			StartDate:     entry.StartDate,
			EndDate:       entry.EndDate,
			StartDateUnix: startTime.Unix(),
			EndDateUnix:   endTime.Unix(),
		})
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched academic calendar",
		Data:    items,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(r.Context(), "Failed to encode response", "error", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
