# Stale-session recovery for scrapers

Date: 2026-06-24
Status: approved (design)

> Spec lives under `specs/` (not `docs/`) because `main.go` does
> `//go:embed docs/*` — anything under `docs/` is baked into the binary.

## Problem

Caching the i-Ma'luum session cookie for 30 minutes (the
`imaluumSessionTTL` fix) introduced a stale-cookie window. The chain:

1. `DecodePasetoToken` → `TokenManager.GetToken` returns a cached cookie if the
   entry is < 30m old, **without re-authenticating**.
2. If i-Ma'luum killed that session early (user logged in elsewhere, server-side
   timeout shorter than expected), the cached cookie is stale.
3. The scraper uses it; i-Ma'luum 302-redirects to the CAS login page; colly
   follows the redirect and scrapes the **login page**.
4. Parsers match nothing → empty slice → the handler returns `"… is empty"` for
   **up to 30 minutes**, with no self-healing.

The GAS auth service (`~/Github/gas`) does a full, stateless CAS login on every
call and has no session validation — so *when* to re-login is entirely
gomaluum's decision, and today that decision is purely time-based (30m).

## Goals

- Detect when a scrape was bounced to the CAS login page (stale session).
- On detection: evict the cached session, re-login, and retry the scrape **once**
  with the fresh cookie.
- Cover the 7 authenticated colly scrapers.
- Keep the change targeted (no full collector-factory refactor).

## Non-goals

- The `download/*` PDF endpoints (different, non-colly fetch path) — follow-up.
- `ads` (public, no cookie) and `academic_calendar` (embedded JSON) — no session.
- Proactive cookie validation (would cost an extra request per cache hit).

## Prerequisite fixes (in the path we're refactoring)

These are latent bugs in `DecodePasetoToken` that `refreshSession` would inherit:

- **A. Normalize `TokenPayload.password` to plaintext on both return paths.**
  The expired path returns `string(decodedPassword)` (plaintext); the
  non-expired path returns `password` (still base64). `refreshSession` uses
  `payload.password` for the gRPC `Login`, so it must always be plaintext.
  Fix: base64-decode in the non-expired path before returning.
- **B. Remove `log.Fatal(err)` on `GetToken` failure** (paseto.go ~line 168). A
  single failed login currently crashes the whole server. Return the error
  instead. `refreshSession` likewise returns errors, never exits.

## Components

### 1. `TokenManager.Invalidate(matric string)` — `pkg/sf`

```go
func (tm *TokenManager) Invalidate(matric string) {
    tm.mu.Lock()
    delete(tm.tokens, matric)
    tm.mu.Unlock()
}
```

Evicts a user's cached entry so the next `GetToken` is forced to re-login.

### 2. Shared login closure + `s.refreshSession`

Factor the existing `refresh` closure in `DecodePasetoToken` into a reusable
login function so both the normal refresh path and `refreshSession` share it
(including the debug-user short-circuit and the 30m TTL):

```go
func (s *Server) loginFunc(ctx context.Context, username, password string) func() (string, time.Time, error)

func (s *Server) refreshSession(ctx context.Context, username, password string) (string, error) {
    s.tokenManager.Invalidate(username)
    return s.tokenManager.GetToken(username, s.loginFunc(ctx, username, password))
}
```

`Invalidate` + `GetToken` guarantees a fresh login; singleflight collapses a
burst of concurrent stale-detecting requests for the same user into one login.

### 3. Session in context

The auth middleware already stores the cookie under `ctxToken`. Additionally
store the decoded `*TokenPayload` under a new `ctxSession` key (it carries
`username` + plaintext `password`). `ctxToken` stays unchanged so
`downloads.go` and `LogoutHandler` are untouched.

### 4. `detectStale(c *colly.Collector, stale *atomic.Bool)`

```go
func detectStale(c *colly.Collector, stale *atomic.Bool) {
    c.OnHTML(`input[name="password"]`, func(*colly.HTMLElement) {
        stale.Store(true)
    })
}
```

The CAS login page contains a password input; authenticated i-Ma'luum data
pages do not. Each scraper calls this once, right after creating its collector.

