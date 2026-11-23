package dbtransfer

import (
	"context"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5"

	"github.com/shurco/mycart/internal/database"
)

// pgSource reads a PostgreSQL database through its own connection: COPY is a
// protocol feature, not SQL, so it needs the raw driver rather than the
// database/sql wrapper the application queries through.
type pgSource struct {
	conn    *pgx.Conn
	version string
}

func openPostgresSource(ctx context.Context, dsn string) (*pgSource, error) {
	conn, err := connectPostgres(ctx, dsn)
	if err != nil {
		return nil, err
	}

	var version string
	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		_ = conn.Close(ctx)
		return nil, fmt.Errorf("read the postgres version: %w", err)
	}

	return &pgSource{conn: conn, version: version}, nil
}

func (s *pgSource) Driver() string  { return database.DriverPostgres }
func (s *pgSource) Version() string { return s.version }

func (s *pgSource) MigrationVersion(ctx context.Context) (int64, error) {
	return latestMigration(ctx, s.conn)
}

func (s *pgSource) Tables(ctx context.Context) ([]table, error) {
	tables, err := discoverPostgres(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	return orderTables(tables)
}

func (s *pgSource) CopyTo(ctx context.Context, w io.Writer, t table) (int64, error) {
	tag, err := s.conn.PgConn().CopyTo(ctx, w, copyToStatement(t))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *pgSource) Count(ctx context.Context, t table) (int64, error) {
	var rows int64
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM "+quoteIdent(t.Name)).Scan(&rows)
	if err != nil {
		return 0, fmt.Errorf("count %s: %w", t.Name, err)
	}
	return rows, nil
}

func (s *pgSource) Close(ctx context.Context) error { return s.conn.Close(ctx) }

// latestMigration returns the goose version a PostgreSQL database is on.
func latestMigration(ctx context.Context, conn *pgx.Conn) (int64, error) {
	var version int64
	if err := conn.QueryRow(ctx, latestMigrationSQL).Scan(&version); err != nil {
		return 0, fmt.Errorf("read the migration version: %w", err)
	}
	return version, nil
}
