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
