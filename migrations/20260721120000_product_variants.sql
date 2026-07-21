-- +goose Up
-- +goose StatementBegin

-- Add new columns to product table
ALTER TABLE product ADD COLUMN quantity INTEGER DEFAULT 0;
ALTER TABLE product ADD COLUMN sku TEXT;
ALTER TABLE product ADD COLUMN has_variants BOOLEAN DEFAULT FALSE;

-- Create unique index on sku (only if not null)
CREATE UNIQUE INDEX idx_product_sku ON product (sku) WHERE sku IS NOT NULL;

-- Add position to product_image for ordering
ALTER TABLE product_image ADD COLUMN position INTEGER DEFAULT 0;

-- Product options (e.g., "Size", "Color", "Material")
CREATE TABLE product_option (
    id          TEXT PRIMARY KEY NOT NULL,
    product_id  TEXT NOT NULL,
    name        TEXT NOT NULL,
    position    INTEGER DEFAULT 0,
    created     TIMESTAMP DEFAULT (datetime('now')),
    FOREIGN KEY (product_id) REFERENCES product(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_option_product_id ON product_option (product_id);

-- Option values (e.g., "Medium", "Large", "Black", "Orange")
CREATE TABLE product_option_value (
    id          TEXT PRIMARY KEY NOT NULL,
    option_id   TEXT NOT NULL,
    value       TEXT NOT NULL,
    position    INTEGER DEFAULT 0,
    FOREIGN KEY (option_id) REFERENCES product_option(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_option_value_option_id ON product_option_value (option_id);

-- Product variants (combinations of option values)
CREATE TABLE product_variant (
    id              TEXT PRIMARY KEY NOT NULL,
    product_id      TEXT NOT NULL,
    sku             TEXT,
    price_surcharge NUMERIC DEFAULT 0,
    quantity        INTEGER DEFAULT 0,
    option_values   TEXT NOT NULL DEFAULT '{}',
    active          BOOLEAN DEFAULT TRUE,
    deleted         BOOLEAN DEFAULT FALSE,
    created         TIMESTAMP DEFAULT (datetime('now')),
    updated         TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES product(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_product_id ON product_variant (product_id);
CREATE UNIQUE INDEX idx_product_variant_sku ON product_variant (sku) WHERE sku IS NOT NULL;

-- Junction table: links variants to their option value combinations
CREATE TABLE product_variant_option (
    variant_id      TEXT NOT NULL,
    option_value_id TEXT NOT NULL,
    PRIMARY KEY (variant_id, option_value_id),
    FOREIGN KEY (variant_id) REFERENCES product_variant(id) ON DELETE CASCADE,
    FOREIGN KEY (option_value_id) REFERENCES product_option_value(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_option_variant ON product_variant_option (variant_id);
CREATE INDEX idx_product_variant_option_value ON product_variant_option (option_value_id);

-- Variant-specific images
CREATE TABLE product_variant_image (
    id          TEXT PRIMARY KEY NOT NULL,
    variant_id  TEXT NOT NULL,
    name        TEXT NOT NULL,
    ext         TEXT NOT NULL,
    orig_name   TEXT NOT NULL,
    position    INTEGER DEFAULT 0,
    FOREIGN KEY (variant_id) REFERENCES product_variant(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_image_variant_id ON product_variant_image (variant_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_product_variant_image_variant_id;
DROP TABLE IF EXISTS product_variant_image;

DROP INDEX IF EXISTS idx_product_variant_option_value;
DROP INDEX IF EXISTS idx_product_variant_option_variant;
DROP TABLE IF EXISTS product_variant_option;

DROP INDEX IF EXISTS idx_product_variant_sku;
DROP INDEX IF EXISTS idx_product_variant_product_id;
DROP TABLE IF EXISTS product_variant;

DROP INDEX IF EXISTS idx_product_option_value_option_id;
DROP TABLE IF EXISTS product_option_value;

DROP INDEX IF EXISTS idx_product_option_product_id;
DROP TABLE IF EXISTS product_option;

ALTER TABLE product_image DROP COLUMN position;

DROP INDEX IF EXISTS idx_product_sku;
-- SQLite doesn't support DROP COLUMN directly, so we skip these in down migration
-- In production, document that rollback requires manual intervention

-- +goose StatementEnd
