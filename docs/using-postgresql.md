# Using PostgreSQL

myCart runs on an embedded SQLite database by default — one file in `./lc_base`, nothing to install — and on
PostgreSQL 13 or newer if you prefer a database server. **You choose once, when you install the cart.** An existing
installation stays on SQLite and nothing about its upgrade path changes.

## Which one?

| | SQLite (default) | PostgreSQL |
|---|---|---|
| Setup | none | a server to run, back up and patch |
| Data location | `./lc_base/data.db` | the server, optionally elsewhere |
| Backups | copy the folder | `mycart db backup` |
| Concurrent writers | one at a time | many |
| Fits | a single small shop, a VPS, a laptop | many writers, managed backups, an existing DBA |

Both run the same code and the same migrations. There is no reduced-feature mode.

## Installing on PostgreSQL

Create the database and a user for it first — myCart creates its tables, not the database:

```bash
createuser --pwprompt mycart
createdb --owner mycart mycart
```

Then either:

**From the setup wizard.** Open `/_/install`, choose *PostgreSQL*, paste the connection string, press **Test
connection** (it reports the server version and whether the database already holds a cart), then install.

**From the command line.**

```bash
./mycart install \
  --email admin@example.com \
  --password 'YourSecurePass' \
  --domain example.com \
  --db postgres \
  --db-dsn 'postgres://mycart:secret@db.example.com:5432/mycart?sslmode=disable'
```

The choice is written to `./lc_base/config.json` — mode `0600`, because it holds the password — and every later
command reads it from there. `./mycart serve`, `./mycart migrate` and `./mycart update` need no further flags.

The connection string may also be a set of `key=value` pairs:

```
host=db.example.com port=5432 dbname=mycart user=mycart password=secret sslmode=disable
```

### Without the wizard

`MYCART_DB_DRIVER` and `MYCART_DB_DSN`, or `--db` and `--db-dsn`, take precedence over `config.json`:

```bash
MYCART_DB_DRIVER=postgres \
MYCART_DB_DSN='postgres://mycart:secret@db.example.com:5432/mycart?sslmode=disable' \
./mycart serve
```

A database configured this way is **fixed**: the wizard shows it as locked and refuses to install anywhere else,
since silently connecting to a different database than the operator asked for is never right.

## Moving an existing shop from SQLite to PostgreSQL

One command. Stop the service first, keep a copy of `./lc_base` and of `./lc_uploads` and `./lc_digitals`, then:

```bash
./mycart db copy \
  --from ./lc_base/data.db \
  --to 'postgres://mycart:secret@db.example.com:5432/mycart?sslmode=disable' \
  --dry-run
```

`--dry-run` reports how many rows each table would contribute, checks that the target can take them, and writes
nothing. Drop the flag to do it for real, then point the installation at PostgreSQL and start it:

```bash
./mycart --db postgres --db-dsn 'postgres://mycart:secret@db.example.com:5432/mycart?sslmode=disable' serve
```

`--from` is a SQLite file (or `sqlite:./lc_base/data.db`) or a PostgreSQL connection string; `--to` is the
PostgreSQL target and defaults to the configured database. The whole copy is one transaction:

* the target is migrated first, so it does not have to be prepared;
* it is emptied table by table before the data lands, because a migrated database is not empty — the migrations
  seed `setting` and `page`;
* rows are written in foreign-key order, so a child row is never inserted before its parent;
* the row count of every table is checked after the insert, and anything that does not match rolls the whole thing
  back.

A target that already holds an installation is refused unless you pass `--force`. Nothing is deleted from the
SQLite side: the old cart keeps working until you change the driver, which is what makes the move reversible.

What is *not* moved: `./lc_uploads` (product images), `./lc_digitals` (sellable files) and `./site` are files, not
rows — copy them across as they are. There is no incremental sync and no move from a running service: the cart has
to be stopped while its database is being read, or the last few orders will be missing.

## Backups and maintenance

```bash
./mycart db backup --out mycart-$(date +%F).sql.gz     # write every row to one file
./mycart db restore --from mycart-2026-09-12.sql.gz     # put it back
```

`db backup` needs no server-side tools: it connects with the same driver the application uses and reads the tables
through PostgreSQL's own `COPY`, so it is available wherever `mycart` runs. The default output name is
`mycart-<timestamp>.sql.gz`; a path ending in `.gz` is compressed and anything else is written as plain text.

