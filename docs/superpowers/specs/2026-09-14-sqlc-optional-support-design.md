# sqlc Optional Support Design

**Date:** 2026-09-14  
**Author:** Claude Sonnet 4.5  
**Status:** Design Approved  
**Complexity:** Large

## Summary

Add optional sqlc code generation support for both SQLite and PostgreSQL while preserving the existing queries layer structure. sqlc support is enabled via Go build tags (`-tags sqlc`), making it completely opt-in with zero overhead when not used. The existing raw SQL implementation remains the default, ensuring backward compatibility.

## Requirements

1. **Preserve existing structure**: Keep current queries layer intact
2. **Optional via build tag**: Use `-tags sqlc` to enable at compile time
3. **Support both dialects**: Generate code for SQLite and PostgreSQL
4. **Minimal code changes**: Touch existing files as little as possible
5. **Incremental migration**: Migrate one query group at a time
6. **Commit generated code**: Users don't need sqlc installed to build
7. **Backward compatible**: Default build works exactly as today

## Design Decisions

### Build Tag System (Compile-Time Flag)

**Decision:** Use Go build tags for conditional compilation  
**Alternative considered:** Runtime environment variable  
**Rationale:** 
- Zero runtime overhead when not using sqlc
- Generated code only compiled when needed
- Idiomatic Go pattern for conditional compilation
- Clear separation in codebase

### SQL File Organization (Separate Directories)

**Decision:** Separate `db/queries/postgres/` and `db/queries/sqlite/` directories  
**Alternatives considered:** Single directory with dialect markers, single directory split in config  
**Rationale:**
- Matches previous implementation exactly
- Clear separation of dialect-specific SQL
- Each engine's queries stay independent
- Easy to see dialect differences

### Integration Strategy (Wrapper Delegation)

**Decision:** Existing methods delegate to generated code when available  
**Alternatives considered:** Direct replacement, interface-based switching  
**Rationale:**
- Keeps existing structure intact
- Each handwritten method stays as public API
- Generated code is implementation detail
- Allows custom logic before/after generated calls
- Minimal changes to existing code

### Dialect Handling (Single Interface Wrapper)

**Decision:** Single `sqlcBackend` interface, dialect chosen at initialization  
**Alternatives considered:** Direct to sql.DB bypassing Conn, separate pointers for each dialect  
**Rationale:**
- Clean single field instead of checking two pointers
- Dialect selected once at initialization
- Matches current architecture (dialect-agnostic at query layer)
- sqlc packages already match signature
- No runtime branching in every method

### Generated Code Location

**Decision:** `internal/queries/sqlc/{postgres,sqlite}/`  
**Alternatives considered:** Separate internal package, top-level generated directory  
**Rationale:**
- Generated code lives close to where it's used
- Clear that sqlc/ is generated
- Keeps all query-related code in one subtree
- Easy imports
- Mirrors previous implementation

### Git Strategy (Commit Generated Code)

**Decision:** Commit generated Go code to repository  
**Alternatives considered:** Regenerate on build, hybrid approach  
**Rationale:**
- Users don't need sqlc installed to build with `-tags sqlc`
- CI doesn't need sqlc tooling
- Faster builds (no generation step)
- Clear diff when queries change
- Matches Go ecosystem patterns (protobuf, etc.)

### Migration Strategy (Incremental by Query Group)

**Decision:** Migrate one query group at a time in order of complexity  
**Alternatives considered:** All at once, critical queries first, optional per-method  
**Rationale:**
- Lower risk - test each group independently
- Easier to debug issues
- Build confidence with simple queries first
- Each commit is reviewable and testable

## Architecture

### File Structure

```
mycart/
├── db/
│   └── queries/
│       ├── postgres/
│       │   ├── auth.sql
│       │   ├── setting.sql
│       │   ├── product.sql
│       │   ├── cart.sql
│       │   ├── customer.sql
│       │   └── pages.sql
│       └── sqlite/
│           ├── auth.sql
│           ├── setting.sql
│           ├── product.sql
│           ├── cart.sql
│           ├── customer.sql
│           └── pages.sql
├── internal/
│   └── queries/
│       ├── sqlc/                    # Generated (committed)
│       │   ├── postgres/
│       │   │   ├── auth.sql.go
│       │   │   ├── db.go
│       │   │   ├── models.go
│       │   │   └── ...
│       │   └── sqlite/
│       │       ├── auth.sql.go
│       │       ├── db.go
│       │       ├── models.go
│       │       └── ...
│       ├── backend_sqlc.go          # +build sqlc - interface
│       ├── queries_sqlc.go          # +build sqlc - init logic
│       ├── queries_stub.go          # +build !sqlc - no-op
│       ├── auth.go                  # Existing (+1 field)
│       ├── auth_sqlc.go             # +build sqlc - delegation
│       ├── auth_test.go             # Existing (unchanged)
│       ├── setting.go               # Existing (+1 field)
│       ├── setting_sqlc.go          # +build sqlc
│       └── ...
└── sqlc.yaml                        # sqlc configuration
```

