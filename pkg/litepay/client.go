package litepay

import (
	"net/http"

	"github.com/shurco/mycart/pkg/httpclient"
)

// httpClient is the shared HTTP client used by every payment provider in this
// package. Using a package-level client (rather than per-call http.DefaultClient)
// guarantees a hard timeout, reuses the connection pool, and makes outbound
// payment calls observable from a single chokepoint.
var httpClient *http.Client = httpclient.New()
