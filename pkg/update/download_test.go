package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// installFakeDownload swaps the package-level downloadClient for a client
// that always resolves to the supplied test server, independent of URL.
func installFakeDownload(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prev := downloadClient
	t.Cleanup(func() { downloadClient = prev })

	downloadClient = &http.Client{
		Timeout: prev.Timeout,
		Transport: &rewriteTransport{
			base: http.DefaultTransport,
			rewrite: func(req *http.Request) *http.Request {
				u, _ := req.URL.Parse(srv.URL + req.URL.Path)
				clone := req.Clone(req.Context())
				clone.URL = u
				clone.Host = ""
				return clone
			},
		},
	}
}

func TestDownloadFile_HappyPath(t *testing.T) {
	payload := "hello world"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(srv.Close)
	installFakeDownload(t, srv)

	dest := filepath.Join(t.TempDir(), "nested", "dir", "out.bin")
	if err := downloadFile(context.Background(), "http://fake/download.bin", dest); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(b) != payload {
		t.Errorf("payload mismatch: %q", b)
	}
}

func TestDownloadFile_Non2xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	installFakeDownload(t, srv)

	dest := filepath.Join(t.TempDir(), "out.bin")
	err := downloadFile(context.Background(), "http://fake/download.bin", dest)
	if err == nil {
		t.Fatal("expected error on 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected status in error, got %v", err)
	}
}

func TestDownloadFile_BadRequestURL(t *testing.T) {
	// http.NewRequestWithContext fails fast on URLs with control chars.
	err := downloadFile(context.Background(), "http://\x7f/", "/tmp/whatever")
	if err == nil {
		t.Fatal("expected request construction error")
	}
}
