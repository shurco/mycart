# AGENTS.md

Concise, progressive-disclosure guide for AI coding agents working on
**myCart** (formerly *litecart*): a single-binary e-commerce backend written
in Go + SQLite with two SvelteKit frontends (admin panel and storefront).

Start here, then descend into the directory-scoped `AGENTS.md` files when
touching that subtree.

---

## 1. Orientation

- **Language/runtime:** Go 1.26, SvelteKit (Svelte 5), TailwindCSS v4.
- **Database:** embedded SQLite via `modernc.org/sqlite` (pure Go, no CGO).
- **Migrations:** [`goose`](https://github.com/pressly/goose) SQL files in `migrations/`.
- **Entrypoint:** `cmd/main.go` → `internal/app.go` (Fiber v3 HTTP server).
- **Distribution:** single binary with frontends embedded via `//go:embed`.

Repo layout:

| Path | Purpose |
|------|---------|
| `cmd/` | `main` package and runtime-writable `lc_base/`, `lc_uploads/`, `lc_digitals/` dirs used in dev. |
| `internal/` | Private application code (HTTP handlers, DB queries, middleware, mailer, webhooks). |
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
- **SQL safety.** Always parameterised queries. Use `INSERT OR REPLACE`
  for idempotent session writes (`queries.AddSession`).

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
- Use [`testify`](https://github.com/stretchr/testify)
  `require`/`assert`; helpers live in `internal/testutil/`.
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
- [ ] Error responses do not leak internals (`log.ErrorStack`, but
      return `StatusInternalServerError` to clients).

---

## 6. Descending Further

For deeper, directory-scoped guidance, read the nearest `AGENTS.md`:

- `internal/AGENTS.md` — handler, query, middleware, webhook layers.
- `pkg/AGENTS.md` — public-ish library packages.
- `web/AGENTS.md` — both SvelteKit apps.
- `migrations/AGENTS.md` — migration authoring and pitfalls.
