-- +goose Up
-- +goose StatementBegin

-- Add skip functionality to scraped_products
ALTER TABLE scraped_products ADD COLUMN is_skipped BOOLEAN DEFAULT false;
ALTER TABLE scraped_products ADD COLUMN skip_reason TEXT;

-- Individual images scraped from source
CREATE TABLE scraped_product_images (
    id TEXT PRIMARY KEY,
    scraped_product_id TEXT NOT NULL REFERENCES scraped_products(id) ON DELETE CASCADE,
    source_url TEXT NOT NULL,
    local_filename TEXT,
    download_status TEXT DEFAULT 'pending',
    download_error TEXT,
    is_selected_for_import BOOLEAN DEFAULT true,
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_scraped_product_images_product ON scraped_product_images(scraped_product_id);
CREATE INDEX idx_scraped_product_images_status ON scraped_product_images(download_status);

-- AI-generated replacement images
CREATE TABLE scraped_product_ai_images (
    id TEXT PRIMARY KEY,
    scraped_product_id TEXT NOT NULL REFERENCES scraped_products(id) ON DELETE CASCADE,
    source_image_id TEXT REFERENCES scraped_product_images(id) ON DELETE SET NULL,
    local_filename TEXT NOT NULL,
    prompt_used TEXT,
    model_used TEXT,
    status TEXT DEFAULT 'pending',
    is_selected_for_import BOOLEAN DEFAULT false,
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_scraped_product_ai_images_product ON scraped_product_ai_images(scraped_product_id);
CREATE INDEX idx_scraped_product_ai_images_source ON scraped_product_ai_images(source_image_id);
CREATE INDEX idx_scraped_product_ai_images_status ON scraped_product_ai_images(status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_scraped_product_ai_images_status;
DROP INDEX IF EXISTS idx_scraped_product_ai_images_source;
DROP INDEX IF EXISTS idx_scraped_product_ai_images_product;
DROP TABLE IF EXISTS scraped_product_ai_images;

DROP INDEX IF EXISTS idx_scraped_product_images_status;
DROP INDEX IF EXISTS idx_scraped_product_images_product;
DROP TABLE IF EXISTS scraped_product_images;

-- SQLite doesn't support DROP COLUMN directly, but since these are nullable columns
-- added at the end, we can recreate the table without them
CREATE TABLE scraped_products_backup AS SELECT
    id, designer_slug, platform, source_url, name, description,
    original_price_cents, release_date, image_urls, tags, raw_html,
    scraped_at, imported_product_id, ai_category, ai_price_cents, ai_size
FROM scraped_products;

DROP TABLE scraped_products;

CREATE TABLE scraped_products (
    id TEXT PRIMARY KEY,
    designer_slug TEXT NOT NULL,
    platform TEXT NOT NULL,
    source_url TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    original_price_cents INTEGER,
    release_date DATETIME,
    image_urls TEXT,
    tags TEXT,
    raw_html TEXT,
    scraped_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    imported_product_id TEXT REFERENCES products(id) ON DELETE SET NULL,
    ai_category TEXT,
    ai_price_cents INTEGER,
    ai_size TEXT
);

INSERT INTO scraped_products SELECT * FROM scraped_products_backup;
DROP TABLE scraped_products_backup;

CREATE INDEX idx_scraped_products_designer ON scraped_products(designer_slug);
CREATE INDEX idx_scraped_products_platform ON scraped_products(platform);
CREATE INDEX idx_scraped_products_source_url ON scraped_products(source_url);
CREATE INDEX idx_scraped_products_imported ON scraped_products(imported_product_id);

-- +goose StatementEnd
