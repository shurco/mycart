package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	app "github.com/shurco/mycart/internal"
	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/dbtransfer"
)

// cmdDB creates the parent of the database backup, restore and copy commands.
//
// They exist next to the configured database rather than inside it: `backup`
// and `restore` act on the database this installation uses (so `--db` and
// `--db-dsn` apply to them like everywhere else), while `copy` names its two
// ends explicitly, because the whole point of a copy is that they differ.
func cmdDB() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Back up, restore and copy the database",
	}

	cmd.AddCommand(cmdDBBackup())
	cmd.AddCommand(cmdDBRestore())
	cmd.AddCommand(cmdDBCopy())

	return cmd
}

// cmdDBBackup creates and returns the backup command.
func cmdDBBackup() *cobra.Command {
	var out string

	cmd := &cobra.Command{
		Use:   "backup [flags]",
		Short: "Write a dump of the PostgreSQL database to a file",
		Long: "Write the contents of the configured PostgreSQL database to a file.\n\n" +
			"The result is a SQL script: `psql -f FILE` replays it into a migrated, empty\n" +
			"database, and `mycart db restore --from FILE` reads it back without psql.\n" +
			"A path ending in .gz is compressed.",
		Run: func(_ *cobra.Command, _ []string) {
			path := out
			if path == "" {
				path = fmt.Sprintf("mycart-%s.sql.gz", time.Now().Format("20060102-150405"))
			}

			manifest, err := app.BackupDatabase(context.Background(), resolveDB(), path)
			if err != nil {
				handleCommandError(err)
				return
			}
			printDumpReport("Backed up", manifest, "to "+path)
		},
	}

	cmd.Flags().StringVar(&out, "out", "", "file to write (default mycart-<timestamp>.sql.gz)")

	return cmd
}

// cmdDBRestore creates and returns the restore command.
func cmdDBRestore() *cobra.Command {
	var from string
	var force bool

	cmd := &cobra.Command{
		Use:   "restore [flags]",
		Short: "Replace the PostgreSQL database with a dump",
		Long: "Migrate the configured PostgreSQL database and replace its contents with a\n" +
			"dump written by `mycart db backup`.\n\n" +
			"The whole load runs in one transaction: a dump that is truncated or does not\n" +
			"fit the schema leaves the database as it was. A database that already holds\n" +
			"an installation is refused unless --force is given, because restoring into it\n" +
			"erases the shop that is in there.",
		Run: func(_ *cobra.Command, _ []string) {
			manifest, err := app.RestoreDatabase(context.Background(), resolveDB(), from, force)
			if err != nil {
				handleCommandError(err)
				return
			}
			printDumpReport("Restored", manifest, "from "+from)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "file to read (a .gz file is decompressed)")
	cmd.Flags().BoolVar(&force, "force", false, "restore over an installation, erasing it")
	_ = cmd.MarkFlagRequired("from")

	return cmd
}

// cmdDBCopy creates and returns the copy command.
func cmdDBCopy() *cobra.Command {
	var from, to string
	var dryRun, force bool

	cmd := &cobra.Command{
		Use:   "copy [flags]",
		Short: "Copy a cart into a PostgreSQL database",
		Long: "Copy every row of --from into --to, which is the configured database when\n" +
			"--to is not given.\n\n" +
			"--from is a SQLite file (spelled sqlite:PATH to be explicit) or a PostgreSQL\n" +
			"connection string, --to is a PostgreSQL connection string. This is how an existing\n" +
			"SQLite cart is moved onto PostgreSQL.\n\n" +
			"The target is migrated first and emptied table by table, and the rows are\n" +
			"verified with COUNT(*) as they land. Like `restore`, it refuses a target that\n" +
			"already holds an installation unless --force is given.",
		Run: func(_ *cobra.Command, _ []string) {
			runDBCopy(from, to, dryRun, force)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "SQLite file (or sqlite:PATH) or PostgreSQL connection string to read")
	cmd.Flags().StringVar(&to, "to", "", "PostgreSQL connection string to write (default: the configured database)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would be copied, without writing")
	cmd.Flags().BoolVar(&force, "force", false, "copy over an installation, erasing it")
	_ = cmd.MarkFlagRequired("from")

	return cmd
}

// copyEndpoints resolves the two ends of a copy: --from, and --to or the
// configured database when --to is not given.
func copyEndpoints(from, to string) (src, dst database.Config, err error) {
	if src, err = dbConfigFrom(from); err != nil {
		return src, dst, err
	}

	dst = resolveDB()
	if strings.TrimSpace(to) != "" {
		if dst, err = dbConfigFrom(to); err != nil {
			return src, dst, err
		}
	}

	return src, dst, nil
}

// runDBCopy runs the copy command.
func runDBCopy(from, to string, dryRun, force bool) {
	src, dst, err := copyEndpoints(from, to)
	if err != nil {
		handleCommandError(err)
		return
	}

	// The target is checked before it is touched: migrating a SQLite
	// destination would create the file a copy then refuses.
	if err := dbtransfer.CheckCopyTarget(dst); err != nil {
		handleCommandError(fmt.Errorf("copy into %s: %w", dst.Redacted(), err))
		return
	}

	// A copy into a database that was never migrated has nothing to
	// write into, so the target is brought up to date first — the same
	// thing `restore` does.
	if !dryRun {
		if err := app.Migrate(dst); err != nil {
			handleCommandError(fmt.Errorf("migrate the target: %w", err))
			return
		}
	}

	manifest, err := app.CopyDatabase(context.Background(), src, dst, dbtransfer.CopyOptions{
		DryRun:  dryRun,
		Replace: force,
	})
	if err != nil {
		handleCommandError(err)
		return
	}

	verb := "Copied"
	if dryRun {
		verb = "Would copy"
	}
	printDumpReport(verb, manifest, "into "+dst.Redacted())
}

// dbConfigFrom reads a connection argument: a PostgreSQL DSN, or anything else
// as the path of a SQLite database. The `sqlite:` prefix is accepted so a path
// can be spelled out, and so a script does not depend on which of the two a
// bare word happens to look like.
func dbConfigFrom(spec string) (database.Config, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return database.Config{}, fmt.Errorf("no database given")
	}

	switch {
	case strings.HasPrefix(spec, "postgres://"), strings.HasPrefix(spec, "postgresql://"):
		return database.Config{Driver: database.DriverPostgres, DSN: spec, Source: database.SourceFlag}, nil
	case strings.HasPrefix(spec, "sqlite:"):
		path := strings.TrimPrefix(spec, "sqlite:")
		if strings.TrimSpace(path) == "" {
			return database.Config{}, fmt.Errorf("%q names no SQLite file", spec)
		}
		return database.Config{Driver: database.DriverSQLite, DSN: path, Source: database.SourceFlag}, nil
	default:
		return database.Config{Driver: database.DriverSQLite, DSN: spec, Source: database.SourceFlag}, nil
	}
}

// printDumpReport shows what a transfer moved, table by table. The counts are
// the same ones the restore verified, so they double as the record of what
// happened.
func printDumpReport(verb string, manifest *dbtransfer.Manifest, where string) {
	fmt.Printf("%s %d rows in %d tables %s\n", verb, manifest.Rows, len(manifest.Tables), where)
	for _, t := range manifest.Tables {
		fmt.Printf("  %-28s %8d\n", t.Name, t.Rows)
	}
	if len(manifest.Absent) > 0 {
		fmt.Printf("  left empty, not in the dump: %s\n", strings.Join(manifest.Absent, ", "))
	}
}
