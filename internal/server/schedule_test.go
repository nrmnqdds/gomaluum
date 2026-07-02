package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLatestSessionQuery(t *testing.T) {
	t.Run("picks the newest session regardless of dropdown order", func(t *testing.T) {
		queries := []string{
			"?ses=2023/2024&sem=1",
			"?ses=2024/2025&sem=2",
			"?ses=2024/2025&sem=1",
		}
		names := []string{
			"Sem 1, 2023/2024",
			"Sem 2, 2024/2025",
			"Sem 1, 2024/2025",
		}
		require.Equal(t, "?ses=2024/2025&sem=2", latestSessionQuery(queries, names))
	})

	t.Run("single session", func(t *testing.T) {
		require.Equal(t,
			"?ses=2024/2025&sem=1",
			latestSessionQuery([]string{"?ses=2024/2025&sem=1"}, []string{"Sem 1, 2024/2025"}),
		)
	})
}
