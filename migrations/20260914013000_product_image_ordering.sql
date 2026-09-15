-- +goose Up
-- +goose StatementBegin

-- Add is_representative column to product_image table for marking main/representative images.
-- The position column was already added in 20260721120000_product_variants.sql
ALTER TABLE product_image ADD COLUMN is_representative BOOLEAN DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop the is_representative column when rolling back
ALTER TABLE product_image DROP COLUMN is_representative;

-- +goose StatementEnd
