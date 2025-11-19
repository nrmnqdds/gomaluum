package server

import (
	"encoding/json"
	"net/http"

	"github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/nrmnqdds/gomaluum/pkg/apikey"
)

type APIKeyResponse struct {
	APIKey    string `json:"api_key"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// GenerateAPIKeyHandler generates a new API key for authentication
// @Summary Generate API Key
// @Description Generate a new API key for additional authentication layer
// @Tags key
// @Accept json
// @Produce json
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Success 200 {object} APIKeyResponse
// @Failure 500 {object} errors.CustomError
// @Router /api/key/generate [post]
func (s *Server) GenerateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	logger := s.log.GetLogger()

	// Generate a new API key
	newAPIKey, err := apikey.GenerateTimestampedAPIKey()
	if err != nil {
		logger.Sugar().Errorf("Failed to generate API key: %v", err)
		errors.Render(w, r, errors.ErrFailedToGenerateAPIKey)
		return
	}

	response := APIKeyResponse{
		APIKey:    newAPIKey,
		Message:   "API key generated successfully. Include this key in the 'x-gomaluum-key' header for enhanced security.",
		CreatedAt: "now", // You might want to use actual timestamp
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToGenerateAPIKey)
		return
	}

	logger.Sugar().Infof("Generated new API key: %s", newAPIKey[:10]+"...") // Log only first 10 chars for security
}
