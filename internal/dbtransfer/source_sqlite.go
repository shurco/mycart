package dbtransfer

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/pkg/fsutil"
)

// sqliteSource reads a SQLite database with the application's own connection
// handling, so the file is opened with the same pragmas as always.
type sqliteSource struct {
	conn    *database.Conn
	version string
}

func openSQLiteSource(ctx context.Context, path string) (*sqliteSource, error) {
	// database.Connect creates a missing database file. For a source that is
	// the opposite of what is wanted: an empty file would dump as an empty
	// installation instead of failing.
	if !fsutil.IsFile(path) {
		return nil, fmt.Errorf("no SQLite database at %s", path)
	}

	conn, err := database.Connect(database.Config{Driver: database.DriverSQLite, DSN: path})
	if err != nil {
		return nil, err
	}

	var version string
	if err := conn.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("read the sqlite version: %w", err)
	}

	return &sqliteSource{conn: conn, version: "SQLite " + version}, nil
}

func (s *sqliteSource) Driver() string  { return database.DriverSQLite }
func (s *sqliteSource) Version() string { return s.version }

func (s *sqliteSource) MigrationVersion(ctx context.Context) (int64, error) {
	return latestMigrationSQLite(ctx, s.conn)
}

func (s *sqliteSource) Tables(ctx context.Context) ([]table, error) {
	tables, err := discoverSQLite(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	return orderTables(tables)
}

// CopyTo reads a table row by row and renders each row as COPY text.
func (s *sqliteSource) CopyTo(ctx context.Context, w io.Writer, t table) (int64, error) {
	rows, err := s.conn.QueryContext(ctx, selectStatement(t))
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", t.Name, err)
	}
	defer func() { _ = rows.Close() }()

	out := bufio.NewWriter(w)
	var count int64
	for rows.Next() {
		values := make([]any, len(t.Columns))
		pointers := make([]any, len(t.Columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return 0, fmt.Errorf("read %s: %w", t.Name, err)
		}

		for i, value := range values {
			k := kindText
			if i < len(t.Kinds) {
				k = t.Kinds[i]
			}
			if err := writeField(out, value, k, i == len(values)-1); err != nil {
				return 0, fmt.Errorf("render %s column %s: %w", t.Name, t.Columns[i], err)
			}
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read %s: %w", t.Name, err)
	}
	if err := out.Flush(); err != nil {
		return 0, fmt.Errorf("dump %s: %w", t.Name, err)
	}

	return count, nil
}

func (s *sqliteSource) Count(ctx context.Context, t table) (int64, error) {
	var rows int64
	err := s.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+quoteIdent(t.Name)).Scan(&rows)
	if err != nil {
		return 0, fmt.Errorf("count %s: %w", t.Name, err)
	}
	return rows, nil
}

func (s *sqliteSource) Close(ctx context.Context) error { return s.conn.Close() }

// latestMigrationSQLite returns the goose version a SQLite database is on.
func latestMigrationSQLite(ctx context.Context, conn *database.Conn) (int64, error) {
	var version int64
	if err := conn.QueryRowContext(ctx, latestMigrationSQL).Scan(&version); err != nil {
		return 0, fmt.Errorf("read the migration version: %w", err)
	}
	return version, nil
}
