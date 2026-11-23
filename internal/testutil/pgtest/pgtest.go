// Package pgtest provisions throwaway PostgreSQL databases for the test suite.
//
// Each call hands back a DSN for a database of its own. The schema (and, when
// asked for, the fixtures) is applied to a *template* database that pgtestdb
// builds once per server, and every test database is then cloned from it. The
// migrations therefore run a single time for the whole suite — not once per
// test, and not once per package — while each test still starts from a pristine
// database and never sees another test's rows.
//
// TEST_POSTGRES_DSN is an *administrative* connection: pgtestdb connects with it
// to create the role pgtdbuser, the template database testdb_tpl_*, and one
// testdb_tpl_*_inst_* database per test, and to drop the instances again. Point
// it at a dedicated test server, never at one holding data anybody wants to
// keep. The user it names needs CREATEDB, CREATEROLE and SUPERUSER.
//
// A test that passes has its database dropped; a test that fails keeps its, and
// pgtestdb logs the connection string so the state can be inspected with psql.
//
// The package also owns the fixture filesystem, which the SQLite path shares, so
// that "the fixtures" means one thing for both engines.
package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/migrations"
)

// AdminDSN names the environment variable holding the administrative
// connection string of the PostgreSQL test server.
const AdminDSN = "TEST_POSTGRES_DSN"

// FixturesDir is the directory under the repository root holding the fixture
// migrations.
const FixturesDir = "fixtures"

// FixturesTable is the goose bookkeeping table for the fixtures. It is separate
// from the schema's migrate_db_version on purpose: the fixtures are test data,
// not schema, and folding them into the schema's version history would make a
// fixture version look like an applied migration.
const FixturesTable = "migrate_fixtures_version"

// FixturesFS returns the fixture migrations, rooted at the directory goose
// should read.
//
// The path is anchored to this source file rather than to the working
// directory: the suite runs tests in a temporary directory of their own, so a
// relative path would find nothing.
//
// Developer overlays (*_private.sql, gitignored) are left out: they carry one
// machine's domain and payment credentials for `scripts/migration dev up`, and
// a suite that loaded them would assert against that machine instead of the
// shipped fixtures.
func FixturesFS() fs.FS {
	return publicFixtures{os.DirFS(filepath.Join(repoRoot(), FixturesDir, "migration"))}
}

// privateFixtureSuffix names the developer overlays; it mirrors the .gitignore
// entry fixtures/migration/*_private.sql.
const privateFixtureSuffix = "_private.sql"

// publicFixtures hides the developer overlays from whoever reads the fixtures.
type publicFixtures struct{ fs.FS }

func (f publicFixtures) Open(name string) (fs.File, error) {
	if strings.HasSuffix(name, privateFixtureSuffix) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return f.FS.Open(name)
}

func (f publicFixtures) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.FS, name)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(entries, func(e fs.DirEntry) bool {
		return strings.HasSuffix(e.Name(), privateFixtureSuffix)
	}), nil
}

// EmptyDSN returns the DSN of a fresh database with nothing in it: no tables,
// no bookkeeping. It is for tests of the migration path itself, which have to
// apply the migrations to an empty database the way a real installation does.
func EmptyDSN(t *testing.T) string {
	t.Helper()
	return instance(t, pgtestdb.NoopMigrator{})
}

// MigratedDSN returns the DSN of a fresh database with the embedded migrations
// applied.
func MigratedDSN(t *testing.T) string {
	t.Helper()
	return instance(t, migrator(false))
}

// FixturesDSN returns the DSN of a fresh database with the embedded migrations
// and the fixtures applied.
func FixturesDSN(t *testing.T) string {
	t.Helper()
	return instance(t, migrator(true))
}

// instance provisions a database through pgtestdb and returns its DSN, ready to
// hand to database.Connect or database.Migrate.
func instance(t *testing.T, m pgtestdb.Migrator) string {
	t.Helper()

	// The returned DSN names the pgtdbuser role, not the administrative one;
	// Custom closes its own connection, so the caller owns the database.
	cfg := pgtestdb.Custom(t, config(t), m)

	// Built from the fields rather than with cfg.URL(), which interpolates the
	// password into the URL unescaped and would produce a broken DSN for one
	// containing "@" or ":".
	dsn := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		Path:     "/" + cfg.Database,
		RawQuery: cfg.Options,
	}).String()

	// Then through the application's own normaliser, rather than trusting
	// pgtestdb to round-trip the query string: if this DSN is one the
	// application would refuse, the test should say so here, not somewhere
	// deep in a handler.
	dsn, err := database.NormalizePostgresDSN(dsn)
	if err != nil {
		t.Fatalf("pgtestdb produced a DSN the application refuses: %v", err)
	}
	return dsn
}

