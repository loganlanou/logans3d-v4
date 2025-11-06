-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN seo_title TEXT;
ALTER TABLE products ADD COLUMN seo_description TEXT;
ALTER TABLE products ADD COLUMN seo_keywords TEXT;
ALTER TABLE products ADD COLUMN og_image_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
CREATE TABLE products_backup AS SELECT
    id, name, slug, description, short_description, price_cents,
    category_id, sku, stock_quantity, low_stock_threshold, weight_grams,
    dimensions_length_mm, dimensions_width_mm, dimensions_height_mm,
    lead_time_days, is_active, is_featured, created_at, updated_at,
    shipping_category, is_premium, is_new
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
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    shipping_category TEXT DEFAULT 'small'
        CHECK (shipping_category IN ('small', 'medium', 'large', 'xlarge')),
    is_premium BOOLEAN DEFAULT FALSE,
    is_new BOOLEAN DEFAULT FALSE
);

INSERT INTO products SELECT * FROM products_backup;
DROP TABLE products_backup;

-- Recreate indexes
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_slug ON products(slug);
CREATE INDEX idx_products_is_active ON products(is_active);
CREATE INDEX idx_products_is_featured ON products(is_featured);
CREATE UNIQUE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_shipping_category ON products(shipping_category);
CREATE INDEX idx_products_is_premium ON products(is_premium);
-- +goose StatementEnd
