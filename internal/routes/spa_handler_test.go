package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", indexHTML},
		{"/", indexHTML},
		{"/foo/bar", "foo/bar"},
		{"foo", "foo"},
	}
	for _, tc := range tests {
		if got := normalizePath(tc.in); got != tc.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsStaticAsset(t *testing.T) {
	t.Parallel()
	assets := []string{"app.js", "styles.css", "logo.PNG", "icon.ICO", "bundle.woff2"}
	for _, a := range assets {
		if !isStaticAsset(a) {
			t.Errorf("%s should be a static asset", a)
		}
	}
	nonAssets := []string{"page.html", "index", "data.txt", ""}
	for _, a := range nonAssets {
		if isStaticAsset(a) {
			t.Errorf("%s should not be a static asset", a)
		}
	}
}

func TestGetContentType(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		".html":   "text/html",
		".js":     "application/javascript",
		".CSS":    "text/css",
		".png":    "image/png",
		".jpg":    "image/jpeg",
		".woff2":  "font/woff2",
		".custom": "application/octet-stream",
	}
	for ext, want := range tests {
		if got := getContentType(ext); got != want {
			t.Errorf("getContentType(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestSetupSPAHandler_ServesEmbeddedFileAndIndexFallback(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		indexHTML:    {Data: []byte("<html>index</html>")},
		"app.js":     {Data: []byte("console.log('hi');")},
		"favicon.js": {Data: []byte("ico")},
	}

	app := fiber.New()
	app.Use("/", setupSPAHandler(fsys, func(p string) bool {
		return p == "/skip"
	}, ""))

	// Direct asset lookup.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("app.js status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/javascript" {
		t.Errorf("content-type = %q", ct)
	}

	// Unknown non-asset path falls back to index.html.
	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/some/spa/route", nil))
	if err != nil {
		t.Fatalf("index fallback: %v", err)
	}
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != contentTypeHTML {
		t.Errorf("index fallback status=%d ct=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}

	// Unknown *asset* must 404.
	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/missing.png", nil))
	if err != nil {
		t.Fatalf("missing asset: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing asset status = %d", resp.StatusCode)
	}

	// skipPaths short-circuits and passes to c.Next() — without another handler
	// registered we should fall through to Fiber's default 404.
	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/skip", nil))
	if err != nil {
		t.Fatalf("skip path: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("skip path status = %d", resp.StatusCode)
	}
}

func TestSetupSPAHandler_StripPrefix(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		indexHTML:  {Data: []byte("<html>admin</html>")},
		"admin.js": {Data: []byte("js")},
	}
	app := fiber.New()
	app.Use("/_", setupSPAHandler(fsys, func(string) bool { return false }, "/_"))

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/_/admin.js", nil))
	if err != nil {
		t.Fatalf("admin.js: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestNotFoundRoute_ApiPrefixReturns404(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	NotFoundRoute(app, false)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/unknown", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestNotFoundRoute_NonApiFallsThrough(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	NotFoundRoute(app, false)
	// With no downstream handler, Fiber still returns 404 but from the default
	// 404 handler rather than the explicit webutil response — we care only that
	// the middleware didn't panic and forwarded control.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/_admin", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
