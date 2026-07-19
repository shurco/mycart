-- +goose Up
-- +goose StatementBegin
INSERT INTO setting VALUES ('portone_001', 'portone_active', 'false');
INSERT INTO setting VALUES ('portone_002', 'portone_store_id', '');
INSERT INTO setting VALUES ('portone_003', 'portone_channel_key', '');
INSERT INTO setting VALUES ('portone_004', 'portone_api_secret', '');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM setting WHERE id = 'portone_004';
DELETE FROM setting WHERE id = 'portone_003';
DELETE FROM setting WHERE id = 'portone_002';
DELETE FROM setting WHERE id = 'portone_001';
-- +goose StatementEnd
