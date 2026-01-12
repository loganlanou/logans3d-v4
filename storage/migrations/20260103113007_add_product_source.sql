-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN source_url TEXT;
ALTER TABLE products ADD COLUMN source_platform TEXT;
ALTER TABLE products ADD COLUMN designer_name TEXT;
ALTER TABLE products ADD COLUMN release_date DATETIME;

CREATE INDEX idx_products_source_url ON products(source_url);
CREATE INDEX idx_products_source_platform ON products(source_platform);
CREATE INDEX idx_products_designer_name ON products(designer_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- First, drop the indexes we added
DROP INDEX IF EXISTS idx_products_designer_name;
DROP INDEX IF EXISTS idx_products_source_platform;
DROP INDEX IF EXISTS idx_products_source_url;

-- Create temporary table without the new columns
CREATE TABLE products_backup AS
SELECT id, name, slug, description, short_description, price_cents, category_id,
       sku, stock_quantity, low_stock_threshold, weight_grams, dimensions_length_mm,
       dimensions_width_mm, dimensions_height_mm, lead_time_days, is_active, is_featured,
       created_at, updated_at, shipping_category, is_premium, is_new, seo_title,
       seo_description, seo_keywords, og_image_url, disclaimer, has_variants
FROM products;

-- Drop the old table
DROP TABLE products;

-- Recreate the original table structure (without source columns)
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
    shipping_category TEXT DEFAULT 'small' CHECK (shipping_category IN ('small', 'medium', 'large', 'xlarge')),
    is_premium BOOLEAN DEFAULT FALSE,
    is_new BOOLEAN DEFAULT FALSE,
    seo_title TEXT,
    seo_description TEXT,
    seo_keywords TEXT,
    og_image_url TEXT,
    disclaimer TEXT,
    has_variants BOOLEAN DEFAULT FALSE
);

-- Copy data back
INSERT INTO products SELECT * FROM products_backup;

-- Drop backup table
DROP TABLE products_backup;

-- Recreate existing indexes
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_slug ON products(slug);
CREATE INDEX idx_products_is_active ON products(is_active);
CREATE INDEX idx_products_is_featured ON products(is_featured);
CREATE UNIQUE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_shipping_category ON products(shipping_category);
CREATE INDEX idx_products_is_premium ON products(is_premium);
-- +goose StatementEnd
