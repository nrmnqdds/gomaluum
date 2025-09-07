package server

import (
	"context"
	"fmt"
	"net/http"
)

type originCookie int

const (
	ctxToken originCookie = iota
)

func (s *Server) PasetoAuthenticator() func(http.Handler) http.Handler {
	logger := s.log.GetLogger()
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			fullAauthHeader := r.Header.Get("Authorization")
			path := r.URL.Path

			if path == "/api/login" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := fullAauthHeader[7:]

			token, err := s.DecodePasetoToken(authHeader)
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
