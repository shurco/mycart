package dbtransfer

import (
	"context"
	"fmt"
	"io"

	"github.com/shurco/mycart/internal/database"
)

// source is one database being read from: PostgreSQL or SQLite. Both render
// their rows in the same COPY text format, so nothing downstream needs to know
// which one it is talking to.
type source interface {
	Driver() string
	Version() string
	MigrationVersion(ctx context.Context) (int64, error)
	// Tables lists the tables to copy, in the order they have to be written.
	Tables(ctx context.Context) ([]table, error)
	// CopyTo writes the rows of one table as COPY text.
	CopyTo(ctx context.Context, w io.Writer, t table) (int64, error)
	Count(ctx context.Context, t table) (int64, error)
	Close(ctx context.Context) error
}

// openSource connects to the configured database for reading.
func openSource(ctx context.Context, cfg database.Config) (source, error) {
	switch cfg.Driver {
	case database.DriverPostgres:
		return openPostgresSource(ctx, cfg.DSN)
	case database.DriverSQLite:
		return openSQLiteSource(ctx, cfg.DSN)
	default:
		return nil, fmt.Errorf("unknown database driver %q", cfg.Driver)
	}
}

// latestMigrationSQL is the goose version query both drivers run. It is a const
// and so quotes the table by hand: quoteIdent is not a constant expression, and
// MigrateTable is a fixed name with nothing to escape.
const latestMigrationSQL = `SELECT COALESCE(MAX(version_id), 0) FROM "` +
	database.MigrateTable + `" WHERE is_applied`