// config reads TEST_POSTGRES_DSN and parses it, failing the test if it is
// missing or unusable.
func config(t *testing.T) pgtestdb.Config {
	t.Helper()

	raw := os.Getenv(AdminDSN)
	if raw == "" {
		t.Fatalf("%s is not set", AdminDSN)
	}
	cfg, err := parseConfig(raw)
	if err != nil {
		t.Fatalf("%s: %v", AdminDSN, err)
	}
	return cfg
}

// parseConfig turns an administrative connection string into a pgtestdb
// configuration.
//
// pgtestdb opens its own connections to create and migrate the template, so the
// timezone pin has to be applied here too. Without it the template is built in
// the server's zone, and every row of data in it — the fixtures are INSERTs with
// DEFAULT CURRENT_TIMESTAMP — is stored shifted by the server's offset. Measured
// on a server running Asia/Seoul: the fixture template's timestamps came out
// +32399s, versus -1s with the pin.
func parseConfig(raw string) (pgtestdb.Config, error) {
	dsn, err := database.NormalizePostgresDSN(raw)
	if err != nil {
		return pgtestdb.Config{}, err
	}

	u, err := url.Parse(dsn)
	// The scheme matters, not just the shape: the fields below are handed back
	// to pgtestdb, which rebuilds a postgres:// URL from them. Accepting
	// `mysql://user@host/db` here would silently reconnect to it as PostgreSQL.
	if err != nil || u.Host == "" || !isPostgresScheme(u.Scheme) {
		return pgtestdb.Config{}, fmt.Errorf(
			"must be a postgres:// URL naming a database that exists, got %q", raw)
	}

	password, _ := u.User.Password()
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		// pgtestdb connects to this database to administer the server; the
		// maintenance database always exists.
		name = "postgres"
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}

	return pgtestdb.Config{
		DriverName: database.Postgres().Driver(),
		Host:       u.Hostname(),
		Port:       port,
		User:       u.User.Username(),
		Password:   password,
		Database:   name,
		Options:    u.RawQuery,
	}, nil
}

// isPostgresScheme reports whether a URL scheme names PostgreSQL. NormalizePostgresDSN
// only prefixes the two URL forms; pgx also accepts them, so nothing else is
// offered here.
func isPostgresScheme(scheme string) bool {
	return scheme == "postgres" || scheme == "postgresql"
}

// migrator builds the template migrator: the embedded migrations, optionally
// followed by the fixtures.
func migrator(withFixtures bool) pgtestdb.Migrator {
	schema := goosemigrator.New(".",
		goosemigrator.WithFS(migrations.Embed()),
		goosemigrator.WithTableName(database.MigrateTable))
	if !withFixtures {
		return schema
	}
	return withFixturesApplied{schema: schema}
}

// withFixturesApplied is the schema migrator plus the fixtures. They are applied
// to the template rather than to each test's database, so the fixture script
// runs once for the whole suite.
type withFixturesApplied struct {
	schema pgtestdb.Migrator
}

// Hash keys the template. Both halves contribute, so the fixture template is a
// different database from the plain one and editing either set of files
// rebuilds the right template.
func (m withFixturesApplied) Hash() (string, error) {
	schema, err := m.schema.Hash()
	if err != nil {
		return "", err
	}
	fixtures, err := m.fixtures().Hash()
	if err != nil {
		return "", err
	}
	return schema + "-" + fixtures, nil
}

// Migrate applies the schema first: the fixtures are INSERTs and UPDATEs
// against tables the schema creates.
func (m withFixturesApplied) Migrate(ctx context.Context, db *sql.DB, cfg pgtestdb.Config) error {
	if err := m.schema.Migrate(ctx, db, cfg); err != nil {
		return err
	}
	return m.fixtures().Migrate(ctx, db, cfg)
}

func (m withFixturesApplied) fixtures() pgtestdb.Migrator {
	return goosemigrator.New(".",
		goosemigrator.WithFS(FixturesFS()),
		goosemigrator.WithTableName(FixturesTable))
}

// repoRoot returns the absolute path to the repository root, derived from this
// source file's location at compile time. The test suite changes the working
// directory, so the path has to be anchored to the source, not to the process.
func repoRoot() string {
	_, src, _, _ := runtime.Caller(0)
	// src = <root>/internal/testutil/pgtest/pgtest.go
	return filepath.Join(filepath.Dir(src), "..", "..", "..")
}
