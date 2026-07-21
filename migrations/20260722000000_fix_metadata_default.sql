-- +goose Up
-- Fix metadata column default from '{}' to '[]' to match Go []models.Metadata type

-- Update existing records that have {} to []
UPDATE product SET metadata = '[]' WHERE metadata = '{}';

-- Note: SQLite doesn't support ALTER COLUMN DEFAULT directly
-- The default will need to be fixed in the schema for new installations
-- For existing installations, we've migrated the data

-- +goose Down
-- Revert metadata back to empty object (not recommended)
UPDATE product SET metadata = '{}' WHERE metadata = '[]' AND json_array_length(metadata) = 0;
