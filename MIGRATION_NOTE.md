# PortOne Migration Note

The PortOne payment gateway migration (`migrations/20260714000001_portone.sql`) has been created and will be automatically applied when the application starts.

## Migration Contents

The migration adds the following settings to the database:
- `portone_active` (default: false)
- `portone_store_id` (default: empty)
- `portone_channel_key` (default: empty)
- `portone_api_secret` (default: empty)

## How Migrations Work

Migrations in myCart are:
1. Embedded in the Go binary at compile time (`migrations/embed.go`)
2. Automatically executed on application startup
3. Run in order by timestamp prefix
4. Tracked to prevent duplicate execution

## Manual Migration (if needed)

If you need to run the migration manually on an existing database:

```bash
sqlite3 docker/lc_base/data.db << 'SQL'
INSERT INTO setting VALUES ('portone_001', 'portone_active', 'false');
INSERT INTO setting VALUES ('portone_002', 'portone_store_id', '');
INSERT INTO setting VALUES ('portone_003', 'portone_channel_key', '');
INSERT INTO setting VALUES ('portone_004', 'portone_api_secret', '');
SQL
```

## Verification

After the application starts, verify the migration by checking the admin panel:
Settings → Payment → PortOne

You should see the PortOne configuration form with fields for:
- Store ID
- Channel Key
- API Secret
- Active toggle
