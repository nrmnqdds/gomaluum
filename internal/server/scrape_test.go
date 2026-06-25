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
