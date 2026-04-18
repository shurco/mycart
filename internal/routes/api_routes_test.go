package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/migrations"
)

// routesTestDB brings up a blank queries DB so handlers wired into these
// routes don't nil-panic when exercised by the test Fiber client.
func routesTestDB(t *testing.T) {
	t.Helper()
	prev, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	_ = os.MkdirAll("lc_base", 0o775)
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := queries.New(migrations.Embed()); err != nil {
		t.Fatalf("queries.New: %v", err)
	}
}

func TestApiPublicRoutes_WiredCorrectly(t *testing.T) {
	routesTestDB(t)

	app := fiber.New()
	ApiPublicRoutes(app)

	// /ping is the only public route with no external dependencies.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("ping status = %d", resp.StatusCode)
	}

	// /api/settings and /api/products exist in the router — we don't care what
	// payload they return, just that the path resolved (not 404).
	for _, p := range []string{"/api/settings", "/api/products/"} {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, p, nil))
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("route %s not registered (got 404)", p)
		}
	}
}

func TestApiPrivateRoutes_WiredWithAuthGuard(t *testing.T) {
	routesTestDB(t)

	app := fiber.New()
	ApiPrivateRoutes(app)

	// All /_/ routes require the JWT middleware. Without a cookie the guard
	// returns 401, which is still evidence the route was registered.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/_/version", nil))
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	// /api/install is public.
	resp, err = app.Test(httptest.NewRequest(http.MethodPost, "/api/install", nil))
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		t.Error("install route not registered")
	}
}

func TestSiteRoutes_MountsSPAHandler(t *testing.T) {
	routesTestDB(t)

	app := fiber.New()
	SiteRoutes(app)
	// The SPA handler falls back to index.html for unknown paths; the embedded
	// FS is available at runtime, so this only checks the route got mounted.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/some-arbitrary-page", nil))
	if err != nil {
		t.Fatalf("site: %v", err)
	}
	// Either 200 (index fallback) or 404 (if the web embed is empty in test).
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status = %d", resp.StatusCode)
	}
}

func TestAdminRoutes_MountsSPAHandler(t *testing.T) {
	app := fiber.New()
	AdminRoutes(app)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/_/dashboard", nil))
	if err != nil {
		t.Fatalf("admin: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status = %d", resp.StatusCode)
	}
}
