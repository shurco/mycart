-- +goose Up
-- +goose StatementBegin
INSERT OR IGNORE INTO setting VALUES ('CbA1K3y9XmN7pQ2', 'coinbase_api_key', '');
INSERT OR IGNORE INTO setting VALUES ('CbA2K4y0XnN8pR3', 'coinbase_active', 'false');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM setting WHERE id = 'CbA2K4y0XnN8pR3';
DELETE FROM setting WHERE id = 'CbA1K3y9XmN7pQ2';
-- +goose StatementEnd
