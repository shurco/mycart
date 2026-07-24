-- +goose Up
-- +goose StatementBegin
INSERT INTO setting VALUES ('portone_005', 'portone_debug_enabled', 'false');
INSERT INTO setting VALUES ('portone_006', 'portone_supported_currencies', '["KRW"]');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM setting WHERE id IN ('portone_005', 'portone_006');
-- +goose StatementEnd
