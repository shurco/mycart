package dbtransfer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/shurco/mycart/internal/database"
)

// The dump format is versioned so a future change can be detected instead of
// misread.
const (
	// FormatVersion is the version of the dump format this build writes and the
	// highest it can read.
	FormatVersion = 1
	// Magic tags the header comment; MagicTrailer tags the summary at the end.
	Magic        = "mycart-dump"
	MagicTrailer = "mycart-dump-trailer"
)

// Header is the structured comment a dump starts with.
type Header struct {
	Magic   string    `json:"magic"`
	Format  int       `json:"format"`
	Created time.Time `json:"created"`
	// Driver and Server describe where the data came from, so a dump can be
	// told apart from another installation's.
	Driver string `json:"driver"`
	Server string `json:"server"`
	// App is the myCart version that wrote the dump.
	App string `json:"app,omitempty"`
	// Migrations is the schema version of the source, as goose recorded it.
	Migrations int64 `json:"migrations"`
}

// TableStat is one table's contribution to a dump.
type TableStat struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    int64    `json:"rows"`
}

// Trailer is the structured comment a dump ends with. It carries what can only
// be known after the data has been written, and it is what makes a truncated
// file detectable.
type Trailer struct {
	Magic  string      `json:"magic"`
	Tables []TableStat `json:"tables"`
	Rows   int64       `json:"rows"`
}

// Manifest is what a dump or a load reports: the header of the file, plus the
// tables and rows it actually carried.
type Manifest struct {
	Header
	Tables []TableStat `json:"tables"`
	Rows   int64       `json:"rows"`
	// Absent lists tables the target has that the dump does not mention. They
	// are left empty, which is what a table introduced by a newer migration
	// should be.
	Absent []string `json:"-"`
}

// DumpOptions are the extras the caller knows and this package does not.
type DumpOptions struct {
	// App is the running myCart version, recorded in the header.
	App string
}

// Dump writes the contents of the configured database to w.
//
// The result is a SQL script: psql can replay it against a migrated database
// whose tables have been cleared (the file says how), and Load reads it back
// without psql.
func Dump(ctx context.Context, cfg database.Config, w io.Writer, opts DumpOptions) (*Manifest, error) {
	src, err := openSource(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close(ctx) }()

	tables, header, err := describeSource(ctx, src, opts)
	if err != nil {
		return nil, err
	}
	head, err := commentJSON(header)
	if err != nil {
		return nil, err
	}

	out := bufio.NewWriter(w)
	if err := writeDumpHead(out, head, tables); err != nil {
		return nil, err
	}

	manifest := &Manifest{Header: header}
	for _, t := range tables {
		rows, err := writeTable(ctx, out, src, t)
		if err != nil {
			return nil, err
		}
		manifest.Tables = append(manifest.Tables, TableStat{Name: t.Name, Columns: t.Columns, Rows: rows})
		manifest.Rows += rows
	}

	tail, err := commentJSON(Trailer{Magic: MagicTrailer, Tables: manifest.Tables, Rows: manifest.Rows})
	if err != nil {
		return nil, err
	}
	if _, err := out.WriteString(tail); err != nil {
		return nil, err
	}
	if err := out.Flush(); err != nil {
		return nil, fmt.Errorf("write dump: %w", err)
	}

	return manifest, nil
}

// describeSource reads what the header of a dump or a dry run needs to know
// about a source: its tables, and who and what it is.
func describeSource(ctx context.Context, src source, opts DumpOptions) ([]table, Header, error) {
	tables, err := src.Tables(ctx)
	if err != nil {
		return nil, Header{}, err
	}
	version, err := src.MigrationVersion(ctx)
	if err != nil {
		return nil, Header{}, err
	}

	return tables, Header{
		Magic:      Magic,
		Format:     FormatVersion,
		Created:    time.Now().UTC(),
		Driver:     src.Driver(),
		Server:     src.Version(),
		App:        opts.App,
		Migrations: version,
	}, nil
}

// writeDumpHead writes the title, the header comment and the replay hint a dump
// starts with.
func writeDumpHead(out *bufio.Writer, head string, tables []table) error {
	if _, err := out.WriteString("-- myCart database dump\n"); err != nil {
		return err
	}
	if _, err := out.WriteString(head); err != nil {
		return err
	}

	// The file carries data only, and a migrated database is not empty — the
	// migrations seed it — so replaying it by hand needs the target cleared
	// first. The line is written commented out: a dump that empties a database
	// the moment it is piped into psql is not a file to hand around, and psql
	// still fails loudly on the duplicate keys if it is left out.
	names := tableNames(tables)
	if len(names) == 0 {
		return nil
	}
	for _, line := range []string{
		"-- Replayed with psql, this file needs the target's tables emptied first, in the",
		"-- same transaction: `mycart db restore` does that itself, and verifies the result.",
		"--   psql -1 \"$DSN\" -c 'TRUNCATE TABLE " + quoteIdents(names) + " CASCADE;' -f <this file>",
	} {
		if _, err := out.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("write dump: %w", err)
		}
	}
	return nil
}

// tableNames lists the tables of a dump in the order they are written.
func tableNames(tables []table) []string {
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.Name
	}
	return names
}

// writeTable writes one table's COPY section and returns the number of rows in
// it.
func writeTable(ctx context.Context, out *bufio.Writer, src source, t table) (int64, error) {
	// Written the way psql writes a data section, so the file can also be
	// replayed by psql against a migrated database.
	if _, err := fmt.Fprintf(out, "COPY %s (%s) FROM stdin;\n", quoteIdent(t.Name), quoteIdents(t.Columns)); err != nil {
		return 0, fmt.Errorf("write dump of %s: %w", t.Name, err)
	}

	rows, err := src.CopyTo(ctx, out, t)
	if err != nil {
		return 0, fmt.Errorf("dump %s: %w", t.Name, err)
	}

	// PostgreSQL terminates every row with a newline, and the SQLite source
	// writes one per row too, so the terminator starts on a line of its own.
	if _, err := out.WriteString(copyTerminal + "\n"); err != nil {
		return 0, fmt.Errorf("write dump of %s: %w", t.Name, err)
	}
	return rows, nil
}

// Count reports how many rows each table holds, without writing a dump. It is
// what a dry run shows.
func Count(ctx context.Context, cfg database.Config, opts DumpOptions) (*Manifest, error) {
	src, err := openSource(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close(ctx) }()

	tables, header, err := describeSource(ctx, src, opts)
	if err != nil {
		return nil, err
	}

	manifest := &Manifest{Header: header}
	for _, t := range tables {
		rows, err := src.Count(ctx, t)
		if err != nil {
			return nil, err
		}
		manifest.Tables = append(manifest.Tables, TableStat{Name: t.Name, Columns: t.Columns, Rows: rows})
		manifest.Rows += rows
	}
	return manifest, nil
}
