package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
		logger.ErrorContext(r.Context(), "Failed to create exam slip request", "error", err)
		errors.Render(w, r, errors.Wrap(errors.ErrURLParseFailed, err))
		return
	}

	setHeadersWithCookie(req, cookie)

	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to do request", "error", err)
		errors.Render(w, r, errors.ErrURLParseFailed)
		return
	}
	defer resp.Body.Close() // Use defer to close the body after we're done with it

	if !checkDownloadResponse(r.Context(), logger, resp) {
		errors.Render(w, r, errors.ErrDownloadFailed)
		return
	}

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
		logger.ErrorContext(r.Context(), "Failed to create study plan request", "error", err)
		errors.Render(w, r, errors.Wrap(errors.ErrURLParseFailed, err))
		return
	}

	setHeadersWithCookie(req, cookie)

	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to do request", "error", err)
		errors.Render(w, r, errors.ErrURLParseFailed)
		return
	}
	defer resp.Body.Close() // Use defer to close the body after we're done with it

	if !checkDownloadResponse(r.Context(), logger, resp) {
		errors.Render(w, r, errors.ErrDownloadFailed)
		return
	}

	// Stream to response
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.ErrorContext(r.Context(), "Failed to copy response body", "error", err)
		errors.Render(w, r, errors.ErrDownloadFailed)
		return
	}
}

// checkDownloadResponse reports whether the upstream response is OK to stream.
// i-Ma'luum's /MyAcademic/* download paths can return a 403 block page; without
// this check that HTML body would be streamed to the client masquerading as a
// PDF. On a non-2xx it logs the block-page details (status, headers, snippet) so
// a recurring 403 can be diagnosed from production logs.
func checkDownloadResponse(ctx context.Context, logger *slog.Logger, resp *http.Response) bool {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	url := ""
	if resp.Request != nil && resp.Request.URL != nil {
		url = resp.Request.URL.String()
	}
	logger.ErrorContext(ctx, "i-Ma'luum download upstream error",
		"status", resp.StatusCode,
		"url", url,
		"server", resp.Header.Get("Server"),
		"cf_ray", resp.Header.Get("CF-Ray"),
		"content_type", resp.Header.Get("Content-Type"),
		"body_snippet", string(body),
	)

	span := trace.SpanFromContext(ctx)
	span.RecordError(errors.ErrDownloadFailed, trace.WithAttributes(
		attribute.Int("imaluum.status", resp.StatusCode),
		attribute.String("imaluum.url", url),
		attribute.String("imaluum.server", resp.Header.Get("Server")),
		attribute.String("imaluum.cf_ray", resp.Header.Get("CF-Ray")),
		attribute.String("imaluum.content_type", resp.Header.Get("Content-Type")),
		attribute.String("imaluum.body_snippet", string(body)),
	))
	span.SetStatus(codes.Error, "i-Ma'luum download upstream error")
	return false
}

// Function to set headers for a request.
func setHeadersWithCookie(req *http.Request, cookie string) {
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Accept-Language", "en-US")
	// i-Ma'luum's /MyAcademic/* paths (e.g. study plan) 403 unless the request
	// carries both a real browser User-Agent and an Accept header containing
	// text/html. A bare "Mozilla/5.0" is rejected.
	req.Header.Set("Accept", constants.DefaultAcceptHeader)
	req.Header.Set("User-Agent", constants.DefaultUserAgent)
	req.Header.Set("Cookie", "MOD_AUTH_CAS="+cookie)
}
