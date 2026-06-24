# Stale-Session Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a cached i-Ma'luum session cookie goes stale and a scrape gets bounced to the CAS login page, evict the cached session, re-login, and retry the scrape once with a fresh cookie.

**Architecture:** The auth middleware puts the decoded `*TokenPayload` (cookie + credentials) into the request context. A pure `runWithRetry` helper runs a scrape closure, and on a detected stale session calls `refreshSession` (evict + re-login) and retries once. Each scraper reports "stale" via an `atomic.Bool` set when a CAS password field appears in the fetched HTML.

**Tech Stack:** Go, chi, gocolly/colly v2, gRPC (auth service), stretchr/testify.

## Global Constraints

- Go 1.25 (`go.mod`).
- Commit messages: no `Co-Authored-By` / Claude attribution trailer.
- Spec: `specs/2026-06-24-stale-session-recovery-design.md`.
- Branch: `fix/token-refresh-ttl` (stacked on the 30m TTL fix).
- Retry **once** only. Each scrape closure must **reset its result accumulator** at the top so a retry starts clean.
- `imaluumSessionTTL = 30 * time.Minute` already exists in `internal/server/paseto.go`.

---

## File Structure

- `pkg/sf/singleflight.go` — add `Invalidate`. Test: `pkg/sf/singleflight_test.go` (new).
- `internal/server/paseto.go` — prereq fixes (password normalize, remove `log.Fatal`), extract `loginFunc`, add `refreshSession`.
- `internal/server/middleware.go` — add `ctxSession` key; store `*TokenPayload`.
- `internal/errors/auth.error.go` — add `ErrStaleSession`.
- `internal/server/scrape.go` (new) — `detectStale`, `runWithRetry`, `scrapeWithRetry`. Test: `internal/server/scrape_test.go` (new).
- Scraper integrations: `carry_mark.go`, `disciplinary.go`, `final_exam.go`, `starpoint.go`, `profile.go` + `profile.service.go`, `schedule.go`, `result.go`.

---

## Task 1: `TokenManager.Invalidate`

**Files:**
- Modify: `pkg/sf/singleflight.go`
- Test: `pkg/sf/singleflight_test.go` (create)

**Interfaces:**
- Produces: `func (tm *TokenManager) Invalidate(matric string)`

- [ ] **Step 1: Write the failing test**

Create `pkg/sf/singleflight_test.go`:

```go
package sf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInvalidate_ForcesRefresh(t *testing.T) {
	tm := NewTokenManager()

	calls := 0
	refresh := func() (string, time.Time, error) {
		calls++
		return "token", time.Now().Add(time.Hour), nil
	}

	// First call populates the cache.
	tok, err := tm.GetToken("2212345", refresh)
	require.NoError(t, err)
	require.Equal(t, "token", tok)
	require.Equal(t, 1, calls)

	// Cached: refresh must NOT run again.
	_, err = tm.GetToken("2212345", refresh)
	require.NoError(t, err)
	require.Equal(t, 1, calls)

	// After Invalidate: refresh runs again.
	tm.Invalidate("2212345")
	_, err = tm.GetToken("2212345", refresh)
	require.NoError(t, err)
	require.Equal(t, 2, calls)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sf/ -run TestInvalidate_ForcesRefresh -v`
Expected: FAIL — `tm.Invalidate undefined`.

- [ ] **Step 3: Add the method**

In `pkg/sf/singleflight.go`, after `GetToken`:

```go
// Invalidate removes a cached token so the next GetToken re-runs refreshFunc.
func (tm *TokenManager) Invalidate(matric string) {
	tm.mu.Lock()
	delete(tm.tokens, matric)
	tm.mu.Unlock()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sf/ -run TestInvalidate_ForcesRefresh -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/sf/singleflight.go pkg/sf/singleflight_test.go
git commit -m "feat(sf): add TokenManager.Invalidate to evict cached sessions"
```

---

## Task 2: Prerequisite fixes in `DecodePasetoToken`

Two latent bugs that `refreshSession` will inherit: `TokenPayload.password` is base64 (not plaintext) on the non-expired return path, and a failed `GetToken` calls `log.Fatal` (crashes the server).

**Files:**
- Modify: `internal/server/paseto.go`

- [ ] **Step 1: Normalize the non-expired password to plaintext**

In `internal/server/paseto.go`, the non-expired return path currently reads:

