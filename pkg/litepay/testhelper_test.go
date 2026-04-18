package litepay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// rewritingTransport redirects requests whose URL starts with one of
// `prefixes` to the given test server. All other requests are rejected with
// an error so tests surface accidental outbound traffic.
type rewritingTransport struct {
	prefixes []string
	server   *httptest.Server
}

func (r *rewritingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	origURL := req.URL.String()
	for _, p := range r.prefixes {
		if strings.HasPrefix(origURL, p) {
			u, err := req.URL.Parse(r.server.URL + req.URL.Path)
			if err != nil {
				return nil, err
			}
			req = req.Clone(req.Context())
			req.URL = u
			req.Host = ""
			return r.server.Client().Transport.RoundTrip(req)
		}
	}
	return nil, &unexpectedHostError{url: origURL}
}

type unexpectedHostError struct{ url string }

func (e *unexpectedHostError) Error() string { return "unexpected outbound call to " + e.url }

// redirectHTTPClient installs a temporary httpClient whose outbound requests
// matching any of the given prefixes land on srv. The original client is
// restored on cleanup. Tests that use this helper MUST NOT run in parallel
// because httpClient is a package-level variable.
func redirectHTTPClient(t *testing.T, srv *httptest.Server, prefixes ...string) {
	t.Helper()
	prev := httpClient
	t.Cleanup(func() { httpClient = prev })
	httpClient = &http.Client{
		Timeout:   prev.Timeout,
		Transport: &rewritingTransport{prefixes: prefixes, server: srv},
	}
}
