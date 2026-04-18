# internal/AGENTS.md

Private application code. Not importable outside this module.

## Layout

| Path | Role |
|------|------|
| `app.go` / `init.go` | Bootstraps Fiber, DB, mailer, routes. |
| `base/` | Global runtime state (version, embed flags). |
| `config/` | Runtime configuration parsing. |
| `handlers/private/` | Admin API handlers (`/api/_/…`). |
| `handlers/public/` | Public storefront API handlers (`/api/…`). |
| `middleware/` | Fiber middlewares (JWT auth, CORS, logging). |
| `models/` | Plain data structs shared by queries + handlers. |
| `queries/` | All SQL lives here. One file per logical domain. |
| `routes/` | Router wiring + SPA fallback. |
| `mailer/` | SMTP + templated email rendering. |
| `webhook/` | Outbound webhook dispatch. |
| `testutil/` | In-memory SQLite fixture + Fiber harness. |

## Rules

1. **Handlers are thin.** They parse input, call queries, format the
   response. Business logic belongs in `queries/` or a service helper.
2. **No `http.DefaultClient`.** Webhooks use `webhook.sharedClient` which
   is built from `pkg/httpclient`.
3. **Pagination** uses `webutil.ParsePagination`. Clamp is `[1, 100]`.
4. **Setting keys** map to models via `setting_registry.go`. Adding a new
   setting group is a one-line change there — do not bring back the
   per-switch branches.
5. **Error taxonomy.** Return sentinel errors from `queries/` (e.g.
   `queries.ErrAlreadyInstalled`), translate to HTTP codes in handlers
   with `errors.Is`.
6. **Sessions table** is idempotent (`INSERT OR REPLACE`) — callers may
   refresh the same key without a prior delete.
7. **Tests** use `internal/testutil` for a fresh SQLite DB per test and
   `t.Cleanup` for teardown. Always run with `-race`.
