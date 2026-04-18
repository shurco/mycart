package webhook

import (
	"testing"
)

func TestSend_InvalidURLReturnsError(t *testing.T) {
	t.Parallel()
	// http.NewRequest only rejects a small set of malformed URLs (e.g. those
	// containing a NUL byte). Anything else ends up as a DNS/transport
	// failure from Client.Do which is what we want to surface here.
	_, err := Send("http://127.0.0.1:1/no-such-server", []byte(`{"x":1}`))
	if err == nil {
		t.Fatal("expected transport error for unreachable host")
	}
}

func TestSend_RejectsUnparseableURL(t *testing.T) {
	t.Parallel()
	_, err := Send("http://\x7f/", nil)
	if err == nil {
		t.Fatal("expected request construction error")
	}
}
