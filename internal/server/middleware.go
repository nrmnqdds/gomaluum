package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/a-h/templ"
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

			if token == "" {
				logger.Sugar().Warn("Token is empty")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			logger.Sugar().Debugf("Token is authenticated: %v", fmt.Sprintf("MOD_AUTH_CAS=%s", token))

			// Create a new context from the request context and add the token to it
			ctx := context.WithValue(r.Context(), ctxToken, token)

			// Token is authenticated, pass it through
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}

type CustomContext struct {
	context.Context
	StartTime time.Time
	Cookie    string
}

type (
	TemplHandler    func(ctx *CustomContext, w http.ResponseWriter, r *http.Request)
	TemplMiddleware func(ctx *CustomContext, w http.ResponseWriter, r *http.Request) error
)

func (s *Server) Chain(w http.ResponseWriter, r *http.Request, template templ.Component, middleware ...TemplMiddleware) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	customContext := &CustomContext{
		Context:   context.Background(),
		StartTime: time.Now(),
	}
	for _, mw := range middleware {
		if err := mw(customContext, w, r); err != nil {
			return
		}
	}
	if err := template.Render(customContext, w); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	if err := Log(customContext, w, r); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func Log(ctx *CustomContext, _ http.ResponseWriter, r *http.Request) error {
	elapsedTime := time.Since(ctx.StartTime)
	formattedTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] [%s] [%s] [%s]\n", formattedTime, r.Method, r.URL.Path, elapsedTime)
	return nil
}
