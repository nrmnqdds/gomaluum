package server

import (
	"io"
	"log"
	"net/http"

	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

// @Title ExamSlipHandler
// @Description Get exam slip PDF from i-Ma'luum
// @Tags download
// @Produce application/pdf
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {string} string "Exam slip PDF"
// @Router /api/download/exam-slip [get]
func (s *Server) ExamSlipHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/pdf")

	var (
		logger = s.log
		cookie = r.Context().Value(ctxToken).(string)
		client = s.httpClient
	)

	req, err := http.NewRequestWithContext(r.Context(), "GET", constants.ImaluumExamSlipPage, nil)
	if err != nil {
		log.Printf("Failed to create first request: %v", err)
		if err := req.Body.Close(); err != nil {
			log.Printf("Failed to close request body: %v", err)
			errors.Render(w, r, errors.ErrFailedToCloseRequestBody)
		}
		errors.Render(w, r, errors.ErrURLParseFailed)
	}

	setHeadersWithCookie(req, cookie)

	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to do request", "error", err)
		errors.Render(w, r, errors.ErrURLParseFailed)
		return
	}
	defer resp.Body.Close() // Use defer to close the body after we're done with it

	// Stream to response
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.ErrorContext(r.Context(), "Failed to copy response body", "error", err)
		errors.Render(w, r, errors.ErrDownloadFailed)
		return
	}
}

// @Title StudyPlanHandler
// @Description Get study plan PDF from i-Ma'luum
// @Tags download
// @Produce application/pdf
// @Param x-gomaluum-key header string false "API key for additional security layer"
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success 200 {string} string "Exam slip PDF"
// @Router /api/download/study-plan [get]
func (s *Server) StudyPlanHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/pdf")

	var (
		logger = s.log
		cookie = r.Context().Value(ctxToken).(string)
		client = s.httpClient
	)

	req, err := http.NewRequestWithContext(r.Context(), "GET", constants.ImaluumStudyPlanPage, nil)
	if err != nil {
		log.Printf("Failed to create first request: %v", err)
		if err := req.Body.Close(); err != nil {
			log.Printf("Failed to close request body: %v", err)
			errors.Render(w, r, errors.ErrFailedToCloseRequestBody)
		}
		errors.Render(w, r, errors.ErrURLParseFailed)
	}

	setHeadersWithCookie(req, cookie)

	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to do request", "error", err)
		errors.Render(w, r, errors.ErrURLParseFailed)
		return
	}
	defer resp.Body.Close() // Use defer to close the body after we're done with it

	// Stream to response
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.ErrorContext(r.Context(), "Failed to copy response body", "error", err)
		errors.Render(w, r, errors.ErrDownloadFailed)
		return
	}
}

// Function to set headers for a request.
func setHeadersWithCookie(req *http.Request, cookie string) {
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Cookie", "MOD_AUTH_CAS="+cookie)
}
