package server

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

// @Title ProfileHandler
// @Description Get i-Ma'luum profile
// @Tags scraper
// @Produce json
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/profile [get]
func (s *Server) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger = s.log
	)

	var profile *dtos.Profile
	if err := s.scrapeWithRetry(r.Context(), func(cookie string) (bool, error) {
		p, stale, err := s.Profile(r.Context(), cookie)
		if err != nil {
			return false, err
		}
		profile = p
		return stale, nil
	}); err != nil {
		errors.Render(w, r, err)
		return
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched profile",
		Data:    profile,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(r.Context(), "Failed to encode response", "error", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
