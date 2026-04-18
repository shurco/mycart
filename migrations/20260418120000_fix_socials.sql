-- +goose Up
-- +goose StatementBegin
-- Fix two social settings that never actually materialised.
--
-- Migration 20240111145752_new_sicials.sql used `INSERT OR IGNORE` with
-- setting ids `CoDDXfxF4GZxq6b` and `AC3of7o9pS9HdB1`. Those ids were
-- already taken by the init migration for `mail_letter_purchase` and
-- `smtp_host`, so the inserts were silently skipped and the keys
-- `social_youtube` / `social_other` never existed in the DB.
--
-- Re-inserting here with fresh, unique ids. The keys themselves are
-- `UNIQUE`, so `INSERT OR IGNORE` is still safe on installations that
-- somehow already contain them (e.g. hand-fixed by an operator).
INSERT OR IGNORE INTO setting VALUES ('yt1oK0n7cN6m9pB', 'social_youtube', '');
INSERT OR IGNORE INTO setting VALUES ('oT8hZ2q9vS3rW1L', 'social_other', '');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM setting WHERE key = 'social_youtube' AND id = 'yt1oK0n7cN6m9pB';
DELETE FROM setting WHERE key = 'social_other' AND id = 'oT8hZ2q9vS3rW1L';
-- +goose StatementEnd
