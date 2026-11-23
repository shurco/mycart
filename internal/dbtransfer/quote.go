package dbtransfer

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// copyToStatement reads a table out of PostgreSQL: `COPY "t" ("c") TO STDOUT
// (FORMAT text)`.
func copyToStatement(t table) string {
	return fmt.Sprintf("COPY %s (%s) TO STDOUT (FORMAT text)",
		quoteIdent(t.Name), quoteIdents(t.Columns))
}

// copyFromStatement writes a table into PostgreSQL, the mirror of
// copyToStatement.
func copyFromStatement(t table) string {
	return fmt.Sprintf("COPY %s (%s) FROM STDIN (FORMAT text)",
		quoteIdent(t.Name), quoteIdents(t.Columns))
}

// selectStatement reads a table's columns in the order the dump writes them.
func selectStatement(t table) string {
	return "SELECT " + quoteIdents(t.Columns) + " FROM " + quoteIdent(t.Name)
}

// quoteIdent quotes an identifier the way PostgreSQL does when it has to: always.
// The schema has a column named `desc`, which is a reserved word, and a dump
// that failed on it would be a dump nobody could restore.
func quoteIdent(name string) string {
	return pgx.Identifier{name}.Sanitize()
}

// quoteIdents quotes a list of identifiers.
func quoteIdents(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = quoteIdent(name)
	}
	return strings.Join(quoted, ", ")
}
