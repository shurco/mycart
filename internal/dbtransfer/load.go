package dbtransfer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// LoadOptions are the extras a load cannot infer from the dump.
type LoadOptions struct {
	// Replace allows a dump to be loaded over a database that already holds an
	// installation. Without it a load refuses to run: it empties the target
	// first, and doing that to a live shop by accident is the one mistake this
	// operation must not make quietly.
	Replace bool
}

// Load reads a dump and writes it into the PostgreSQL database named by dsn.
//
// The schema is not created here — the caller migrates the target first, and a
// dump only carries data. The target's tables are emptied before the data is
// loaded, because a migrated database is not empty: the migrations seed the
// setting table, and a load has to leave the target holding exactly what the
// dump holds.
//
// Everything happens in one transaction, so a dump that turns out to be
// truncated, or a row the target refuses, leaves the target as it was.
func Load(ctx context.Context, dsn string, r io.Reader, opts LoadOptions) (*Manifest, error) {
	in := bufio.NewReaderSize(r, 1<<16)
	header, err := readHeader(in)
	if err != nil {
		return nil, err
	}

	conn, err := connectPostgres(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()

	target, err := checkTarget(ctx, tx, header, opts.Replace, true)
	if err != nil {
		return nil, err
	}

	// One statement, before any data is written: a half-emptied target left
	// behind by a failure is worse than a refusal.
	if _, err := tx.Exec(ctx, "TRUNCATE TABLE "+quoteIdents(target)+" CASCADE"); err != nil {
		return nil, fmt.Errorf("clear the target: %w", err)
	}

	manifest, err := restore(ctx, tx, in, header, target)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	committed = true

	return manifest, nil
}

// restore loads the body of a dump into the emptied target and checks it against
// the dump's own summary and against what the target now holds.
func restore(ctx context.Context, tx pgx.Tx, in *bufio.Reader, header Header, target []string) (*Manifest, error) {
	manifest := &Manifest{Header: header}
	trailer, loaded, err := readSections(ctx, tx, in, target, manifest)
	if err != nil {
		return nil, err
	}
	manifest.Absent = tablesMissingFrom(target, loaded)

	if err := checkDumpEnd(trailer, manifest); err != nil {
		return nil, err
	}

	if err := verifyRows(ctx, tx, manifest.Tables); err != nil {
		return nil, err
	}
	return manifest, nil
}

// readSections reads the body of a dump into tx, one section at a time, and
// reports the summary the file ends with plus the tables it actually carried.
func readSections(ctx context.Context, tx pgx.Tx, in *bufio.Reader, target []string,
	manifest *Manifest) (*Trailer, map[string]bool, error) {
	loaded := map[string]bool{}
	var trailer *Trailer

	for {
		// A read that returns nothing is the end of the file, or a file that
		// stopped arriving; there is no third case.
		line, readErr := in.ReadString('\n')
		if line == "" {
			if err := readFailure(readErr); err != nil {
				return nil, nil, err
			}
			break
		}

		next, err := readSectionLine(ctx, tx, in, line, target, loaded, manifest, trailer)
		if err != nil {
			return nil, nil, err
		}
		trailer = next
		if err := readFailure(readErr); err != nil {
			return nil, nil, err
		}
	}

	return trailer, loaded, nil
}

// readSectionLine consumes one line of the dump body: a blank line, the closing
// summary, or the COPY header that starts a section. It returns the summary in
// force afterwards, which is the one it was given unless this line carried a new
// one.
func readSectionLine(ctx context.Context, tx pgx.Tx, in *bufio.Reader, line string, target []string,
	loaded map[string]bool, manifest *Manifest, trailer *Trailer) (*Trailer, error) {
	text := strings.TrimSpace(line)
	switch {
	case text == "":
		// Blank lines between sections are harmless.
	case strings.HasPrefix(text, "--"):
		if t, ok := parseTrailer(text); ok {
			trailer = t
		}
	case hasPrefixFold(text, "COPY"):
		stat, err := loadSection(ctx, tx, in, text, target, loaded)
		if err != nil {
			return nil, err
		}
		manifest.Tables = append(manifest.Tables, stat)
		manifest.Rows += stat.Rows
	default:
		return nil, fmt.Errorf("this is not a myCart dump: unexpected line %s", quoteLine(text))
	}
	return trailer, nil
}

// readFailure turns the error a read of the next line came back with into the
// error a load reports, or nil when the read was fine or simply hit the end.
func readFailure(err error) error {
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read the dump: %w", err)
	}
	return nil
}