### Build Tag System

**Two compilation modes:**

1. **Default mode** (no build tag):
   ```bash
   go build ./...
   ```
   - Only compiles existing `queries/*.go` files
   - Uses raw SQL implementation
   - Zero overhead, no sqlc code included

2. **sqlc mode** (with build tag):
   ```bash
   go build -tags sqlc ./...
   ```
   - Compiles existing files PLUS `queries/*_sqlc.go` files
   - Uses sqlc-generated type-safe queries
   - Existing methods delegate to generated code

### Minimal Changes to Existing Code

**Total changes to existing files: 7 lines**

1. Add `sqlc sqlcBackend` field to each query struct (6 structs × 1 line = 6 lines)
2. Add `initSqlc(base, conn)` call in `queries.go` NewBase (1 line)

**Example:**

```go
// internal/queries/auth.go
type AuthQueries struct {
    DB   *database.Conn
    sqlc sqlcBackend  // ← ONLY new line
}

// internal/queries/queries.go
func NewBase(conn *database.Conn) *Base {
    base := &Base{
        conn:            conn,
        AuthQueries:     AuthQueries{DB: conn},
        // ... existing code
    }
    
    initSqlc(base, conn)  // ← ONLY new line
    
    return base
}
```

All other changes are NEW files with build tags.

## sqlc Configuration

**sqlc.yaml** (project root):

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations"
    queries: "db/queries/postgres"
    gen:
      go:
        package: "postgres"
        out: "internal/queries/sqlc/postgres"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: false
        emit_exact_table_names: false
        emit_empty_slices: true
        emit_pointers_for_null_types: false
        emit_result_struct_pointers: false
        emit_params_struct_pointers: false
        overrides:
          - column: "product.seo"
            go_type: "string"
          - column: "product.metadata"
            go_type: "string"
          - column: "product.attributes"
            go_type: "string"
          - column: "page.seo"
            go_type: "string"

  - engine: "sqlite"
    schema: "migrations"
    queries: "db/queries/sqlite"
    gen:
      go:
        package: "sqlite"
        out: "internal/queries/sqlc/sqlite"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: false
        emit_exact_table_names: false
        emit_empty_slices: true
        emit_pointers_for_null_types: false
        emit_result_struct_pointers: false
        emit_params_struct_pointers: false
        overrides:
          - column: "product.seo"
            go_type: "string"
          - column: "product.metadata"
            go_type: "string"
          - column: "product.attributes"
            go_type: "string"
          - column: "page.seo"
            go_type: "string"
```

## Query File Structure

### Dialect Differences

| Feature | PostgreSQL | SQLite | Notes |
|---------|-----------|--------|-------|
| Placeholders | `$1, $2, $3` | `?` | sqlc handles this |
| Array params | `ANY($1::text[])` | `IN (sqlc.slice('name'))` | Different syntax |
| JSON aggregation | `json_agg(...)` | `json_group_array(...)` | Use Dialect helpers |
| Epoch conversion | `EXTRACT(EPOCH FROM ...)::bigint` | `strftime('%s', ...)` | Use Dialect helpers |

### Example SQL Files

**db/queries/postgres/auth.sql:**
```sql
-- name: GetAuthSettings :many
SELECT key, value
FROM setting
WHERE key = ANY($1::text[]);
```

**db/queries/sqlite/auth.sql:**
```sql
-- name: GetAuthSettings :many
SELECT key, value
FROM setting
WHERE key IN (sqlc.slice('keys'));
```

### Query Naming Convention

- **SQL query name**: PascalCase matching purpose (e.g., `GetAuthSettings`)
- **Wrapper method**: Matches existing API (e.g., `GetPasswordByEmail`)
- One SQL query may serve multiple wrapper methods
- Complex business logic stays in Go wrapper

## Generated Code Integration

### sqlcBackend Interface

**internal/queries/backend_sqlc.go:**

```go
// +build sqlc

package queries

import "context"

// sqlcBackend defines the interface that both postgres.Queries and sqlite.Queries satisfy.
type sqlcBackend interface {
    // Auth queries
    GetAuthSettings(ctx context.Context, keys []string) ([]SettingRow, error)
    
    // Setting queries
    GetAllSettings(ctx context.Context) ([]SettingRow, error)
    UpsertSetting(ctx context.Context, arg UpsertSettingParams) error
    
    // Add more methods as we migrate each query group
}

// Common types shared across dialects
type SettingRow struct {
    Key   string
    Value string
}

type UpsertSettingParams struct {
    Key   string
    Value string
}
```

### Initialization Logic

**internal/queries/queries_sqlc.go:**

```go
// +build sqlc

