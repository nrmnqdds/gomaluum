package server

import (
	"embed"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/riandyrn/otelchi"
	slogchi "github.com/samber/slog-chi"
)

var DocsPath embed.FS

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()

	// OpenTelemetry server-side tracing. Names spans by the matched chi route
	// pattern (e.g. "GET /api/schedule") and sets http.route, so SigNoz can
	// break down traffic per endpoint. Registered first so the span context is
	// available to downstream middleware (access logs get trace/span IDs).
	r.Use(otelchi.Middleware(os.Getenv("OTEL_SERVICE_NAME"), otelchi.WithChiRoutes(r)))

	// Structured access logging with trace/span correlation. Health checks are
	// filtered out to avoid probe noise.
	r.Use(slogchi.NewWithConfig(s.log, slogchi.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
		WithRequestID:    true,
		WithSpanID:       true,
		WithTraceID:      true,
		Filters: []slogchi.Filter{
			slogchi.IgnorePath("/health"),
		},
	}))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "x-gomaluum-key"},
		AllowCredentials: true,
		// MaxAge:           300,
	}))

	// Recoverer middleware recovers from panics, logs the panic (and a backtrace), and returns a HTTP 500 (Internal Server Error) status if possible.
	r.Use(middleware.Recoverer)

	// RedirectSlashes middleware is a simple middleware that will match request paths with a trailing slash, strip it, and redirect.
	r.Use(middleware.RedirectSlashes)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/reference", http.StatusMovedPermanently)
	})

	r.Get("/health", s.HealthHandler())

	// All routes in this group start with /api
	r.Route("/api", func(r chi.Router) {
		// Scalar UI
		r.Get("/reference", s.ScalarReference)

		r.Get("/analytics", s.GetAnalyticsSummaryHandler)

		// Backward compatibility
		r.Post("/login", s.LoginHandler)

		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", s.LoginHandler)
			r.Group(func(r chi.Router) {
				// Check for PASETO token in Authorization header
				r.Use(s.PasetoAuthenticator())
				r.Get("/logout", s.LogoutHandler)
			})
		})

		// API Key routes
		r.Route("/key", func(r chi.Router) {
			r.Post("/generate", s.GenerateAPIKeyHandler)
		})

		r.Get("/ads", s.AdsHandler)
		r.Get("/academic-calendar", s.AcademicCalendarHandler)

		// All routes in this group require authentication
		r.Group(func(r chi.Router) {
			// Check for PASETO token in Authorization header
			r.Use(s.PasetoAuthenticator())

			r.Get("/profile", s.ProfileHandler)
			r.Get("/schedule", s.ScheduleHandler)
			r.Get("/result", s.ResultHandler)
			r.Get("/starpoint", s.StarpointHandler)
			r.Get("/exam-timetable", s.FinalExamHandler)
			r.Get("/disciplinary", s.DisciplinaryHandler)
			r.Get("/carry-mark", s.CarryMarkHandler)
			r.Get("/logout", s.LogoutHandler)

			r.Route("/download", func(r chi.Router) {
				r.Get("/exam-slip", s.ExamSlipHandler)
				r.Get("/study-plan", s.StudyPlanHandler)
			})
		})
	})

	return r
}
