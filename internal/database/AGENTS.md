# internal/database/AGENTS.md

The dialect layer. Everything that differs between SQLite and PostgreSQL lives
here, and nothing else in the codebase is allowed to know which engine is in
use. Read this before adding a query, a migration or a connection option.

## Layout

| File | Role |
|------|------|
| `dialect.go` | The `Dialect` interface: the irreducible set of differences. |
| `dialect_sqlite.go` / `dialect_postgres.go` | The two implementations. |
| `rebind.go` | `?` → `$n` rewriting (`rebindDollar`), a lexer that skips strings, identifiers and comments. |
| `conn.go` / `tx.go` | `Conn` and `Tx`: the wrappers the application uses. |
| `open.go` | Connecting, DSN handling, pool settings, the UTC check. |
| `migrate.go` | The goose runner for both engines. |
| `config.go` / `config_file.go` | Configuration resolution and `lc_base/config.json`. |

## The model

The application writes **one** dialect of SQL: the SQLite one — `?`
placeholders, `TRUE`/`FALSE`, `ON CONFLICT`, `CURRENT_TIMESTAMP`. `Conn.Rebind`
and the `Dialect` fragments translate it on the way to the driver. Portability
is maintained by *writing portable SQL*, not by branching per engine; a
construct that only one engine has belongs in `Dialect`.

## Rules

1. **`Conn` does not embed `*sql.DB`.** That is deliberate: embedding would
   expose the unbound methods and let a caller send `?` placeholders to
   PostgreSQL. Do not "simplify" it back.
2. **`Raw()` is for goose and test fixtures only.** `internal/queries` must
   never call it; `wrapper_invariant_test.go` fails the build if it does.
   `q.DB` in `queries/` *is* the wrapper — that is the sanctioned path.
3. **Adding a `Dialect` method costs two implementations and two tests.**
   Only add one when the difference is genuinely irreducible. Everything else
   belongs in portable SQL, and edits to portable SQL should keep
   `migrations/schema_conformance_test.go` green.
4. **PostgreSQL sessions are pinned to UTC** and `connectPostgres` fails fast
   if `SHOW timezone` says otherwise. Timestamps are stored without a time
   zone and read as unix seconds, so a non-UTC session shifts every stored
   date by the server's offset — the drift measured against
   `postgres:18-alpine` under `TZ=Asia/Seoul` is exactly 32400 s. Do not remove
   the pin or downgrade the check to a warning.
5. **`EXTRACT(EPOCH FROM …)::bigint`.** The cast is not optional: `EXTRACT`
   returns `numeric`, which does not scan into an `int64`.
6. **`JSONObject`/`JSONAgg`/`JSONValue`/`JSONBool` exist because the engines
   disagree.** PostgreSQL's `json_agg` emits `{"id" : null}` (spaces), SQLite's
   `json_group_array` emits `{"id":null}`. Never compare a JSON *string* — decode
   it (`internal/queries/json.go`), or the comparison will pass on SQLite and
   fail on PostgreSQL.
7. **Never log or return a raw DSN.** It carries the password. Use
   `Config.Redacted()`. The PostgreSQL connect error deliberately contains no
   part of the DSN: it reaches an unauthenticated caller through the install
   wizard (`/api/install/db/test`), which is why
   `describeConnectFailure` maps it to a fixed set of explanations.
8. **Configuration precedence:** flags → environment → `lc_base/config.json` →
   the built-in SQLite default. `Source` records which won; `Pinned()` is true
   for flags and the environment, and a pinned database cannot be changed from
   the wizard. `SourceWizard` and the default are changeable.
9. **Pool settings.** SQLite keeps the pool defaults — changing them changes
   behaviour that existing installations rely on. PostgreSQL is sized by
   `MYCART_DB_MAX_OPEN_CONNS`, `MYCART_DB_MAX_IDLE_CONNS` and
   `MYCART_DB_CONN_MAX_LIFETIME`; `pool_*` keys in a DSN are rejected with a
   message naming those variables, because pgx passes them to the server as
   GUCs and PostgreSQL answers `unrecognized configuration parameter`.

## Tests

`go test ./... -count=1` covers both engines; `TEST_DB_DRIVER=postgres` plus
`TEST_POSTGRES_DSN` switches to PostgreSQL. The PostgreSQL side is provisioned
by pgtestdb (see `internal/testutil/pgtest`) and connects through `Connect`, so
the session-timezone pin and the `SHOW timezone` check are on the path of every
PostgreSQL test. A change here is not finished until both runs are green, and
the non-UTC run is worth doing too — it is the only one where a lost pin shows
up as failing dates rather than passing quietly. Both run against the
`pgtestdb` service in `docker/docker-compose_dev.yml`, a tmpfs-backed
`postgres:18-alpine` on port 5433 that holds nothing worth keeping; `PGTESTDB_TZ`
is what puts that server in another zone:

```bash
PGTESTDB_TZ=Asia/Seoul docker compose -f docker/docker-compose.yml -f docker/docker-compose_dev.yml up -d pgtestdb
TEST_DB_DRIVER=postgres TEST_POSTGRES_DSN='postgres://postgres:password@localhost:5433/postgres?sslmode=disable' go test ./... -count=1
```

`NormalizePostgresDSN` is exported for the test suite: anything that builds a
PostgreSQL DSN for a test goes through it rather than restating the pin.
