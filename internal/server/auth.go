package server

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/bytedance/sonic"
	"github.com/mailru/easyjson"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	pb "github.com/nrmnqdds/gomaluum/internal/proto"
	"github.com/nrmnqdds/gomaluum/pkg/apikey"
)

// @Title LoginHandler
// @Description Logs in the user. Save the token and use it in the Authorization header for future requests.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body pb.LoginRequest true "Login properties"
// @Success 200 {object} dtos.ResponseDTO
// @Router /api/auth/login [post]
func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := context.Background()

	logger := s.log.GetLogger()

	user := &pb.LoginRequest{}

	// Bind the request body to the user struct
	if err := easyjson.UnmarshalFromReader(r.Body, user); err != nil {
		logger.Sugar().Errorf("Failed to decode request body: %v", err)
		errors.Render(w, r, errors.ErrInvalidRequest)
		return
	}

	// Call the Login method from the GRPC server
	resp, err := s.grpc.Login(ctx, user)
	if err != nil {
		logger.Sugar().Errorf("Login failed: %v", err)
		errors.Render(w, r, err)
		return
	}

	// Get API key from header
	userAPIKey := r.Header.Get("x-gomaluum-key")
	if userAPIKey == "" {
		logger.Sugar().Debug("No API key provided in login, using default key")
		userAPIKey = apikey.DefaultAPIKey
	} else {
		// Validate the provided API key format
		if !apikey.ValidateAPIKey(userAPIKey) {
			logger.Sugar().Warn("Invalid API key format provided in login")
			errors.Render(w, r, errors.ErrInvalidAPIKey)
			return
		}
	}

	payload := TokenPayload{
		username:      resp.Username,
		password:      resp.Password,
		imaluumCookie: resp.Token,
		apiKey:        userAPIKey,
	}

	// Generate a new PASETO token
	newCookie, _, err := s.GeneratePasetoToken(payload)
	if err != nil {
		logger.Sugar().Errorf("Failed to generate PASETO token: %v", err)
		errors.Render(w, r, errors.ErrFailedToDecodePASETO)
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

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}

// @Title LogoutHandler
// @Description Logs out the user. Clears the token from IIUM's CAS. PASETO token is still valid.
// @Tags auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {object} dtos.ResponseDTO
// @Router /auth/logout [get]
func (s *Server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	logger := s.log.GetLogger()

	jar, _ := cookiejar.New(nil)

	cookie := r.Context().Value(ctxToken).(string)

	urlObj, err := url.Parse(constants.ImaluumLogoutPage)
	if err != nil {
		errors.Render(w, r, errors.ErrURLParseFailed)
		return
	}

	jar.SetCookies(urlObj, []*http.Cookie{
		{
			Name:  "MOD_AUTH_CAS",
			Value: cookie,
		},
	})

	client := &http.Client{
		Jar: jar,
	}

	req, _ := http.NewRequest("GET", constants.ImaluumCasLogoutPage, nil)
	setHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		errors.Render(w, r, errors.ErrURLParseFailed)
		return
	}
	resp.Body.Close()

	response := &dtos.ResponseDTO{
		Message: "Logout successful! Token has been cleared.",
		Data:    nil,
	}

	if err := sonic.ConfigFastest.NewEncoder(w).Encode(response); err != nil {
		logger.Sugar().Errorf("Failed to encode response: %v", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}
}

// Function to set headers for a request.
func setHeaders(req *http.Request) {
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", "Mozilla/5.0")
}
