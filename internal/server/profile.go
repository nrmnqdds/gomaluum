package server

import (
	"net/http"

	"github.com/mailru/easyjson"
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
	var (
		logger = s.log.GetLogger()
		cookie = r.Context().Value(ctxToken).(string)
	)

	profile, err := s.Profile(cookie)
	if err != nil {
		logger.Sugar().Errorf("Failed to get profile: %v", err)
		errors.Render(w, err)
		return
	}

	response := &dtos.ResponseDTO{
		Message: "Successfully fetched profile",
		Data:    profile,
	}

	if _, _, err := easyjson.MarshalToHTTPResponseWriter(response, w); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, errors.ErrFailedToEncodeResponse)
	}
}
