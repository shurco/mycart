package testutil

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/pressly/goose/v3"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/migrations"
	"github.com/shurco/mycart/pkg/jwtutil"
	_ "modernc.org/sqlite"
)

const (
	FixtureJWTSecret = "d58ca30c8e5ca96695451fa27af949d9"
	FixtureEmail     = "user@mail.com"
	FixturePassword  = "Pass123"
)

// projectRoot returns the absolute path to the repository root,
// derived from this source file's location at compile time.
func projectRoot() string {
	_, src, _, _ := runtime.Caller(0)
	// src = <root>/internal/testutil/testdb.go → root = ../../
	return filepath.Join(filepath.Dir(src), "..", "..")
}

// SetupTestDB creates in-memory SQLite, runs schema migrations + fixtures,
// sets queries.DB global. Returns cleanup function.
func SetupTestDB(t *testing.T) func() {
	t.Helper()

	dirCleanup := WithCmdTestDir(t)

	sqlite, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	sqlite.SetMaxOpenConns(1)

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}

	goose.SetBaseFS(migrations.Embed())
	goose.SetTableName("migrate_db_version")
	if err := goose.Up(sqlite, "."); err != nil {
		t.Fatalf("run schema migrations: %v", err)
	}

	fixturesDir := filepath.Join(projectRoot(), "fixtures")
	goose.SetBaseFS(os.DirFS(fixturesDir))
	goose.SetTableName("migrate_fixtures_version")
	if err := goose.Up(sqlite, "migration"); err != nil {
		t.Fatalf("run fixtures: %v", err)
	}

	queries.NewFromDB(sqlite)

	return func() {
		_ = sqlite.Close()
		dirCleanup()
	}
}

// SetupTestApp creates in-memory DB with fixtures, Fiber app, and JWT cookie.
// Fixtures already contain installed state + JWT secret.
func SetupTestApp(t *testing.T) (app *fiber.App, cookie string, cleanup func()) {
	t.Helper()

	dbCleanup := SetupTestDB(t)
	app = fiber.New()

	exp := time.Now().Add(time.Hour).Unix()
	tok, err := jwtutil.GenerateNewToken(FixtureJWTSecret, "test-user-id", exp, nil)
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	return app, "token=" + tok, func() {
		_ = app.Shutdown()
		dbCleanup()
	}
}

// DoRequest is a DRY helper for table-driven HTTP handler tests.
func DoRequest(t *testing.T, app *fiber.App, method, path, body, cookie string) *http.Response {
	t.Helper()

	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// AssertStatus checks HTTP status and closes body.
func AssertStatus(t *testing.T, resp *http.Response, want ...int) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	for _, w := range want {
		if resp.StatusCode == w {
			return
		}
	}
	t.Errorf("status = %d, want one of %v", resp.StatusCode, want)
}
