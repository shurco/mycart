package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// rewriteTransport hijacks outbound requests so we can point the
// production code at an httptest.Server without changing the public API.
type rewriteTransport struct {
	base    http.RoundTripper
	rewrite func(*http.Request) *http.Request
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.base.RoundTrip(r.rewrite(req))
}

// installFakeGitHub redirects GitHub API calls to the provided test server.
// The previous releaseClient is restored on cleanup.
func installFakeGitHub(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prev := releaseClient
	t.Cleanup(func() { releaseClient = prev })

	releaseClient = &http.Client{
		Timeout: prev.Timeout,
		Transport: &rewriteTransport{
			base: http.DefaultTransport,
			rewrite: func(req *http.Request) *http.Request {
				if strings.HasPrefix(req.URL.String(), "https://api.github.com/") {
					u, _ := req.URL.Parse(srv.URL + req.URL.Path)
					clone := req.Clone(req.Context())
					clone.URL = u
					clone.Host = ""
					return clone
				}
				return req
			},
		},
	}
}

func TestFetchLatestRelease_HappyPath(t *testing.T) {
	// NOT t.Parallel — mutates package-level releaseClient.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 1,
			"name": "v1.2.3",
			"tag_name": "v1.2.3",
			"html_url": "https://example/v1.2.3",
			"assets": [
				{"name": "app_linux-amd64.tar.gz", "browser_download_url": "https://example/a"}
			]
		}`))
	}))
	t.Cleanup(srv.Close)
	installFakeGitHub(t, srv)

	rel, err := FetchLatestRelease(context.Background(), "shurco", "mycart")
	if err != nil {
		t.Fatalf("FetchLatestRelease: %v", err)
	}
	if rel.Name != "v1.2.3" || rel.Tag != "v1.2.3" {
		t.Errorf("unexpected release: %+v", rel)
	}
	if len(rel.Assets) != 1 || rel.Assets[0].Name != "app_linux-amd64.tar.gz" {
		t.Errorf("assets wrong: %+v", rel.Assets)
	}
}

func TestFetchLatestRelease_Non2xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	t.Cleanup(srv.Close)
	installFakeGitHub(t, srv)

	_, err := FetchLatestRelease(context.Background(), "shurco", "missing")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error does not mention status: %v", err)
	}
}

func TestFetchLatestRelease_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	t.Cleanup(srv.Close)
	installFakeGitHub(t, srv)

	if _, err := FetchLatestRelease(context.Background(), "a", "b"); err == nil {
		t.Fatal("expected JSON unmarshal error")
	}
}

func TestReleaseInfo_NoUpgradeNeeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"name":"v1.0.0","tag_name":"v1.0.0","assets":[]}`))
	}))
	t.Cleanup(srv.Close)
	installFakeGitHub(t, srv)

	asset, err := ReleaseInfo(context.Background(), &Config{
		Owner: "a", Repo: "b", CurrentVersion: "v1.0.0",
	})
	if err != nil {
		t.Fatalf("ReleaseInfo: %v", err)
	}
	if asset != nil {
		t.Errorf("expected nil asset when versions match, got %+v", asset)
	}
}
