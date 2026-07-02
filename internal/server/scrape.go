package server

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gocolly/colly/v2"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// isTruthyParam reports whether a query-param value means "on" (e.g. ?refresh=1
// or ?refresh=true). Everything else — "0", "false", or absent — is false, so
// callers default to the cached path.
func isTruthyParam(v string) bool {
	return v == "1" || strings.EqualFold(v, "true")
}

// detectStale flags the scrape as stale if the fetched page contains a CAS
// login password field. Authenticated i-Ma'luum data pages never do, so its
// presence means we were bounced to the login page (cookie no longer valid).
func detectStale(c *colly.Collector, stale *atomic.Bool) {
	c.OnHTML(`input[name="password"]`, func(*colly.HTMLElement) {
		stale.Store(true)
	})
}

// applyImaluumHeaders registers the headers every authenticated i-Ma'luum
// scrape must send: the session cookie, a real browser User-Agent, and an
// Accept header containing text/html. The latter two are mandatory — /MyAcademic/*
// responds 403 without them. See constants.DefaultUserAgent / DefaultAcceptHeader.
func applyImaluumHeaders(c *colly.Collector, cookie string) {
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", "MOD_AUTH_CAS="+cookie)
		r.Headers.Set("User-Agent", constants.DefaultUserAgent)
		r.Headers.Set("Accept", constants.DefaultAcceptHeader)
	})
}

// classifyVisitError maps a colly Visit error to a CustomError. colly reports a
// non-2xx upstream response as errors.New(http.StatusText(code)), so a 403
// surfaces as "Forbidden". That means i-Ma'luum blocked the request (typically
// the server's IP being banned) — an upstream failure, not our bug — so it maps
// to ErrUpstreamForbidden (502) to stay distinguishable from a genuine 500.
// Everything else (transport failures, other statuses) stays a generic 500.
func classifyVisitError(err error) *errors.CustomError {
	if err != nil && err.Error() == http.StatusText(http.StatusForbidden) {
		return errors.Wrap(errors.ErrUpstreamForbidden, err)
	}
	return errors.Wrap(errors.ErrFailedToGoToURL, err)
}

// newImaluumCollector builds a colly.Collector wired for an authenticated
// i-Ma'luum scrape: the shared HTTP transport, stale-session detection, and the
// required request headers. ctx carries the request's trace span so upstream
// errors are correlated. Callers add their own OnHTML handlers and Visit.
func (s *Server) newImaluumCollector(ctx context.Context, cookie string, stale *atomic.Bool) *colly.Collector {
	c := colly.NewCollector()
	c.WithTransport(s.httpClient.Transport)
	detectStale(c, stale)
	applyImaluumHeaders(c, cookie)
	s.logUpstreamError(ctx, c)
	return c
}

// logUpstreamError records the details of any non-2xx i-Ma'luum response so a
// recurring 403 can be diagnosed from production logs. colly does not run OnHTML
// for error responses, so the block page (Cloudflare challenge, WAF/IP-ban page,
// Laravel error, etc.) is otherwise thrown away and every failure looks like a
// generic "Forbidden". The detail is emitted both as a trace-correlated log
// record (via the otelslog bridge) and as an event on the request's span, so it
// surfaces in SigNoz on the offending trace. Body is truncated to keep the
// signal bounded.
func (s *Server) logUpstreamError(ctx context.Context, c *colly.Collector) {
	c.OnError(func(r *colly.Response, err error) {
		url := ""
		if r.Request != nil && r.Request.URL != nil {
			url = r.Request.URL.String()
		}
		body := string(r.Body)
		if len(body) > 512 {
			body = body[:512]
		}
		var server, cfRay, contentType string
		if r.Headers != nil {
			server = r.Headers.Get("Server")
			cfRay = r.Headers.Get("CF-Ray")
			contentType = r.Headers.Get("Content-Type")
		}
		s.log.ErrorContext(ctx, "i-Ma'luum upstream error",
			"status", r.StatusCode,
			"url", url,
			"server", server,
			"cf_ray", cfRay,
			"content_type", contentType,
			"body_snippet", body,
			"error", err,
		)

		span := trace.SpanFromContext(ctx)
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("imaluum.status", r.StatusCode),
			attribute.String("imaluum.url", url),
			attribute.String("imaluum.server", server),
			attribute.String("imaluum.cf_ray", cfRay),
			attribute.String("imaluum.content_type", contentType),
			attribute.String("imaluum.body_snippet", body),
		))
		span.SetStatus(codes.Error, "i-Ma'luum upstream error")
	})
}

// runWithRetry runs fn with cookie. If fn reports a stale session, it calls
// refresh for a new cookie and retries fn exactly once. Still stale after the
// retry returns ErrStaleSession.
func runWithRetry(
	cookie string,
	refresh func() (string, error),
	fn func(cookie string) (stale bool, err error),
) error {
	stale, err := fn(cookie)
	if err != nil {
		return err
	}
	if !stale {
		return nil
	}

	cookie, err = refresh()
	if err != nil {
		return err
	}

	stale, err = fn(cookie)
	if err != nil {
		return err
	}
	if stale {
		return errors.ErrStaleSession
	}
	return nil
}

// scrapeWithRetry wires runWithRetry to the request's session: it supplies the
// current cookie and a refresh that evicts + re-logins the session.
func (s *Server) scrapeWithRetry(ctx context.Context, fn func(cookie string) (bool, error)) error {
	sess, ok := ctx.Value(ctxSession).(*TokenPayload)
	if !ok || sess == nil {
		return errors.ErrInvalidToken
	}
	return runWithRetry(
		sess.imaluumCookie,
		func() (string, error) { return s.refreshSession(ctx, sess.username, sess.password) },
		fn,
	)
}