### 5. Retry helper (pure core + method wiring)

Keep the retry logic as a free function so it is unit-testable without gRPC:

```go
// runWithRetry runs fn with cookie; on stale, calls refresh and retries once.
func runWithRetry(
    cookie string,
    refresh func() (string, error),
    fn func(cookie string) (stale bool, err error),
) error {
    stale, err := fn(cookie)
    if err != nil { return err }
    if !stale { return nil }

    cookie, err = refresh()
    if err != nil { return err }

    stale, err = fn(cookie)
    if err != nil { return err }
    if stale { return errors.ErrStaleSession }
    return nil
}

func (s *Server) scrapeWithRetry(ctx context.Context, fn func(cookie string) (bool, error)) error {
    sess := ctx.Value(ctxSession).(*TokenPayload)
    return runWithRetry(sess.imaluumCookie,
        func() (string, error) { return s.refreshSession(ctx, sess.username, sess.password) },
        fn)
}
```

### 6. `errors.ErrStaleSession`

New sentinel for "still bounced to login after a fresh re-login" (rare: bad
creds or i-Ma'luum issue). Rendered as an auth-level failure.

## Per-scraper change pattern

Applies to: **schedule, result, carry_mark, disciplinary, final_exam,
starpoint, profile** (profile via `Profile(ctx, cookie)`).

**Single-collector** (carry_mark, disciplinary, final_exam, starpoint, profile):

```go
err := s.scrapeWithRetry(r.Context(), func(cookie string) (bool, error) {
    var stale atomic.Bool
    subjects = subjects[:0]            // RESET accumulator for clean retry
    c := colly.NewCollector()
    c.WithTransport(s.httpClient.Transport)
    detectStale(c, &stale)
    c.OnRequest(/* Cookie: MOD_AUTH_CAS=<cookie>, UA */)
    c.OnHTML(/* existing parse */)
    if err := c.Visit(page); err != nil { return false, errors.ErrFailedToGoToURL }
    return stale.Load(), nil
})
// existing empty/encode handling stays
```

**Profile** is a special single-collector case: the collector lives inside
`Profile(ctx, cookie)`, not the handler. Change its signature to
`Profile(ctx, cookie) (*dtos.Profile, bool, error)` (returns `stale`); the
handler wraps it: `profile, stale, err = s.Profile(ctx, cookie); return stale, err`.

**Worker-pool** (schedule, result): the dropdown collector and all worker
collectors share one `*atomic.Bool`; the whole dropdown→fan-out runs inside the
`scrapeWithRetry` closure, so a retry re-fetches everything with the fresh
cookie. `processSchedulesWithWorkerPool` / `processResultsWithWorkerPool` and
their workers take the `*atomic.Bool` and the cookie as parameters.

## Error handling

- `c.Visit` transport error → existing `ErrFailedToGoToURL` (no retry; not a
  stale-session signal).
- Stale on first attempt → refresh + retry (logged at Info: "stale session,
  re-authenticating", with `username`).
- Stale after retry → `ErrStaleSession` rendered to the user.
- `refreshSession` / login error → returned and rendered, never `log.Fatal`.

## Testing

- `TokenManager.Invalidate`: entry removed; next `GetToken` calls refresh.
- `runWithRetry` (pure, no gRPC): not-stale → no refresh, one call; stale-then-ok
  → refresh called once, two calls, nil; stale-twice → `ErrStaleSession`, refresh
  once; `fn` error / `refresh` error → propagated, no further calls.
- Accumulator reset: a retry does not duplicate first-attempt rows (table test on
  one single-collector scraper).

## Risks / notes

- **Retry once only** — avoids hammering CAS when credentials are genuinely bad.
- **Accumulator reset is mandatory** in every closure, else a retry appends to
  first-attempt data. Called out per scraper.
- Detection assumes no authenticated i-Ma'luum data page contains
  `input[name="password"]`; true for current scraped pages.
- Pairs with the Tier-1 scrape `empty` metric: stale that slips through (e.g.
  i-Ma'luum changes the login markup) still shows up as an `empty` spike.
