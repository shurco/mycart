# AGENTS.md

Concise, progressive-disclosure guide for AI coding agents working on
**myCart** (formerly *litecart*): a single-binary e-commerce backend written
in Go + SQLite with two SvelteKit frontends (admin panel and storefront).

Start here, then descend into the directory-scoped `AGENTS.md` files when
touching that subtree.

---

## 1. Orientation

- **Language/runtime:** Go 1.26, SvelteKit (Svelte 5), TailwindCSS v4.
- **Database:** embedded SQLite via `modernc.org/sqlite` (pure Go, no CGO) by
  default, PostgreSQL via `github.com/jackc/pgx/v5` when the operator chooses it
  at install time. One query layer serves both — see
  `internal/database/AGENTS.md` before writing SQL.
- **Migrations:** [`goose`](https://github.com/pressly/goose) SQL files in
  `migrations/`, one shared set for both engines.
- **Entrypoint:** `cmd/main.go` → `internal/app.go` (Fiber v3 HTTP server).
- **Distribution:** single binary with frontends embedded via `//go:embed`.

Repo layout:

| Path | Purpose |
|------|---------|
| `cmd/` | `main` package and runtime-writable `lc_base/`, `lc_uploads/`, `lc_digitals/` dirs used in dev. |
| `internal/` | Private application code (HTTP handlers, DB queries, middleware, mailer, webhooks). |
| `internal/database/` | The only place that knows which engine is in use: connection, configuration, dialects, migrations runner. |
| `pkg/` | Reusable packages that could in theory live in their own repo (`litepay`, `jwtutil`, `httpclient`, `webutil`, …). |
| `web/admin/` | SvelteKit admin panel, served at `/_/`. |
| `web/site/` | SvelteKit storefront, served at `/`. |
| `migrations/` | Goose SQL migrations, embedded via `migrations/embed.go`. |
| `docs/` | User-facing documentation. |
| `scripts/` | Developer convenience scripts (see README). |

---

## 2. Build / Test / Run

```bash
# Go
go build ./...
go vet ./...
go test ./... -count=1 -race

# The same suite on PostgreSQL. Without TEST_DB_DRIVER the tests run on
# in-memory SQLite; the PostgreSQL run is what proves a change is portable.
#
# pgtestdb is the throwaway server from docker/docker-compose_dev.yml: a
# tmpfs-backed postgres:18-alpine on port 5433, holding nothing worth keeping.
# It is not the `postgres` service in that file, which holds a real shop's data.
#
# TEST_POSTGRES_DSN is an administrator connection to a *dedicated* test
# server: pgtestdb creates a role (pgtdbuser) and template databases on it and
# drops the per-test databases again. Never point it at a server with data
# anybody wants to keep.
docker compose -f docker/docker-compose.yml -f docker/docker-compose_dev.yml up -d pgtestdb
TEST_DB_DRIVER=postgres \
TEST_POSTGRES_DSN='postgres://postgres:password@localhost:5433/postgres?sslmode=disable' \
go test ./... -count=1 -race

# Admin SPA
cd web/admin && bun install && bun run build

# Storefront SPA
cd web/site && bun install && bun run build

# Run locally (serves admin at /_/ and storefront at /)
go run ./cmd serve
```

Default admin credentials after `./scripts/migration dev up`:
`user@mail.com` / `Pass123`.

---

## 3. Coding Conventions

- **KISS / DRY / SRP.** Prefer small, single-purpose functions. If a
  `switch` grows across handlers, promote it to a registry (see
  `internal/handlers/private/setting_registry.go` for the canonical
  example).
- **Errors.** Wrap with `fmt.Errorf("context: %w", err)`. Compare with
  `errors.Is` / `errors.As`, never `==`. The custom helper
  `pkg/errors.ErrorStack` produces annotated stack traces for logs.
- **Resource management.** Never `defer` inside a loop. Extract the
  per-iteration body into a helper so `defer` runs per call (see
  `scanDigitalFiles` in `internal/queries/cart.go`).
- **HTTP clients.** Never use `http.DefaultClient` or an ad-hoc
  `http.Client{}` for outbound calls. Use `pkg/httpclient.New()` or
  `pkg/httpclient.NewWithTimeout(...)` to inherit the shared timeout
  profile (dial 10 s, TLS 10 s, overall 30 s).
- **JWT.** Sign/verify via `pkg/jwtutil`. The parser enforces HMAC-only
  signing to block "alg confusion" attacks; do not loosen that check.
- **Passwords & tokens.** Hash passwords with `bcrypt.DefaultCost`
  (`pkg/security`). Never introduce MD5 — `NewToken` intentionally uses
  `bcrypt` + SHA-256.
- **Pagination.** Use `webutil.ParsePagination(c)` in list handlers. It
  clamps to `[1, 100]` items per page.
- **SQL safety.** Always parameterised queries, through the wrapper in
  `internal/database` — never `*sql.DB` directly, never `?`-only SQL sent to
  PostgreSQL. Write portable SQL: `ON CONFLICT … DO UPDATE` instead of
  `INSERT OR REPLACE`, `TRUE`/`FALSE` instead of `1`/`0`,
  `CURRENT_TIMESTAMP` instead of `datetime('now')`, and dialect fragments
  (epoch conversion, JSON aggregation) through `Dialect`.
- **Dates.** Timestamps are stored without a time zone and read back as unix
  seconds. The PostgreSQL session is pinned to UTC and the process refuses to
  start otherwise; do not work around that pin, and do not format a stored
  timestamp with the process's local time zone.

Frontend (SvelteKit / Svelte 5):

- Use runes (`$state`, `$derived`, `$effect`, `$props`, `$bindable`).
- Prefer `{@render children?.()}` over legacy `<slot />`.
- Any rendering of user-authored HTML **must** go through
  `sanitizeHTML()` (`$lib/utils/sanitize.ts`) before `{@html}`.
- Always clear outstanding `setTimeout` handles in `onDestroy` to avoid
  writing to state after unmount (see `Drawer.svelte`).

---

## 4. Testing Standards

- Files: `*_test.go` next to the code under test.
- Style: table-driven, parallel (`t.Parallel()`), `t.Cleanup()` /
  `t.TempDir()` / `t.Setenv()` instead of hand-rolled teardown.
- Helpers live in `internal/testutil/`: `SetupTestDB` / `SetupCleanDB` /
  `SetupTestApp` give each test a migrated database and install it as the
  process-wide handle. They are dialect-parameterised — the same test runs on
  SQLite and on PostgreSQL according to `TEST_DB_DRIVER`, so never hard-code
  `sql.Open("sqlite", …)` or `:memory:` in a test.
- On PostgreSQL each test gets a database of its own, cloned by pgtestdb from a
  template that already holds the schema and the fixtures, so the migrations run
  once per server rather than once per test. `internal/testutil/pgtest` owns
  that; its AGENTS.md explains the constraints on `TEST_POSTGRES_DSN`.
- Every public function should have at least one happy-path and one
  error-path test. Integration-style tests for handlers live in
  `internal/handlers/*/...*_test.go`.

---

## 5. Security Checklist (for any change)

- [ ] No secret values (passwords, tokens, API keys) committed.
- [ ] No new outbound HTTP call without `pkg/httpclient`.
- [ ] JWT signing method assertion preserved in any new verifier.
- [ ] User-authored HTML sanitised on the frontend.
- [ ] New migrations have a working, non-destructive `Down`.
- [ ] New SQL is portable (see `migrations/AGENTS.md`) and goes through the
      dialect wrapper.
- [ ] Nothing new logs a connection string: use `Config.Redacted()`.
- [ ] Error responses do not leak internals (`log.ErrorStack`, but
      return `StatusInternalServerError` to clients).

---

## 6. Descending Further

For deeper, directory-scoped guidance, read the nearest `AGENTS.md`:

- `internal/AGENTS.md` — handler, query, middleware, webhook layers.
- `internal/database/AGENTS.md` — the dialect layer, connections and
  configuration; read this before adding a query.
- `pkg/AGENTS.md` — public-ish library packages.
- `web/AGENTS.md` — both SvelteKit apps.
- `migrations/AGENTS.md` — migration authoring, portability and the
  append-only rule.
