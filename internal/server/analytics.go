package server

import (
	"net/http"
	"sort"

	"github.com/bytedance/sonic"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

func (s *Server) UpdateAnalytics(matricNo string) error {
	_, err := s.db.Exec(`
			INSERT INTO analytics (matric_no)
			VALUES (?)
			ON CONFLICT(matric_no)
			DO UPDATE SET timestamp = current_timestamp
		`, matricNo)
	if err != nil {
		return err
	}

	return nil
}

// @Title GetAnalyticsSummaryHandler
// @Description Get analytics summary grouped by level and batch
// @Tags analytics
// @Produce json
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/analytics [get]
func (s *Server) GetAnalyticsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := s.db.Query(`
		SELECT level, batch, COUNT(*) as student_count
		FROM analytics
		GROUP BY level, batch
		ORDER BY level, batch
	`)
	if err != nil {
		errors.Render(w, r, errors.ErrFailedToQueryDB)
		return
	}
	defer rows.Close()

	summaryMap := make(map[string][]dtos.StudentBatch)

	for rows.Next() {
		var level string
		var batch, count int

		if err := rows.Scan(&level, &batch, &count); err != nil {
			errors.Render(w, r, errors.ErrFailedToMapDBRows)
			return
		}

		summaryMap[level] = append(summaryMap[level], dtos.StudentBatch{
			Batch: batch,
			Count: count,
		})
	}

	// Convert map → slice and sort batches within each level
	var result []dtos.LevelSummary
	for level, students := range summaryMap {
		// Sort batches descending inside this level
		sort.Slice(students, func(i, j int) bool {
			return students[i].Batch > students[j].Batch
		})

		result = append(result, dtos.LevelSummary{
			Level:    level,
			Students: students,
		})
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched analytics summary",
		Data:    result,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		s.log.GetLogger().Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
