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
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/profile [get]
func (s *Server) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		logger = s.log.GetLogger()
		cookie = r.Context().Value(ctxToken).(string)
	)

	profile, err := s.Profile(cookie)
	if err != nil {
		logger.Sugar().Errorf("Failed to get profile: %v", err)
		errors.Render(w, r, err)
		return
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched profile",
		Data:    profile,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}
