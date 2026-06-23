package server

import (
	"net/http"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

func (s *Server) ScalarReference(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	logger := s.log

	swaggerContent, err := DocsPath.ReadFile("docs/swagger/swagger.json")
	if err != nil {
		logger.ErrorContext(r.Context(), "could not read swagger.json", "error", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
		return
	}
	customCSS, err := DocsPath.ReadFile("docs/swagger/flytheme.css")
	if err != nil {
		logger.ErrorContext(r.Context(), "could not read flytheme.css", "error", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
		return
	}

	htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
		SpecContent: string(swaggerContent),
		CustomOptions: scalar.CustomOptions{
			PageTitle: "GoMaluum API",
		},
		DarkMode:  true,
		CustomCss: string(customCSS),
	})
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to render API reference HTML", "error", err)
		errors.Render(w, r, errors.ErrFailedToEncodeResponse)
	}

	_, _ = w.Write([]byte(htmlContent))
}
