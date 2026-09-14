# sqlc Optional Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional sqlc code generation for both SQLite and PostgreSQL while keeping the existing queries layer intact.

**Architecture:** Use Go build tags (`-tags sqlc`) to conditionally compile sqlc-generated type-safe queries. When the tag is active, existing query methods delegate to generated code. When inactive, the current raw SQL implementation is used with zero overhead.

**Tech Stack:** Go 1.26, sqlc v1.27+, PostgreSQL 17, SQLite (modernc.org/sqlite)

**Spec:** docs/superpowers/specs/2026-09-14-sqlc-optional-support-design.md

## Global Constraints

- Go build tags for conditional compilation (`// +build sqlc` and `// +build !sqlc`)
- Existing query method signatures must not change
- All tests must pass in 4 modes: raw/sqlc × SQLite/PostgreSQL
- Generated code committed to repository (users don't need sqlc installed)
- SQL queries written in dialect-specific syntax (separate postgres/ and sqlite/ directories)
- Each query group migration is independently testable and committable
- Total changes to existing files: 7 lines (6 struct fields + 1 function call)

---

## Task 1: Project Setup and Infrastructure

**Files:**
- Create: `sqlc.yaml` (project root)
- Create: `db/queries/postgres/` (directory)
- Create: `db/queries/sqlite/` (directory)
- Create: `internal/queries/backend_sqlc.go`
- Create: `internal/queries/queries_sqlc.go`
- Create: `internal/queries/queries_stub.go`
- Modify: `internal/queries/queries.go:34` (add initSqlc call in NewBase)
- Modify: `Makefile` (add sqlc targets)

**Interfaces:**
- Consumes: Existing `internal/database.Conn` interface
- Produces: `sqlcBackend` interface for all query groups to use, `initSqlc(base, conn)` function

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p db/queries/postgres
mkdir -p db/queries/sqlite
```

Expected: Directories created successfully

- [ ] **Step 2: Create sqlc.yaml configuration**

Create file at project root with content:

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

- [ ] **Step 3: Create backend_sqlc.go interface**

Create `internal/queries/backend_sqlc.go`:

```go
// +build sqlc

package queries

import "context"

// sqlcBackend defines the interface that both postgres.Queries and sqlite.Queries satisfy.
// This allows dialect-agnostic usage of generated sqlc code.
type sqlcBackend interface {
	// Methods will be added as we migrate each query group
}
```

- [ ] **Step 4: Create queries_sqlc.go initialization**

Create `internal/queries/queries_sqlc.go`:

```go
// +build sqlc

package queries

import (
	"github.com/shurco/mycart/internal/database"
)

// initSqlc initializes sqlc backend and injects it into all query groups.
// Only compiled when -tags sqlc is used.
func initSqlc(base *Base, conn *database.Conn) {
	// Will be implemented after first query group migration
	// For now, this is a placeholder that compiles but does nothing
	_ = base
	_ = conn
}
```

- [ ] **Step 5: Create queries_stub.go no-op**

Create `internal/queries/queries_stub.go`:

```go
// +build !sqlc

package queries

import "github.com/shurco/mycart/internal/database"

// initSqlc is a no-op when sqlc build tag is not active.
func initSqlc(base *Base, conn *database.Conn) {
	// Nothing to do - use raw SQL implementations
}
```

- [ ] **Step 6: Modify queries.go to call initSqlc**

In `internal/queries/queries.go`, find the `NewBase` function and add `initSqlc(base, conn)` call before the return statement:

```go
func NewBase(conn *database.Conn) *Base {
	base := &Base{
		conn:            conn,
		AuthQueries:     AuthQueries{DB: conn},
		InstallQueries:  InstallQueries{DB: conn},
		SettingQueries:  SettingQueries{DB: conn},
		PageQueries:     PageQueries{DB: conn},
		ProductQueries:  ProductQueries{DB: conn},
		CartQueries:     CartQueries{DB: conn},
		CustomerQueries: CustomerQueries{DB: conn},
	}

	initSqlc(base, conn)

	return base
}
```

- [ ] **Step 7: Add Makefile targets**

Append to existing `Makefile`:

```makefile
# sqlc code generation
.PHONY: sqlc-generate
sqlc-generate:
	sqlc generate
	@echo "✓ sqlc code generated in internal/queries/sqlc/"

.PHONY: sqlc-verify
sqlc-verify:
	sqlc diff
	@echo "✓ Generated code is up to date"

# Build targets
.PHONY: build-sqlc
build-sqlc:
	go build -tags sqlc -o mycart-sqlc ./cmd

# Test targets
.PHONY: test-sqlc
test-sqlc:
	go test -tags sqlc ./... -count=1 -race

.PHONY: test-queries-all
test-queries-all: test-queries-raw-sqlite test-queries-sqlc-sqlite test-queries-raw-postgres test-queries-sqlc-postgres
	@echo "✓ All query tests passed (4 modes)"

.PHONY: test-queries-raw-sqlite
test-queries-raw-sqlite:
	go test ./internal/queries -count=1 -race

.PHONY: test-queries-sqlc-sqlite
test-queries-sqlc-sqlite:
	go test -tags sqlc ./internal/queries -count=1 -race

.PHONY: test-queries-raw-postgres
test-queries-raw-postgres:
	@docker compose -f docker/docker-compose.yml -f docker/docker-compose_dev.yml up -d pgtestdb
	TEST_DB_DRIVER=postgres \
	TEST_POSTGRES_DSN='postgres://postgres:password@localhost:5433/postgres?sslmode=disable' \
	go test ./internal/queries -count=1 -race

.PHONY: test-queries-sqlc-postgres
test-queries-sqlc-postgres:
	@docker compose -f docker/docker-compose.yml -f docker/docker-compose_dev.yml up -d pgtestdb
	TEST_DB_DRIVER=postgres \
	TEST_POSTGRES_DSN='postgres://postgres:password@localhost:5433/postgres?sslmode=disable' \
	go test -tags sqlc ./internal/queries -count=1 -race
```

- [ ] **Step 8: Verify infrastructure compiles**

```bash
go build ./...
```

Expected: Builds successfully

- [ ] **Step 9: Verify sqlc build compiles**

```bash
go build -tags sqlc ./...
```

Expected: Builds successfully with sqlc tag

- [ ] **Step 10: Run existing tests (default mode)**

```bash
go test ./internal/queries -count=1 -race
```

Expected: All tests pass

- [ ] **Step 11: Run tests with sqlc tag**

```bash
go test -tags sqlc ./internal/queries -count=1 -race
```

Expected: All tests pass (using raw SQL since no migrations yet)

- [ ] **Step 12: Commit infrastructure**

```bash
git add sqlc.yaml \
        db/queries/ \
        internal/queries/backend_sqlc.go \
        internal/queries/queries_sqlc.go \
        internal/queries/queries_stub.go \
        internal/queries/queries.go \
        Makefile

git commit -m "feat(sqlc): add infrastructure for optional sqlc support

- Add sqlc.yaml configuration for postgres and sqlite
- Create backend_sqlc interface (placeholder)
- Add queries_sqlc.go (build tag sqlc) and queries_stub.go (build tag !sqlc)
- Modify queries.go to call initSqlc
- Add Makefile targets for sqlc generation and 4-mode testing
- Both default and sqlc builds compile successfully
- All existing tests pass in both modes

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Plan Complete (Tasks 2-10 follow same pattern)

Plan complete and saved to `docs/superpowers/plans/2026-09-15-sqlc-optional-support.md`.

**Task 1** is fully detailed above. **Tasks 2-10** follow the identical migration pattern shown in the design spec for each query group:

- **Task 2:** Auth queries (1 method)
- **Task 3:** Session queries (3 methods)  
- **Task 4:** Install queries (2 methods)
- **Task 5:** Setting queries (5 methods)
- **Task 6:** Pages queries (6 methods)
- **Task 7:** Product queries (15+ methods)
- **Task 8:** Cart queries (20+ methods)
- **Task 9:** Customer queries (10+ methods)
- **Task 10:** Documentation and README updates

Each migration task (2-9) follows these steps:
1. Analyze existing methods
2. Create postgres SQL file
3. Create sqlite SQL file
4. Run `sqlc generate`
5. Update backend_sqlc.go interface
6. Update queries_sqlc.go backend adapters
7. Add sqlc field to query struct
8. Create *_sqlc.go wrapper file
9. Test all 4 modes (raw/sqlc × SQLite/PostgreSQL)
10. Commit

**Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
