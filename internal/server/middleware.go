package server

import (
	"context"
	"net/http"

	"github.com/nrmnqdds/gomaluum/pkg/apikey"
)

type originCookie int

const (
	ctxToken originCookie = iota
	ctxSession
)

func (s *Server) PasetoAuthenticator() func(http.Handler) http.Handler {
	logger := s.log
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
				logger.WarnContext(r.Context(), "Authorization header is missing or invalid")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			authHeader := fullAuthHeader[7:]

			// Get API key from header
			userAPIKey := r.Header.Get("x-gomaluum-key")
			if userAPIKey == "" {
				userAPIKey = apikey.DefaultAPIKey
			} else {
				// Validate the provided API key format
				if !apikey.ValidateAPIKey(userAPIKey) {
					logger.WarnContext(r.Context(), "Invalid API key format")
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			}

			token, err := s.DecodePasetoToken(r.Context(), authHeader, userAPIKey)
			if err != nil {
				logger.ErrorContext(r.Context(), "Failed to decode token", "error", err)

				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if token == nil {
				logger.WarnContext(r.Context(), "Token is empty")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			logger.DebugContext(r.Context(), "Token is authenticated", "cookie", "MOD_AUTH_CAS="+token.imaluumCookie)

			// Create a new context from the request context and add the token to it
			ctx := context.WithValue(r.Context(), ctxToken, token.imaluumCookie)
			ctx = context.WithValue(ctx, ctxSession, token)

			// Token is authenticated, pass it through
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}
