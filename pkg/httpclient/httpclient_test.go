package httpclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew_AppliesDefaultTimeout(t *testing.T) {
	t.Parallel()

	c := New()
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.Timeout != DefaultTimeout {
		t.Errorf("Timeout: got %s, want %s", c.Timeout, DefaultTimeout)
	}
	if c.Transport == nil {
		t.Error("Transport should not be nil; must not rely on http.DefaultTransport")
	}
}

func TestNewWithTimeout_TimesOutOnSlowServer(t *testing.T) {
	t.Parallel()

	// Server that intentionally hangs past the client timeout to verify the
	// ceiling is actually enforced. Without the timeout, the test would block
	// forever and be killed by go test's package timeout, not by the client.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := NewWithTimeout(50 * time.Millisecond)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	start := time.Now()
	resp, err := c.Do(req)
	if err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond {
		t.Errorf("request took %s, which is beyond the 50ms ceiling", elapsed)
	}
}

func TestNewWithTimeout_SucceedsBelowCeiling(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := NewWithTimeout(2 * time.Second)
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}