// tablesMissingFrom names the tables a dump did not carry, in the order the
// target lists them.
func tablesMissingFrom(target []string, loaded map[string]bool) []string {
	var absent []string
	for _, name := range target {
		if !loaded[name] {
			absent = append(absent, name)
		}
	}
	return absent
}

// checkDumpEnd refuses a dump that carries no tables or lost its summary: the
// trailer is written last, so a file without one stopped arriving somewhere
// between the machine that wrote it and this one.
func checkDumpEnd(trailer *Trailer, manifest *Manifest) error {
	if len(manifest.Tables) == 0 {
		return errors.New("the dump carries no tables")
	}
	if trailer == nil {
		return errors.New("the dump has no summary at the end: the file is incomplete")
	}
	return checkTrailer(trailer, manifest)
}

// loadSection reads one COPY section out of the dump and hands it to the
// server, which parses it with the same code that reads a psql script.
func loadSection(ctx context.Context, tx pgx.Tx, in *bufio.Reader, header string, target []string,
	loaded map[string]bool,
) (TableStat, error) {
	t, err := parseCopyHeader(header)
	if err != nil {
		return TableStat{}, err
	}
	if loaded[t.Name] {
		return TableStat{}, fmt.Errorf("the dump carries %s twice", t.Name)
	}
	loaded[t.Name] = true

	if !slices.Contains(target, t.Name) {
		return TableStat{}, fmt.Errorf("the dump carries a table this schema does not have: %s "+
			"(the dump is from a newer myCart)", t.Name)
	}

	section := &copySection{br: in}
	tag, err := tx.Conn().PgConn().CopyFrom(ctx, section, copyFromStatement(t))
	if err != nil {
		return TableStat{}, fmt.Errorf("load %s: %w", t.Name, err)
	}

	// The server's count and the file's count have to agree. A difference means
	// the file was mangled in a way PostgreSQL tolerated, and the row count is
	// the only thing that would have noticed.
	if accepted := tag.RowsAffected(); accepted != section.rows {
		return TableStat{}, fmt.Errorf("load %s: read %d rows from the dump, the database took %d",
			t.Name, section.rows, accepted)
	}

	return TableStat{Name: t.Name, Columns: t.Columns, Rows: section.rows}, nil
}

// verifyRows confirms the target holds exactly the rows that were loaded. It is
// the check a human would make with COUNT(*), and it is what catches data a
// trigger, a default or a constraint changed on the way in.
func verifyRows(ctx context.Context, tx pgx.Tx, tables []TableStat) error {
	for _, t := range tables {
		var rows int64
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM "+quoteIdent(t.Name)).Scan(&rows); err != nil {
			return fmt.Errorf("verify %s: %w", t.Name, err)
		}
		if rows != t.Rows {
			return fmt.Errorf("verify %s: the dump has %d rows, the target holds %d", t.Name, t.Rows, rows)
		}
	}
	return nil
}

// checkTrailer compares the summary a dump ends with against what was read. The
// trailer is written last, so a file that was cut short fails here rather than
// restoring a shop without its orders.
func checkTrailer(trailer *Trailer, manifest *Manifest) error {
	if trailer.Rows != manifest.Rows {
		return fmt.Errorf("the dump claims %d rows but carries %d: the file is incomplete",
			trailer.Rows, manifest.Rows)
	}
	if len(trailer.Tables) != len(manifest.Tables) {
		return fmt.Errorf("the dump claims %d tables but carries %d: the file is incomplete",
			len(trailer.Tables), len(manifest.Tables))
	}
	for i, want := range trailer.Tables {
		got := manifest.Tables[i]
		if want.Name != got.Name || want.Rows != got.Rows {
			return fmt.Errorf("the dump claims %s holds %d rows but carries %d",
				want.Name, want.Rows, got.Rows)
		}
	}
	return nil
}
