-- +goose Up
-- +goose StatementBegin

-- Add columns for LLM-generated content
-- original_description: preserves the scraped description for auditing/regeneration
-- generated_name: cleaned product name (removes printing jargon)
-- generated_description: the converted "finished product" description
-- description_model: which LLM model generated it (e.g., "mistral:7b")
-- description_generated_at: when the content was generated

ALTER TABLE scraped_products ADD COLUMN original_description TEXT;
ALTER TABLE scraped_products ADD COLUMN generated_name TEXT;
ALTER TABLE scraped_products ADD COLUMN generated_description TEXT;
ALTER TABLE scraped_products ADD COLUMN description_model TEXT;
ALTER TABLE scraped_products ADD COLUMN description_generated_at DATETIME;

-- Migrate existing description to original_description
UPDATE scraped_products
SET original_description = description
WHERE description IS NOT NULL AND description != '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
-- This preserves data in the original columns

CREATE TABLE scraped_products_backup AS SELECT
    id,
    designer_slug,
    platform,
    source_url,
    name,
    description,
    original_price_cents,
    release_date,
    image_urls,
    tags,
    raw_html,
    scraped_at,
    imported_product_id,
    ai_category,
    ai_price_cents,
    ai_size,
    is_skipped,
    skip_reason
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
    imported_product_id TEXT REFERENCES products(id),
    ai_category TEXT,
    ai_price_cents INTEGER,
    ai_size TEXT,
    is_skipped BOOLEAN DEFAULT FALSE,
    skip_reason TEXT
);

INSERT INTO scraped_products SELECT * FROM scraped_products_backup;
DROP TABLE scraped_products_backup;

CREATE INDEX idx_scraped_products_designer ON scraped_products(designer_slug);
CREATE INDEX idx_scraped_products_imported ON scraped_products(imported_product_id);

-- +goose StatementEnd
