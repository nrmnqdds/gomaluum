package server

import (
	"context"
	"sync/atomic"

	"github.com/gocolly/colly/v2"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
)

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
