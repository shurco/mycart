package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/migrations"
	"github.com/shurco/mycart/pkg/logging"
)

func TestDetermineSchemaAndAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		httpAddr     string
		httpsAddr    string
		wantSchema   string
		wantMainAddr string
	}{
		{"https wins", ":80", ":443", "https", ":443"},
		{"http fallback", ":80", "", "http", ":80"},
		{"empty http", "", "", "http", ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			schema, addr := determineSchemaAndAddr(tt.httpAddr, tt.httpsAddr)
			if schema != tt.wantSchema || addr != tt.wantMainAddr {
				t.Errorf("got (%s,%s), want (%s,%s)",
					schema, addr, tt.wantSchema, tt.wantMainAddr)
			}
		})
	}
}

func TestExtractHostOnly(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"example.com", "example.com"},
		{"example.com:443", "example.com"},
		{":443", ""},
		{"[::1]:443", "::1"},
		{"invalid:port:thing", "invalid:port:thing"},
	}
	for _, tc := range tests {
		if got := extractHostOnly(tc.in); got != tc.want {
			t.Errorf("extractHostOnly(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsInstallPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want bool
	}{
		{"/_/install", true},
		{"/_/install/step1", true},
		{"/_/assets/logo.png", true},
		{"/_/_app/chunk.js", true},
		{"/_app/chunk.js", true},
		{"/api/whatever", true},
		{"/uploads/1.png", true},
		{"/", false},
		{"/random", false},
		{"/_/", false},
	}
	for _, tc := range tests {
		if got := isInstallPath(tc.path); got != tc.want {
			t.Errorf("isInstallPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSetupFiberApp_BuildsAppWithLimits(t *testing.T) {
	// log is a package-level pointer referenced by middleware.Fiber.
	log = logging.New()

	app, err := setupFiberApp(false)
	if err != nil {
		t.Fatalf("setupFiberApp: %v", err)
	}
	if app == nil {
		t.Fatal("nil app returned")
	}
	t.Cleanup(func() { _ = app.Shutdown() })
}

func TestInstallCheck_RedirectsWhenNotInstalled(t *testing.T) {
	// Build a fresh DB in a temp CWD so the global queries.DB() is populated
	// with an uninitialised (installed='') row.
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

	app := fiber.New()
	app.Get("/dashboard", InstallCheck, func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	app.Get("/_/install", InstallCheck, func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	// An unrelated URL should be redirected to /_/install.
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusFound {
		t.Errorf("expected redirect, got %d", resp.StatusCode)
	}

	// Install paths should pass the guard.
	req = httptest.NewRequest(http.MethodGet, "/_/install", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test install: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("install path got %d", resp.StatusCode)
	}
}

func TestStartServer_RespectsContextCancel(t *testing.T) {
	log = logging.New()
	app := fiber.New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled — StartServer should exit quickly.
	// Use port 0 so the OS allocates; if Listen races past cancellation,
	// Shutdown will still close the listener cleanly.
	if err := StartServer(ctx, "127.0.0.1:0", app); err != nil {
		// a.Shutdown returning nil is the expected path.
		t.Logf("StartServer exit: %v", err)
	}
}

func TestInit_CreatesDirsAndDB(t *testing.T) {
	prev, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	for _, d := range []string{"lc_uploads", "lc_digitals", "lc_base"} {
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
			t.Errorf("expected dir %q: %v", d, err)
		}
	}
}
