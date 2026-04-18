// Package httpclient centralizes HTTP client creation for outbound calls.
//
// All external HTTP requests (GitHub release fetch, payment gateways, webhook
// delivery, etc.) MUST use this package instead of http.DefaultClient. The
// stdlib default has no timeout and will hang the caller indefinitely if a
// remote endpoint stops responding mid-stream — this has real operational
// impact on a payment path.
package httpclient

import (
	"net"
	"net/http"
	"time"
)

// DefaultTimeout is the overall request timeout (connect + TLS + headers + body).
// Chosen as a balance between typical payment-gateway latency (≈1s) and
// catastrophic stalls. CI runners can have slow TLS handshakes against
// third-party sandboxes, so the ceiling is intentionally generous.
const DefaultTimeout = 30 * time.Second

// New returns a *http.Client with sensible production defaults. The returned
// client is safe for concurrent use and may be reused across goroutines.
func New() *http.Client {
	return NewWithTimeout(DefaultTimeout)
}

// NewWithTimeout returns a *http.Client with a caller-provided overall timeout.
// Use NewWithTimeout for endpoints that are known to be legitimately slow
// (e.g. large binary downloads) and DefaultTimeout otherwise.
func NewWithTimeout(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: newTransport(),
	}
}

// newTransport builds a *http.Transport with connection pooling and aggressive
// per-phase timeouts. Returning a fresh transport per client (rather than
// reusing http.DefaultTransport) lets callers tune timeouts without mutating
// process-global state.
func newTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
	}
}
