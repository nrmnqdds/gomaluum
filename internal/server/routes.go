package server

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

var DocsPath embed.FS

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		// MaxAge:           300,
	}))

	// Recoverer middleware recovers from panics, logs the panic (and a backtrace), and returns a HTTP 500 (Internal Server Error) status if possible.
	r.Use(middleware.Recoverer)

	// RedirectSlashes middleware is a simple middleware that will match request paths with a trailing slash, strip it, and redirect.
	r.Use(middleware.RedirectSlashes)

	r.Get("/reference", s.ScalarReference)

	// All routes in this group require authentication
	r.Group(func(r chi.Router) {
		// Check for PASETO token in Authorization header
		r.Use(s.PasetoAuthenticator())

		// All routes in this group start with /api
		r.Route("/api", func(r chi.Router) {
			r.Post("/login", s.LoginHandler)
			r.Get("/profile", s.ProfileHandler)
			r.Get("/schedule", s.ScheduleHandler)
			r.Get("/result", s.ResultHandler)
		})
	})

	return r
}