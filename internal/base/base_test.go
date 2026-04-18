package base

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shurco/mycart/migrations"
	_ "modernc.org/sqlite"
)

func TestBuildDSN_ContainsExpectedPragmas(t *testing.T) {
	t.Parallel()
	dsn := buildDSN("/tmp/x.db")
	for _, frag := range []string{
		"/tmp/x.db",
		"journal_mode(WAL)",
		"busy_timeout(10000)",
		"foreign_keys(ON)",
	} {
		if !strings.Contains(dsn, frag) {
			t.Errorf("DSN missing %q: %s", frag, dsn)
		}
	}
}

func TestNew_CreatesFileAndRunsMigrations(t *testing.T) {
	// Not parallel: goose mutates package-level state (SetBaseFS/SetTableName).
	tmp := t.TempDir()
	prev, _ := os.Getwd()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(prev) })

	dbPath := filepath.Join(tmp, "lc_base", "data.db")
	db, err := New(dbPath, migrations.Embed())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("database file not created: %v", err)
	}

	// Schema should include the 'setting' table created by the first migration.
	var name string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='setting'`).Scan(&name)
	if err != nil {
		t.Fatalf("migrations did not create 'setting' table: %v", err)
	}
}

func TestNew_AlreadyExistingDatabaseSkipsMigrations(t *testing.T) {
	tmp := t.TempDir()
	prev, _ := os.Getwd()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(prev) })

	dbPath := filepath.Join(tmp, "lc_base", "data.db")

	first, err := New(dbPath, migrations.Embed())
	if err != nil {
		t.Fatalf("first New: %v", err)
	}
	_ = first.Close()

	// The file now exists, so the second call skips migration entirely.
	second, err := New(dbPath, migrations.Embed())
	if err != nil {
		t.Fatalf("second New: %v", err)
	}
	defer func() { _ = second.Close() }()

	if _, err := second.Exec("SELECT 1"); err != nil {
		t.Errorf("reopened db not usable: %v", err)
	}
}