```go
	// If token not expired yet - decrypt the cookie
	encryptedCookie, _ := decodedToken.GetString("imaluumCookie")
	imaluumCookie, err := apikey.DecryptWithAPIKey(encryptedCookie, userAPIKey)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to decrypt cookie with API key", "error", err)
		return nil, err
	}

	go s.UpdateAnalytics(username)
	return &TokenPayload{
		username:      username,
		password:      password,
		imaluumCookie: imaluumCookie,
		apiKey:        userAPIKey,
	}, nil
```

Replace with (base64-decode the password so it matches the expired path):

```go
	// If token not expired yet - decrypt the cookie
	encryptedCookie, _ := decodedToken.GetString("imaluumCookie")
	imaluumCookie, err := apikey.DecryptWithAPIKey(encryptedCookie, userAPIKey)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to decrypt cookie with API key", "error", err)
		return nil, err
	}

	plainPassword, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to decode password", "error", err)
		return nil, err
	}

	go s.UpdateAnalytics(username)
	return &TokenPayload{
		username:      username,
		password:      string(plainPassword),
		imaluumCookie: imaluumCookie,
		apiKey:        userAPIKey,
	}, nil
```

- [ ] **Step 2: Replace `log.Fatal` with returning the error**

In the expired path, change:

```go
		newToken, err := s.tokenManager.GetToken(username, refresh)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to get token", "error", err)
			log.Fatal(err)
		}
```

to:

```go
		newToken, err := s.tokenManager.GetToken(username, refresh)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to get token", "error", err)
			return nil, err
		}
```

- [ ] **Step 3: Remove the now-unused `log` import**

In the import block of `internal/server/paseto.go`, delete the `"log"` line.

- [ ] **Step 4: Build & vet**

Run: `go build ./... && go vet ./internal/server/`
Expected: no output (success). If `"log" imported and not used`, confirm Step 3 was applied.

- [ ] **Step 5: Commit**

```bash
git add internal/server/paseto.go
git commit -m "fix(auth): normalize decoded password to plaintext; don't log.Fatal on login failure"
```

---

## Task 3: Extract `loginFunc` and add `refreshSession`

**Files:**
- Modify: `internal/server/paseto.go`

**Interfaces:**
- Produces:
  - `func (s *Server) loginFunc(ctx context.Context, username, password string) func() (string, time.Time, error)`
  - `func (s *Server) refreshSession(ctx context.Context, username, password string) (string, error)`

- [ ] **Step 1: Add `loginFunc` and `refreshSession`**

In `internal/server/paseto.go`, add after `DecodePasetoToken`:

```go
// loginFunc returns a TokenManager refresh closure that logs into i-Ma'luum
// (via the gRPC auth service) and caches the resulting cookie for
// imaluumSessionTTL. password must be plaintext.
func (s *Server) loginFunc(ctx context.Context, username, password string) func() (string, time.Time, error) {
	return func() (string, time.Time, error) {
		logger := s.log
		logger.DebugContext(ctx, "Refreshing session token", "username", username)

		var resp *pb.LoginResponse
		var err error

		if username == constants.DebugUsername && password == constants.DebugPassword {
			logger.InfoContext(ctx, "Using fake user for debugging (token refresh)")
			resp = &pb.LoginResponse{
				Username: constants.DebugUsername,
				Password: constants.DebugPassword,
				Token:    constants.DebugUserCookie,
			}
		} else {
			resp, err = s.grpc.client.Login(ctx, &pb.LoginRequest{
				Username: username,
				Password: password,
			})
			if err != nil {
				logger.ErrorContext(ctx, "Failed to login", "error", err)
				return "", time.Now(), err
			}
		}

		return resp.Token, time.Now().Add(imaluumSessionTTL), nil
	}
}

// refreshSession evicts the cached session for username and forces a fresh
// login, returning the new cookie. Used to recover from a stale session.
func (s *Server) refreshSession(ctx context.Context, username, password string) (string, error) {
	s.tokenManager.Invalidate(username)
	return s.tokenManager.GetToken(username, s.loginFunc(ctx, username, password))
}
```

- [ ] **Step 2: Replace the inline `refresh` closure in `DecodePasetoToken`**

In the expired path of `DecodePasetoToken`, replace the whole inline `refresh := func() (string, time.Time, error) { ... }` block with:

```go
		refresh := s.loginFunc(ctx, username, string(decodedPassword))
```

