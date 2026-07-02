package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewImaluumProxyFunc(t *testing.T) {
	t.Run("empty disables proxy", func(t *testing.T) {
		fn, err := newImaluumProxyFunc("")
		require.NoError(t, err)
		require.Nil(t, fn)
	})

	t.Run("routes only i-Ma'luum through the proxy", func(t *testing.T) {
		fn, err := newImaluumProxyFunc("http://user:pass@as.lumiproxy.io:5888")
		require.NoError(t, err)
		require.NotNil(t, fn)

		// i-Ma'luum (with port) -> proxied.
		req, _ := http.NewRequest("GET", "https://imaluum.iium.edu.my:443/MyAcademic/schedule", nil)
		p, err := fn(req)
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Equal(t, "as.lumiproxy.io:5888", p.Host)
		require.Equal(t, "user", p.User.Username())

		// Any other host -> direct (nil).
		for _, u := range []string{"https://gas.quddus.my/x", "https://souq.iium.edu.my/embeded"} {
			other, _ := http.NewRequest("GET", u, nil)
			p, err := fn(other)
			require.NoError(t, err)
			require.Nil(t, p, u)
		}
	})

	t.Run("invalid proxy url errors", func(t *testing.T) {
		_, err := newImaluumProxyFunc("://not a url")
		require.Error(t, err)
	})
}
