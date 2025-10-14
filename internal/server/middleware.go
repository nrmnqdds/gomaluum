package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nrmnqdds/gomaluum/pkg/apikey"
)

type originCookie int

const (
	ctxToken originCookie = iota
)

func (s *Server) PasetoAuthenticator() func(http.Handler) http.Handler {
	logger := s.log.GetLogger()
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			fullAuthHeader := r.Header.Get("Authorization")
			path := r.URL.Path

			// Skip authentication for login routes and API key generation
			if path == "/api/login" || path == "/api/auth/login" || path == "/api/key/generate" {
				next.ServeHTTP(w, r)
				return
			}

			if fullAuthHeader == "" || len(fullAuthHeader) < 7 || fullAuthHeader[:7] != "Bearer " {
				logger.Sugar().Warn("Authorization header is missing or invalid")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			authHeader := fullAuthHeader[7:]

			// Get API key from header
			userAPIKey := r.Header.Get("x-gomaluum-key")
			if userAPIKey == "" {
				logger.Sugar().Debug("No API key provided, using default key")
				userAPIKey = apikey.DefaultAPIKey
			} else {
				// Validate the provided API key format
				if !apikey.ValidateAPIKey(userAPIKey) {
					logger.Sugar().Warn("Invalid API key format")
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			}

			token, err := s.DecodePasetoToken(authHeader, userAPIKey)
			if err != nil {
				logger.Sugar().Errorf("Failed to decode token: %v", err)

				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if token == nil {
				logger.Sugar().Warn("Token is empty")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			logger.Sugar().Debugf("Token is authenticated: %v", fmt.Sprintf("MOD_AUTH_CAS=%s", token.imaluumCookie))

			// Create a new context from the request context and add the token to it
			ctx := context.WithValue(r.Context(), ctxToken, token.imaluumCookie)

			// Token is authenticated, pass it through
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}