package queries

import (
    "github.com/shurco/mycart/internal/database"
    pggen "github.com/shurco/mycart/internal/queries/sqlc/postgres"
    sqlitegen "github.com/shurco/mycart/internal/queries/sqlc/sqlite"
)

func initSqlc(base *Base, conn *database.Conn) {
    var backend sqlcBackend
    
    switch conn.Dialect().Name() {
    case "postgres":
        backend = pggen.New(conn.Raw())
    case "sqlite":
        backend = sqlitegen.New(conn.Raw())
    }
    
    // Inject into query groups as they are migrated
    base.AuthQueries.sqlc = backend
    base.SettingQueries.sqlc = backend
    // ... add more as we migrate
}
```

**internal/queries/queries_stub.go:**

```go
// +build !sqlc

package queries

import "github.com/shurco/mycart/internal/database"

func initSqlc(base *Base, conn *database.Conn) {
    // No-op when sqlc not enabled
}
```

### Wrapper Implementation

**internal/queries/auth_sqlc.go:**

```go
// +build sqlc

package queries

import (
    "context"
    "github.com/shurco/mycart/pkg/errors"
)

func (q *AuthQueries) GetPasswordByEmail(ctx context.Context, email string) (string, error) {
    if q.sqlc == nil {
        return q.getPasswordByEmailRaw(ctx, email)
    }
    
    rows, err := q.sqlc.GetAuthSettings(ctx, []string{"email", "password"})
    if err != nil {
        return "", err
    }
    
    var emailValue, passwordValue string
    var foundEmail, foundPassword bool
    
    for _, row := range rows {
        switch row.Key {
        case "email":
            foundEmail = true
            emailValue = row.Value
        case "password":
            foundPassword = true
            passwordValue = row.Value
        }
    }
    
    if !foundEmail || !foundPassword {
        return "", errors.ErrUserNotFound
    }
    
    if emailValue != email {
        return "", errors.ErrUserEmailNotFound
    }
    
    if passwordValue == "" {
        return "", errors.ErrUserPasswordNotFound
    }
    
    return passwordValue, nil
}

// Fallback implementation (copy of original)
func (q *AuthQueries) getPasswordByEmailRaw(ctx context.Context, email string) (string, error) {
    query := `SELECT key, value FROM setting WHERE key IN ('email', 'password')`
    rows, err := q.DB.QueryContext(ctx, query)
    if err != nil {
        return "", err
    }
    defer rows.Close()
    
    for rows.Next() {
        var key, value string
        if err := rows.Scan(&key, &value); err != nil {
            return "", err
        }
        
        switch key {
        case "email":
            if value != email {
                return "", errors.ErrUserEmailNotFound
            }
        case "password":
            if value == "" {
                return "", errors.ErrUserPasswordNotFound
            }
            return value, nil
        }
    }
    
    if err := rows.Err(); err != nil {
        return "", err
    }
    
    return "", errors.ErrUserNotFound
}
```

## Migration Path

### Migration Order (Simple → Complex)

1. **auth** - Simplest (1 method, basic queries)
2. **session** - Simple (3 methods, straightforward)
3. **install** - Simple (2 methods, one-time operations)
4. **setting** - Medium (5 methods, JSON handling)
5. **pages** - Medium (6 methods, basic CRUD)
6. **product** - Complex (15+ methods, variants, JSON, images)
7. **cart** - Most Complex (20+ methods, transactions, inventory)
8. **customer** - Complex (10+ methods, addresses, orders)

### Step-by-Step Process (9 Steps per Query Group)

Using 'auth' as example:

**Step 1: Analyze Existing Methods**
- Identify all methods in `internal/queries/auth.go`
- Document method signatures and behavior

**Step 2: Write SQL Files**
- Create `db/queries/postgres/auth.sql`
- Create `db/queries/sqlite/auth.sql`
- Use sqlc annotations (`-- name: GetAuthSettings :many`)

**Step 3: Generate sqlc Code**
```bash
sqlc generate
```

**Step 4: Add sqlc Field to Struct**
```go
// internal/queries/auth.go
type AuthQueries struct {
    DB   *database.Conn
    sqlc sqlcBackend  // ← Add this line
}
```

**Step 5: Create Wrapper File**
- Create `internal/queries/auth_sqlc.go`
- Add `// +build sqlc` header
- Implement delegation methods

**Step 6: Update backend_sqlc.go**
- Add method signatures to `sqlcBackend` interface
- Add common types (SettingRow, etc.)

**Step 7: Update queries_sqlc.go**
- Add `base.AuthQueries.sqlc = backend` injection

