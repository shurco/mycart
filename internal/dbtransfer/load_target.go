package dbtransfer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// checkTarget reports the tables a load would replace, and refuses the cases it
// must not do silently. It is the whole of the safety, in one place, so a dry
// run applies exactly the checks a real load would.
//
// requireSchema is false for a dry run of a copy: its caller migrates the
// target before writing anything, so a missing schema is the ordinary case of
// copying into a database that has never held a cart, not something to refuse.
func checkTarget(ctx context.Context, tx pgx.Tx, header Header, replace, requireSchema bool) ([]string, error) {
	// The tables come first: a target that was never migrated has no version
	// table to compare against, and "migrate it first" is the useful thing to
	// say about it.
	target, err := listTargetTables(ctx, tx)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(target, "setting") {
		if !requireSchema {
			return target, nil
		}
		return nil, errors.New("the target has no myCart schema: migrate it first")
	}

	if err := checkSchemaVersion(ctx, tx.Conn(), header); err != nil {
		return nil, err
	}

	installed, err := isInstalled(ctx, tx.Conn())
	if err != nil {
		return nil, err
	}
	if installed && !replace {
		return nil, errors.New("the target database holds an installation and would be erased — " +
			"pass --force to overwrite it")
	}
	return target, nil
}

// checkSchemaVersion refuses a dump this build does not know how to read.
func checkSchemaVersion(ctx context.Context, conn *pgx.Conn, header Header) error {
	if header.Format > FormatVersion {
		return fmt.Errorf("the dump is in format %d and this build reads up to %d: upgrade myCart first",
			header.Format, FormatVersion)
	}

	target, err := latestMigration(ctx, conn)
	if err != nil {
		return err
	}
	if header.Migrations > target {
		return fmt.Errorf("the dump is from schema version %d and this build migrates to %d: "+
			"upgrade myCart before restoring this dump", header.Migrations, target)
	}
	return nil
}

// listTargetTables returns the tables a target database has, except the
// bookkeeping ones the transfer never touches.
func listTargetTables(ctx context.Context, q pgQuerier) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("list the target's tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("list the target's tables: %w", err)
		}
		if isBookkeeping(name) {
			continue
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list the target's tables: %w", err)
	}
	return tables, nil
}

// isInstalled reports whether the target holds an installation. A migrated but
// never installed database has the setting row set to false, or no row at all.
func isInstalled(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var value string
	err := conn.QueryRow(ctx, "SELECT value FROM setting WHERE key = 'installed'").Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read the installation state: %w", err)
	}
	return strings.EqualFold(strings.TrimSpace(value), "true"), nil
}
