package server

import (
	"embed"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/nrmnqdds/gomaluum/templates"
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

	// Serves static files from the given file system root.
	r.Get("/favicon.ico", s.ServeFavicon)
	r.Get("/static/*", s.ServeStaticFiles)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		s.Chain(w, r, templates.HomeScreen())
	})

	r.Get("/reference", s.ScalarReference)

	// All routes in this group require authentication
	r.Group(func(r chi.Router) {
		// Check for PASETO token in Authorization header
		r.Use(s.PasetoAuthenticator())

		r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			s.Chain(w, r, templates.LoadingScreen())
			cookie := r.Context().Value(ctxToken).(string)
			profile, err := s.Profile(cookie)
			if err != nil {
				_, _ = w.Write([]byte("Failed to get profile"))
				return
			}
			s.Chain(w, r, templates.DashboardScreen(profile))
		})

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

func (s *Server) ServeFavicon(w http.ResponseWriter, r *http.Request) {
	filePath := "favicon.ico"
	fullPath := filepath.Join(".", "static", filePath)
	http.ServeFile(w, r, fullPath)
}

func (s *Server) ServeStaticFiles(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path[len("/static/"):]
	fullPath := filepath.Join(".", "static", filePath)
	http.ServeFile(w, r, fullPath)
}
