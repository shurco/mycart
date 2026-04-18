package webhook

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/shurco/mycart/pkg/httpclient"
)

// sharedClient is intentionally module-scoped: *http.Client is safe for
// concurrent use and reusing a single transport avoids creating a new
// connection pool (and DNS cache) for every webhook delivery.
var sharedClient = httpclient.New()

// Send sends an HTTP POST request to the specified URL with the given JSON payload.
// The request is bounded by httpclient.DefaultTimeout, which prevents a slow
// webhook receiver from wedging the calling payment handler.
func Send(url string, payload []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := sharedClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send webhook: %w", err)
	}
	return resp, nil
}