The file is a SQL script — a banner, a metadata comment, one `COPY … FROM stdin;` section per table, then a
summary comment carrying the row count of every table:

```sql
-- myCart database dump
-- {"magic":"mycart-dump","format":1,"created":"…","driver":"postgres","migrations":20260821000000}
COPY "product" ("id", "name", "desc", …) FROM stdin;
p1	Shoes	Leather	…
\.
-- {"magic":"mycart-dump-trailer","tables":[…],"rows":48}
```

That makes it a backup a person can read: the tables are greppable, the metadata says when it was taken and from
which myCart version, and the summary is what makes a file that lost its end somewhere detectable — a truncated
dump is refused instead of restoring a shop without its orders.

`db restore` migrates the target, empties it, loads each section through the server's own `COPY`, verifies the row
count of every table and commits. It is a single transaction: a dump that is truncated, mangled or does not fit
the schema leaves the database exactly as it was. The target must not already hold an installation — restoring into
one erases the shop that is in there — so it is refused with a message naming `--force`, which is the flag that
says you know.

**By hand with `psql`.** The file is valid SQL, and the header says how to replay it; the one thing to know is that
a migrated database is not empty, so the target's tables have to be cleared in the same transaction:

```bash
psql -1 "$MYCART_PG_DSN" \
  -c 'TRUNCATE TABLE "cart", "page", "product", … , "setting" CASCADE;' \
  -f mycart-2026-09-12.sql
```

Without the `TRUNCATE` this fails on a duplicate key rather than half-restoring anything, which is a useful second
net under `--force`.

`pg_dump` and `pg_restore` still work, of course — myCart is an ordinary PostgreSQL client and does not need to be
told where the database lives beyond its connection string. They are simply not the tools `db copy` and the
restore verification speak to.

`./lc_base` still matters — it holds `config.json`, and `./lc_uploads` and `./lc_digitals` live next to it — but the
data itself is no longer there. Back those three up alongside the dump.

Migrations run automatically when the application starts, and can be run explicitly:

```bash
./mycart migrate
```

## Troubleshooting

| Symptom | Cause |
|---|---|
| `postgres session timezone is "…", must be "UTC"` | The connection string carries its own `options` or `timezone`. Remove it and let myCart pin UTC, or set it to `UTC`. |
| `unrecognized configuration parameter "pool_max_conns"` | A `pgxpool` option in the DSN — use `MYCART_DB_MAX_OPEN_CONNS`. |
| `authentication failed: check the user and password` | The wizard's message for a rejected login; the same text hides the server's wording on purpose, since the form is reachable before installation. |
| `the database does not exist on that server` | Connect to an existing database; myCart does not create one. |
| Dates are off by whole hours | The session timezone is not UTC. myCart refuses to start in that state, so seeing this means something bypasses the check — report it. |

## Running the tests on PostgreSQL

```bash
docker compose -f docker/docker-compose.yml -f docker/docker-compose_dev.yml up -d pgtestdb

TEST_DB_DRIVER=postgres \
TEST_POSTGRES_DSN='postgres://postgres:password@localhost:5433/postgres?sslmode=disable' \
go test ./... -count=1
```

The `pgtestdb` service in `docker/docker-compose_dev.yml` is a throwaway `postgres:18-alpine` on port 5433: its data
directory is a tmpfs and its durability is off, so there is nothing in it to preserve or to lose.

`TEST_POSTGRES_DSN` is an administrator connection, not a database the suite works in. The tests provision their own
databases with [pgtestdb](https://github.com/peterldowns/pgtestdb): it creates a role (`pgtdbuser`), one template
database per schema variant, and one database per test, which it drops again when the test passes. The database named
in the connection string is only what pgtestdb connects to in order to administer the server, so it has to exist —
`postgres` above is the maintenance database every PostgreSQL server has.

That means the connection string must name a **dedicated test server**: the account in it needs `CREATEDB`,
`CREATEROLE` and `SUPERUSER`. Pointing it at a server whose data you care about is a mistake — myCart itself would
never do this, but the test suite will create and delete databases on whatever you name.

The migrations run once per server, into the template, rather than once per test, so a full run is fast and every test
still starts from a pristine, fully migrated database. A test that fails keeps its database and logs the connection
string so you can inspect it with `psql`; a test that passes has it dropped.
