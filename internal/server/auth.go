package server

import (
	"encoding/json"
	"net/http"

	"github.com/nrmnqdds/gomaluum/internal/dtos"
	pb "github.com/nrmnqdds/gomaluum/internal/proto"
)

// @Title LoginHandler
// @Description Logs in the user. Save the token and use it in the Authorization header for future requests.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body pb.LoginRequest true "Login properties"
// @Success 200 {object} dtos.ResponseDTO
// @Router /auth/login [post]
func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	redirect := r.URL.Query().Get("redirect")
	logger := s.log.GetLogger()

	user := &pb.LoginRequest{}

	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		logger.Sugar().Errorf("Failed to decode request body: %v", err)
		_, _ = w.Write([]byte("Failed to decode request body"))
		return
	}

	resp, err := s.Login(r.Context(), user)
	if err != nil {
		logger.Sugar().Errorf("Failed to login: %v", err)
		_, _ = w.Write([]byte("Failed to login"))
		return
	}

	newCookie, _, err := s.GeneratePasetoToken(resp.Token, resp.Username, resp.Password)
	if err != nil {
		logger.Sugar().Errorf("Failed to generate PASETO token: %v", err)
		_, _ = w.Write([]byte("Failed to generate PASETO token"))
		return
	}

	result := &pb.LoginResponse{
		Token:    newCookie,
		Username: resp.Username,
	}

	response := &dtos.ResponseDTO{
		Message: "Login successful! Please use the token in the Authorization header for future requests.",
		Data:    result,
	}

	jsonResp, err := json.Marshal(response)
	if err != nil {
		logger.Sugar().Errorf("Failed to marshal JSON: %v", err)
		_, _ = w.Write([]byte("Failed to marshal JSON"))
		return
	}

	if redirect == "" {
		_, _ = w.Write(jsonResp)
		return
	}

	w.Header().Add("Hx-Redirect", "/dashboard")
	// _, _ = w.Write(jsonResp)
}
