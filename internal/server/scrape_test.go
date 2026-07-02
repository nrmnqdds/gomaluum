package server

import (
	"context"
	"errors"
	"net/http"
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

func TestScrapeWithRetry_MissingSession(t *testing.T) {
	s := &Server{}
	called := false
	err := s.scrapeWithRetry(context.Background(), func(string) (bool, error) {
		called = true
		return false, nil
	})
	require.ErrorIs(t, err, apperrors.ErrInvalidToken)
	require.False(t, called)
}

func TestIsTruthyParam(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"0", false},
		{"false", false},
		{"", false},
		{"yes", false},
	} {
		require.Equal(t, tc.want, isTruthyParam(tc.in), "input %q", tc.in)
	}
}

func TestClassifyVisitError(t *testing.T) {
	t.Run("403 Forbidden: maps to upstream forbidden (502)", func(t *testing.T) {
		// colly reports a non-2xx as errors.New(http.StatusText(code)).
		collyErr := errors.New(http.StatusText(http.StatusForbidden))
		got := classifyVisitError(collyErr)
		require.Equal(t, apperrors.ErrUpstreamForbidden.Message, got.Message)
		require.Equal(t, 502, got.GetStatusCode())
		require.ErrorIs(t, got.OriginalErr, collyErr)
	})

	t.Run("other status: stays generic failure (500)", func(t *testing.T) {
		collyErr := errors.New(http.StatusText(http.StatusInternalServerError))
		got := classifyVisitError(collyErr)
		require.Equal(t, apperrors.ErrFailedToGoToURL.Message, got.Message)
		require.Equal(t, 500, got.GetStatusCode())
	})

	t.Run("transport error: stays generic failure (500)", func(t *testing.T) {
		got := classifyVisitError(errors.New("dial tcp: connection refused"))
		require.Equal(t, apperrors.ErrFailedToGoToURL.Message, got.Message)
		require.Equal(t, 500, got.GetStatusCode())
	})
}
