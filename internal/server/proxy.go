package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/nrmnqdds/gomaluum/internal/constants"
)

// imaluumHost is the only host routed through the residential egress proxy.
var imaluumHost = mustHost(constants.ImaluumPage)

func mustHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(fmt.Sprintf("invalid i-Ma'luum URL constant %q: %v", rawURL, err))
	}
	return u.Hostname()
}

// newImaluumProxyFunc builds a transport Proxy function that routes ONLY
// i-Ma'luum requests through rawProxyURL (a residential rotating proxy, e.g.
// LumiProxy), leaving every other host direct so we don't pay proxy bandwidth
// for GAS/GEI/OTLP/ads traffic. Returns a nil func when rawProxyURL is empty
// (proxy disabled), or an error when it is set but unparseable.
func newImaluumProxyFunc(rawProxyURL string) (func(*http.Request) (*url.URL, error), error) {
	if rawProxyURL == "" {
		return nil, nil
	}
	proxyURL, err := url.Parse(rawProxyURL)
	if err != nil {
		return nil, fmt.Errorf("parsing IMALUUM_PROXY_URL: %w", err)
	}
	return func(r *http.Request) (*url.URL, error) {
		if r.URL.Hostname() == imaluumHost {
			return proxyURL, nil
		}
		return nil, nil
	}, nil
}