**Step 8: Test Both Modes**
```bash
# Test all 4 combinations
go test ./internal/queries -run TestAuth -count=1 -race
go test -tags sqlc ./internal/queries -run TestAuth -count=1 -race
TEST_DB_DRIVER=postgres ... go test ./internal/queries -run TestAuth -count=1 -race
TEST_DB_DRIVER=postgres ... go test -tags sqlc ./internal/queries -run TestAuth -count=1 -race
```

**Step 9: Commit**
```bash
git add db/queries/ internal/queries/
git commit -m "feat(sqlc): add sqlc support for auth queries"
```

## Testing Strategy

### No Changes to Existing Tests

Existing `*_test.go` files stay **completely unchanged**. They test the public API, which both implementations satisfy.

### Test Matrix (4 Combinations Required)

| Build Mode | Database | Command |
|------------|----------|---------|
| Raw SQL | SQLite | `go test ./internal/queries -count=1 -race` |
| Raw SQL | PostgreSQL | `TEST_DB_DRIVER=postgres ... go test ./internal/queries -count=1 -race` |
| sqlc | SQLite | `go test -tags sqlc ./internal/queries -count=1 -race` |
| sqlc | PostgreSQL | `TEST_DB_DRIVER=postgres ... go test -tags sqlc ./internal/queries -count=1 -race` |

All 4 must pass before migration is complete.

### Makefile Test Targets

```makefile
.PHONY: test-queries-all
test-queries-all: test-queries-raw-sqlite test-queries-raw-postgres test-queries-sqlc-sqlite test-queries-sqlc-postgres
	@echo "✓ All query tests passed (4 modes)"

.PHONY: test-queries-raw-sqlite
test-queries-raw-sqlite:
	go test ./internal/queries -count=1 -race

.PHONY: test-queries-sqlc-sqlite
test-queries-sqlc-sqlite:
	go test -tags sqlc ./internal/queries -count=1 -race

.PHONY: test-auth-all
test-auth-all:
	go test ./internal/queries -run TestAuth -count=1 -race
	go test -tags sqlc ./internal/queries -run TestAuth -count=1 -race
	TEST_DB_DRIVER=postgres ... go test ./internal/queries -run TestAuth -count=1 -race
	TEST_DB_DRIVER=postgres ... go test -tags sqlc ./internal/queries -run TestAuth -count=1 -race
	@echo "✓ Auth tests passed (all 4 modes)"
```

## Build & Deployment

### Build Commands

```bash
# Default build (no sqlc)
go build ./cmd

# Build with sqlc
go build -tags sqlc ./cmd
```

### Makefile Targets

```makefile
.PHONY: build
build:
	go build -o mycart ./cmd

.PHONY: build-sqlc
build-sqlc:
	go build -tags sqlc -o mycart-sqlc ./cmd

.PHONY: sqlc-generate
sqlc-generate:
	sqlc generate
	@echo "✓ sqlc code generated"

.PHONY: sqlc-verify
sqlc-verify:
	sqlc diff
	@echo "✓ Generated code is up to date"
```

### CI/CD Integration

GitHub Actions should test all 4 modes:
- SQLite + raw SQL
- SQLite + sqlc
- PostgreSQL + raw SQL
- PostgreSQL + sqlc

### Developer Workflow

```bash
# First time setup
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# After writing SQL files
make sqlc-generate

# Build and test
make build-sqlc
make test-sqlc

# Commit (including generated code)
git add db/queries/ internal/queries/sqlc/
git commit -m "feat(sqlc): update queries"
```

### Distribution

Provide both binaries in releases:
```bash
mycart          # Raw SQL (default, smaller)
mycart-sqlc     # Type-safe sqlc queries
```

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Generated code diverges from raw SQL | Medium | 4-mode test matrix catches discrepancies |
| sqlc doesn't support edge cases | Low | Keep raw SQL as fallback, fallback path in wrappers |
| Build complexity increases | Low | Clear documentation, Makefile targets |
| Performance regression | Low | Both use same database.Conn, minimal overhead |
| Maintenance burden (two implementations) | Medium | Incremental migration allows stopping if burden too high |

## Success Criteria

- [ ] All 8 query groups migrated to sqlc
- [ ] All tests pass in 4 modes (raw/sqlc × SQLite/PostgreSQL)
- [ ] Generated code committed to repository
- [ ] Documentation updated (README, build instructions)
- [ ] CI/CD tests all 4 modes
- [ ] Zero changes to existing behavior when built without `-tags sqlc`
- [ ] Each query group migration is independently testable

## Future Enhancements

- Consider making sqlc the default after migration complete
- Explore removing raw SQL implementation if sqlc proves stable
- Add performance benchmarks comparing both implementations
- Document performance characteristics for users choosing between modes

## References

- Previous sqlc implementation: `/home/wj/work/dure-mycart`
- Current queries layer: `/home/wj/work/mycart/internal/queries/`
- Database abstraction: `/home/wj/work/mycart/internal/database/`
- sqlc documentation: https://docs.sqlc.dev/
