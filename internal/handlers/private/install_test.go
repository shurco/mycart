package handlers

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/pressly/goose/v3"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/testutil"
	"github.com/shurco/mycart/migrations"
	_ "modernc.org/sqlite"
)

func setupCleanDB(t *testing.T) (*fiber.App, func()) {
	t.Helper()
	dirCleanup := testutil.WithCmdTestDir(t)

	sqlite, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatal(err)
	}
	sqlite.SetMaxOpenConns(1)

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(migrations.Embed())
	goose.SetTableName("migrate_db_version")
	if err := goose.Up(sqlite, "."); err != nil {
		t.Fatal(err)
	}

	queries.NewFromDB(sqlite)
	app := fiber.New()

	return app, func() {
		_ = app.Shutdown()
		_ = sqlite.Close()
		dirCleanup()
	}
}

func TestInstall(t *testing.T) {
	app, cleanup := setupCleanDB(t)
	defer cleanup()

	app.Post("/api/install", Install)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			"invalid email",
			`{"email":"bad","password":"secret","domain":"example.com"}`,
			http.StatusBadRequest,
		},
		{
			"short password",
			`{"email":"admin@example.com","password":"12","domain":"example.com"}`,
			http.StatusBadRequest,
		},
		{
			"empty body",
			`{}`,
			http.StatusBadRequest,
		},
		{
			"valid install (last — mutates DB)",
			`{"email":"admin@example.com","password":"secret","domain":"example.com"}`,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodPost, "/api/install", tt.body, "")
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
