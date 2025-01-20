package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
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

	ctx := context.Background()

	logger := s.log.GetLogger()

	user := &pb.LoginRequest{}

	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		logger.Sugar().Errorf("Failed to decode request body: %v", err)
		errors.Render(w, errors.ErrInvalidRequest)
		return
	}

	resp, err := s.grpc.Login(ctx, user)
	if err != nil {
		errors.Render(w, err)
		return
	}

	newCookie, _, err := s.GeneratePasetoToken(resp.Token, resp.Username, resp.Password)
	if err != nil {
		logger.Sugar().Errorf("Failed to generate PASETO token: %v", err)
		errors.Render(w, errors.ErrFailedToDecodePASETO)
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

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, errors.ErrFailedToEncodeResponse)
	}
}