(The surrounding `decodedPassword`, `s.tokenManager.GetToken(username, refresh)`, and `UpdateAnalytics` lines stay.)

- [ ] **Step 3: Build & vet**

Run: `go build ./... && go vet ./internal/server/`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add internal/server/paseto.go
git commit -m "refactor(auth): extract loginFunc; add refreshSession (evict + re-login)"
```

---

## Task 4: Store `*TokenPayload` in request context

**Files:**
- Modify: `internal/server/middleware.go`

**Interfaces:**
- Produces: context key `ctxSession` carrying `*TokenPayload`; consumed by `scrapeWithRetry` (Task 5).

- [ ] **Step 1: Add the `ctxSession` key**

In `internal/server/middleware.go`, the const block currently is:

```go
const (
	ctxToken originCookie = iota
)
```

Change to:

```go
const (
	ctxToken originCookie = iota
	ctxSession
)
```

- [ ] **Step 2: Store the payload in context**

In `PasetoAuthenticator`'s handler, the line:

```go
			ctx := context.WithValue(r.Context(), ctxToken, token.imaluumCookie)
```

becomes:

```go
			ctx := context.WithValue(r.Context(), ctxToken, token.imaluumCookie)
			ctx = context.WithValue(ctx, ctxSession, token)
```

- [ ] **Step 3: Build & vet**

Run: `go build ./... && go vet ./internal/server/`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add internal/server/middleware.go
git commit -m "feat(auth): carry decoded session payload in request context"
```

---

## Task 5: Retry infrastructure (`ErrStaleSession`, `detectStale`, `runWithRetry`, `scrapeWithRetry`)

**Files:**
- Modify: `internal/errors/auth.error.go`
- Create: `internal/server/scrape.go`
- Test: `internal/server/scrape_test.go` (create)

**Interfaces:**
- Consumes: `ctxSession` (Task 4), `*TokenPayload` fields `username`/`password`/`imaluumCookie`, `s.refreshSession` (Task 3).
- Produces:
  - `errors.ErrStaleSession` (`*errors.CustomError`)
  - `func detectStale(c *colly.Collector, stale *atomic.Bool)`
  - `func runWithRetry(cookie string, refresh func() (string, error), fn func(cookie string) (bool, error)) error`
  - `func (s *Server) scrapeWithRetry(ctx context.Context, fn func(cookie string) (bool, error)) error`

- [ ] **Step 1: Add the sentinel error**

In `internal/errors/auth.error.go`, add:

```go
var ErrStaleSession = &CustomError{
	Message:    "Session expired and could not be refreshed, please log in again",
	StatusCode: 401,
}
```

- [ ] **Step 2: Write the failing test for `runWithRetry`**

Create `internal/server/scrape_test.go`:

```go
package server

import (
	"errors"
	"testing"

	apperrors "github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/stretchr/testify/require"
)

func TestRunWithRetry(t *testing.T) {
	t.Run("not stale: runs once, no refresh", func(t *testing.T) {
		calls, refreshes := 0, 0
		err := runWithRetry("c0",
			func() (string, error) { refreshes++; return "c1", nil },
			func(cookie string) (bool, error) { calls++; require.Equal(t, "c0", cookie); return false, nil },
		)
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Equal(t, 0, refreshes)
	})

	t.Run("stale once: refreshes and retries with new cookie", func(t *testing.T) {
		calls, refreshes := 0, 0
		err := runWithRetry("c0",
			func() (string, error) { refreshes++; return "c1", nil },
			func(cookie string) (bool, error) {
				calls++
				if calls == 1 {
					require.Equal(t, "c0", cookie)
					return true, nil
				}
				require.Equal(t, "c1", cookie)
				return false, nil
			},
		)
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, 1, refreshes)
	})

	t.Run("stale twice: returns ErrStaleSession", func(t *testing.T) {
		err := runWithRetry("c0",
			func() (string, error) { return "c1", nil },
			func(cookie string) (bool, error) { return true, nil },
		)
		require.ErrorIs(t, err, apperrors.ErrStaleSession)
	})

	t.Run("fn error: propagated, no refresh", func(t *testing.T) {
		boom := errors.New("boom")
		refreshes := 0
		err := runWithRetry("c0",
			func() (string, error) { refreshes++; return "c1", nil },
			func(cookie string) (bool, error) { return false, boom },
		)
		require.ErrorIs(t, err, boom)
		require.Equal(t, 0, refreshes)
	})

	t.Run("refresh error: propagated", func(t *testing.T) {
		boom := errors.New("login down")
		err := runWithRetry("c0",
			func() (string, error) { return "", boom },
			func(cookie string) (bool, error) { return true, nil },
		)
		require.ErrorIs(t, err, boom)
	})
}
```

