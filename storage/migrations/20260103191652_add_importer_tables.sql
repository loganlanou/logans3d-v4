-- +goose Up
-- +goose StatementBegin

-- Import jobs table for tracking scrape/import operations
CREATE TABLE import_jobs (
    id TEXT PRIMARY KEY,
    designer_slug TEXT NOT NULL,
    platform TEXT NOT NULL,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    total_items INTEGER DEFAULT 0,
    processed_items INTEGER DEFAULT 0,
    error_message TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_import_jobs_designer ON import_jobs(designer_slug);
CREATE INDEX idx_import_jobs_status ON import_jobs(status);
CREATE INDEX idx_import_jobs_created ON import_jobs(created_at);

-- Scraped products cache table
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

CREATE INDEX idx_scraped_products_designer ON scraped_products(designer_slug);
CREATE INDEX idx_scraped_products_platform ON scraped_products(platform);
CREATE INDEX idx_scraped_products_source_url ON scraped_products(source_url);
CREATE INDEX idx_scraped_products_imported ON scraped_products(imported_product_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_scraped_products_imported;
DROP INDEX IF EXISTS idx_scraped_products_source_url;
DROP INDEX IF EXISTS idx_scraped_products_platform;
DROP INDEX IF EXISTS idx_scraped_products_designer;
DROP TABLE IF EXISTS scraped_products;

DROP INDEX IF EXISTS idx_import_jobs_created;
DROP INDEX IF EXISTS idx_import_jobs_status;
DROP INDEX IF EXISTS idx_import_jobs_designer;
DROP TABLE IF EXISTS import_jobs;

-- +goose StatementEnd
