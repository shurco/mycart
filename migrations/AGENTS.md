# migrations/AGENTS.md

Goose-managed SQL migrations embedded into the binary via `embed.go`.

## Conventions

- Filename: `YYYYMMDDHHMMSS_short_description.sql`.
- Each file contains `-- +goose Up` and `-- +goose Down` sections, each
  wrapped in `-- +goose StatementBegin` / `-- +goose StatementEnd` for
  multi-statement scripts.
- **Down must be safe to run.** It should reverse the Up section, not
  touch rows the earlier init migration owned. See the comment in
  `20240111145752_new_sicials.sql` for a real historic footgun: the old
  Down deleted rows by id, but two ids collided with `init_db`, which
  would have wiped `mail_letter_purchase` and `smtp_host` on rollback.
  The fix is to filter by `key`, not `id`.
- **Unique IDs.** Setting rows use a pseudo-random 15-char id string.
  Before inserting a new id, search the entire `migrations/` tree to
  ensure no collision (`rg "'id_here'"`).
- Prefer `INSERT OR IGNORE` for idempotent seed data. If the row must be
  refreshed, use `INSERT OR REPLACE` and document the reason.

## Running locally

```bash
./scripts/migration dev up   # apply migrations + test fixtures
./scripts/migration dev down # rollback the most recent migration
```

## Test fixtures

Fixture SQL (created by `./scripts/migration dev up`) lives outside this
folder and populates the DB with demo products, an admin user and sample
carts. Never rely on fixture state from production migrations.