> Note: `require.ErrorIs(t, err, apperrors.ErrStaleSession)` works because `runWithRetry` returns the `ErrStaleSession` pointer directly (identity match).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/server/ -run TestRunWithRetry -v`
Expected: FAIL — `undefined: runWithRetry`.

- [ ] **Step 4: Create `internal/server/scrape.go`**

```go
package server

import (
	"context"
	"sync/atomic"

	"github.com/gocolly/colly/v2"
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
	sess := ctx.Value(ctxSession).(*TokenPayload)
	return runWithRetry(
		sess.imaluumCookie,
		func() (string, error) { return s.refreshSession(ctx, sess.username, sess.password) },
		fn,
	)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/server/ -run TestRunWithRetry -v`
Expected: PASS (all subtests).

- [ ] **Step 6: Build & vet**

Run: `go build ./... && go vet ./internal/server/ ./internal/errors/`
Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add internal/errors/auth.error.go internal/server/scrape.go internal/server/scrape_test.go
git commit -m "feat(scrape): add stale-session detection and single-retry helper"
```

---

## Task 6: Integrate retry into `carry_mark.go` (single-collector exemplar)

**Files:**
- Modify: `internal/server/carry_mark.go`

**Interfaces:**
- Consumes: `s.scrapeWithRetry`, `detectStale` (Task 5).

The current handler (after the var block) creates the collector inline, visits once, then checks `len(subjects)`. Wrap the collector-build-and-visit in `scrapeWithRetry`, resetting accumulators inside.

- [ ] **Step 1: Add imports**

Ensure `internal/server/carry_mark.go` imports `"sync/atomic"`. (It already imports `sync` and `colly`.)

- [ ] **Step 2: Wrap visit in scrapeWithRetry**

Replace the block that starts at `c := colly.NewCollector()` and ends at the `c.Visit` error check:

```go
	c := colly.NewCollector()
	c.WithTransport(s.httpClient.Transport)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", cookieStr)
		r.Headers.Set("User-Agent", cuid.New())
	})

	c.OnHTML("script", func(e *colly.HTMLElement) {
		// ... existing ...
	})

	c.OnHTML("table.table.table-hover tbody tr", func(e *colly.HTMLElement) {
		// ... existing ...
	})

	if err := c.Visit(constants.ImaluumCarryMarkPage); err != nil {
		logger.ErrorContext(r.Context(), "Failed to go to URL", "error", err)
		errors.Render(w, r, errors.ErrFailedToGoToURL)
		return
	}
```

with a closure that rebuilds the collector each attempt and resets the accumulators:

```go
	if err := s.scrapeWithRetry(r.Context(), func(cookie string) (bool, error) {
		// Reset accumulators so a retry starts clean.
		mu.Lock()
		subjects = subjects[:0]
		currentSubject = nil
		session = ""
		mu.Unlock()

		var stale atomic.Bool
		c := colly.NewCollector()
		c.WithTransport(s.httpClient.Transport)
		detectStale(c, &stale)

		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set("Cookie", "MOD_AUTH_CAS="+cookie)
			r.Headers.Set("User-Agent", cuid.New())
		})

		c.OnHTML("script", func(e *colly.HTMLElement) {
			// ... existing body unchanged ...
		})

		c.OnHTML("table.table.table-hover tbody tr", func(e *colly.HTMLElement) {
			// ... existing body unchanged ...
		})

		if err := c.Visit(constants.ImaluumCarryMarkPage); err != nil {
			return false, errors.ErrFailedToGoToURL
		}
		return stale.Load(), nil
	}); err != nil {
		logger.ErrorContext(r.Context(), "Failed to scrape carry marks", "error", err)
		errors.Render(w, r, err)
		return
	}
```

Notes:
- The `cookieStr := "MOD_AUTH_CAS=" + cookie` line above the old block is now unused — delete it (the closure builds the header from its `cookie` param).
- `mu`, `subjects`, `currentSubject`, `session` remain declared in the handler's `var` block (outside the closure) so the post-scrape `len(subjects) == 0` check still sees them.
- Everything after (`if len(subjects) == 0 {...}`, response build, encode) is unchanged.

- [ ] **Step 3: Build & vet**

Run: `go build ./... && go vet ./internal/server/`
Expected: no output.

- [ ] **Step 4: Manual review checklist** (no unit test — colly hits a hardcoded URL)

Confirm by reading the diff:
- `subjects`, `currentSubject`, `session` are reset at the top of the closure.
- `detectStale(c, &stale)` is present and `return stale.Load(), nil` on success.
- The old standalone `cookieStr` is gone; the header uses the closure's `cookie`.

- [ ] **Step 5: Commit**

```bash
git add internal/server/carry_mark.go
git commit -m "feat(carry-mark): recover from stale session via scrapeWithRetry"
```

---

## Task 7: Integrate retry into `disciplinary.go`, `final_exam.go`, `starpoint.go`

Same single-collector pattern as Task 6, applied per file. For each: import `sync/atomic`; wrap the `colly.NewCollector()`…`c.Visit(...)` block in `s.scrapeWithRetry(r.Context(), func(cookie string) (bool, error) { ... })`; reset that file's accumulator(s) at the top of the closure; add `var stale atomic.Bool` + `detectStale(c, &stale)`; build the cookie header from the closure `cookie`; `return stale.Load(), nil` on success and `return false, errors.ErrFailedToGoToURL` on visit error; replace the old `c.Visit` error branch's render with a render of the `scrapeWithRetry` error.

**Files / per-file specifics:**

- `internal/server/disciplinary.go`
  - Accumulators to reset: `compounds = compounds[:0]` (under `mu`).
  - Visit URL: `constants.ImaluumDisciplinaryPage`.
  - Post-scrape check unchanged: `if len(compounds) == 0 { ... ErrNoDisciplinaryRecord ... }`.
  - Delete the now-unused `cookieStr` line.

- `internal/server/final_exam.go`
  - Accumulators to reset: `exams = exams[:0]` (under `mu`).
  - Visit URL: `constants.ImaluumFinalExamPage`.
  - Post-scrape check unchanged: `if len(exams) == 0 { ... ErrNoFinalExam ... }`.
  - Delete the now-unused `cookieStr` line.

- `internal/server/starpoint.go`
  - Accumulators to reset: `programs = programs[:0]`; also reset `lastSession = ""` and `starpoint.CummulativeAverage = 0; starpoint.TotalPoints = 0` (they accumulate across rows). Reset under `mu`.
  - The `parseProgramRows(tds, &programs, &mu, &lastSession, logger)` call inside `OnHTML` stays; `logger` is the handler's `*slog.Logger` and is captured by the closure.
  - Visit URL: `constants.ImaluumStarpointPage`.
  - Post-scrape check unchanged: `if len(programs) == 0 { ... ErrNoStarpoint ... }`.
  - Delete the now-unused `cookieStr` line.

- [ ] **Step 1: Apply the Task 6 pattern to `disciplinary.go`** (reset `compounds`, URL `ImaluumDisciplinaryPage`).
- [ ] **Step 2: Apply the Task 6 pattern to `final_exam.go`** (reset `exams`, URL `ImaluumFinalExamPage`).
- [ ] **Step 3: Apply the Task 6 pattern to `starpoint.go`** (reset `programs`, `lastSession`, `starpoint` totals, URL `ImaluumStarpointPage`).
- [ ] **Step 4: Build & vet**

Run: `go build ./... && go vet ./internal/server/`
Expected: no output.

- [ ] **Step 5: Review checklist (per file):** accumulators reset at closure top; `detectStale` present; cookie header from closure param; old `cookieStr` removed; `scrapeWithRetry` error rendered.

- [ ] **Step 6: Commit**

```bash
git add internal/server/disciplinary.go internal/server/final_exam.go internal/server/starpoint.go
git commit -m "feat(scrapers): stale-session recovery for disciplinary, final-exam, starpoint"
```

---

## Task 8: Integrate retry into `profile.go` / `profile.service.go`

`Profile` owns its collector, so it reports staleness via its return value.

**Files:**
- Modify: `internal/server/profile.service.go`, `internal/server/profile.go`

**Interfaces:**
- Produces: `func (s *Server) Profile(ctx context.Context, cookie string) (*dtos.Profile, bool, error)` (adds `stale`).

- [ ] **Step 1: Change `Profile` to detect + report stale**

In `internal/server/profile.service.go`, change the signature and body:

```go
func (s *Server) Profile(ctx context.Context, cookie string) (*dtos.Profile, bool, error) {
	logger := s.log

	// Return fake data for fake user (unchanged) — return ..., false, nil
	if cookie == constants.DebugUserCookie {
		return &dtos.Profile{ /* ... existing fake data ... */ }, false, nil
	}

	cookieStr := "MOD_AUTH_CAS=" + cookie

	var stale atomic.Bool
	c := colly.NewCollector()
	c.WithTransport(s.httpClient.Transport)
	detectStale(c, &stale)

	var profileResult *dtos.Profile
	c.OnRequest(func(r *colly.Request) { /* unchanged */ })
	c.OnHTML("body", func(e *colly.HTMLElement) { /* unchanged */ })

	if err := c.Visit(constants.ImaluumProfilePage); err != nil {
		logger.ErrorContext(ctx, "Failed to go to URL", "error", err)
		return nil, false, errors.ErrFailedToGoToURL
	}

	if stale.Load() {
		return nil, true, nil
	}

	if profileResult == nil {
		logger.ErrorContext(ctx, "Failed to extract profile data")
		return nil, false, errors.ErrFailedToGoToURL
	}

	return profileResult, false, nil
}
```

Add `"sync/atomic"` to the imports.

- [ ] **Step 2: Wrap the call in `ProfileHandler`**

In `internal/server/profile.go`, replace:

```go
	profile, err := s.Profile(r.Context(), cookie)
	if err != nil {
		// Profile already logs the specific failure cause.
		errors.Render(w, r, err)
		return
	}
```

with:

```go
	var profile *dtos.Profile
	if err := s.scrapeWithRetry(r.Context(), func(cookie string) (bool, error) {
		p, stale, err := s.Profile(r.Context(), cookie)
		if err != nil {
			return false, err
		}
		profile = p
		return stale, nil
	}); err != nil {
		errors.Render(w, r, err)
		return
	}
```

The `cookie` variable previously read from context at the top of `ProfileHandler` is now unused — delete that line from the `var` block (the closure receives the cookie). Keep `logger` if still used by the encode-error path.

- [ ] **Step 3: Build & vet**

Run: `go build ./... && go vet ./internal/server/`
Expected: no output. (If `cookie declared and not used`, remove the leftover context read.)

- [ ] **Step 4: Commit**

```bash
git add internal/server/profile.go internal/server/profile.service.go
git commit -m "feat(profile): stale-session recovery via scrapeWithRetry"
```

---

## Task 9: Integrate retry into `schedule.go` (worker pool)

The dropdown fetch and all workers must share one `*atomic.Bool`; the whole flow runs inside `scrapeWithRetry`.

**Files:**
- Modify: `internal/server/schedule.go`

**Interfaces:**
- Modifies: `scheduleWorker(jobs, results, cookie string, stale *atomic.Bool)` and `processSchedulesWithWorkerPool(queries, names []string, cookie string, stale *atomic.Bool) ([]dtos.ScheduleResponse, error)` — both gain a `stale *atomic.Bool` param.

- [ ] **Step 1: Thread `stale` through the worker + pool**

In `scheduleWorker`, add `stale *atomic.Bool` to the signature, and after `c.WithTransport(...)` add `detectStale(c, stale)`. In `processSchedulesWithWorkerPool`, add `stale *atomic.Bool` to the signature and pass it into each `go s.scheduleWorker(jobs, results, cookie, stale)`.

- [ ] **Step 2: Wrap the handler body in `scrapeWithRetry`**

In `ScheduleHandler`, the real-user path (after the fake-user `if` block) builds the dropdown collector, visits `ImaluumSchedulePage`, filters sessions, then calls `processSchedulesWithWorkerPool`. Wrap from the dropdown-collector build through the worker-pool call in:

```go
	var schedules []dtos.ScheduleResponse
	if err := s.scrapeWithRetry(r.Context(), func(cookie string) (bool, error) {
		var stale atomic.Bool
		sessionQueries = sessionQueries[:0]
		sessionNames = sessionNames[:0]

		c := colly.NewCollector()
		c.WithTransport(s.httpClient.Transport)
		detectStale(c, &stale)
		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set("Cookie", "MOD_AUTH_CAS="+cookie)
			r.Headers.Set("User-Agent", cuid.New())
		})
		c.OnHTML(".box.box-primary .box-header.with-border .dropdown ul.dropdown-menu", func(e *colly.HTMLElement) {
			sessionQueries = e.ChildAttrs("li[style*='font-size:16px'] a", "href")
			sessionNames = e.ChildTexts("li[style*='font-size:16px'] a")
		})
		if err := c.Visit(constants.ImaluumSchedulePage); err != nil {
			return false, errors.ErrFailedToGoToURL
		}
		if stale.Load() {
			return true, nil
		}

		filteredQueries := make([]string, 0, len(sessionQueries))
		filteredNames := make([]string, 0, len(sessionNames))
		for i := range sessionQueries {
			if !slices.Contains(UnwantedSessionQueries[:], sessionQueries[i]) {
				filteredQueries = append(filteredQueries, sessionQueries[i])
				filteredNames = append(filteredNames, sessionNames[i])
			}
		}
		if len(filteredQueries) == 0 {
			logger.ErrorContext(r.Context(), "No valid sessions found")
			return false, errors.ErrScheduleIsEmpty
		}

		result, err := s.processSchedulesWithWorkerPool(filteredQueries, filteredNames, cookie, &stale)
		if err != nil {
			return false, err
		}
		if stale.Load() {
			return true, nil
		}
		schedules = result
		return false, nil
	}); err != nil {
		logger.ErrorContext(r.Context(), "Failed to scrape schedule", "error", err)
		errors.Render(w, r, err)
		return
	}

	if len(schedules) == 0 {
		logger.ErrorContext(r.Context(), "Schedule is empty")
		errors.Render(w, r, errors.ErrScheduleIsEmpty)
		return
	}
```

Notes:
- Remove the now-unused standalone `cookieStr` in the handler if present.
- `sessionQueries`/`sessionNames` stay in the handler `var` block; reset at closure top.
- Sort + response build + encode after the `len(schedules) == 0` check are unchanged.
- Add `"sync/atomic"` to imports.

- [ ] **Step 3: Build & vet**

Run: `go build ./... && go vet ./internal/server/`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add internal/server/schedule.go
git commit -m "feat(schedule): stale-session recovery across worker-pool scrape"
```

---

## Task 10: Integrate retry into `result.go` (worker pool)

Mirror of Task 9 for results.

**Files:**
- Modify: `internal/server/result.go`

**Interfaces:**
- Modifies: the result worker and `processResultsWithWorkerPool(queries, names []string, cookie string, stale *atomic.Bool) (...)` — add `stale *atomic.Bool`.

- [ ] **Step 1: Thread `stale` through the result worker + pool**

Add `stale *atomic.Bool` to the result worker signature; `detectStale(c, stale)` after `c.WithTransport`. Add `stale *atomic.Bool` to `processResultsWithWorkerPool` and pass it to each worker goroutine.

- [ ] **Step 2: Wrap the real-user handler body in `scrapeWithRetry`**

In `ResultHandler` (after the fake-user path that returns canned data), wrap the dropdown-collector build through `processResultsWithWorkerPool` exactly as in Task 9, substituting:
- accumulators reset: `sessionQueries = sessionQueries[:0]`, `sessionNames = sessionNames[:0]`
- dropdown URL: `constants.ImaluumResultPage`
- empty-session error: `errors.ErrResultIsEmpty` (logged "No valid sessions found")
- pool call: `s.processResultsWithWorkerPool(filteredQueries, filteredNames, cookie, &stale)`
- assign to outer `var results []dtos.ResultResponse`
- post-scrape check unchanged: `if len(results) == 0 { ... ErrResultIsEmpty ... }`, then sort + encode.

Add `"sync/atomic"` to imports; remove any now-unused `cookieStr`.

- [ ] **Step 3: Build & vet**

Run: `go build ./... && go vet ./internal/server/`
Expected: no output.

- [ ] **Step 4: Run the full test suite**

Run: `go test ./...`
Expected: PASS (including `TestInvalidate_ForcesRefresh`, `TestRunWithRetry`).

- [ ] **Step 5: Commit**

```bash
git add internal/server/result.go
git commit -m "feat(result): stale-session recovery across worker-pool scrape"
```

---

## Final verification

- [ ] `go build ./...` — clean
- [ ] `go vet ./...` — only the pre-existing generated `internal/proto/*_easyjson.go` lock-by-value warnings
- [ ] `go test ./...` — PASS
- [ ] Spot-check one scraper diff: accumulator reset present, `detectStale` wired, cookie sourced from the closure param, single retry path intact.
