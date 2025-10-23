-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN is_premium BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_products_is_premium ON products(is_premium);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_products_is_premium;
-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
CREATE TABLE products_backup AS SELECT
    id, name, slug, description, short_description, price_cents,
    category_id, sku, stock_quantity, low_stock_threshold, weight_grams,
    dimensions_length_mm, dimensions_width_mm, dimensions_height_mm,
    lead_time_days, is_active, is_featured, created_at, updated_at
FROM products;

DROP TABLE products;

CREATE TABLE products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT,
    short_description TEXT,
    price_cents INTEGER NOT NULL DEFAULT 0,
    category_id TEXT REFERENCES categories(id),
    sku TEXT,
    stock_quantity INTEGER DEFAULT 0,
    low_stock_threshold INTEGER DEFAULT 5,
    weight_grams INTEGER,
    dimensions_length_mm INTEGER,
    dimensions_width_mm INTEGER,
    dimensions_height_mm INTEGER,
    lead_time_days INTEGER DEFAULT 7,
    is_active BOOLEAN DEFAULT TRUE,
    is_featured BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO products SELECT * FROM products_backup;
DROP TABLE products_backup;

CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_slug ON products(slug);
CREATE INDEX idx_products_is_active ON products(is_active);
CREATE INDEX idx_products_is_featured ON products(is_featured);
-- +goose StatementEnd
