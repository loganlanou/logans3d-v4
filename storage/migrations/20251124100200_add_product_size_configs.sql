-- +goose Up
-- Product Size Configs (which sizes are available for this product + price overrides)
CREATE TABLE product_size_configs (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    size_id TEXT NOT NULL REFERENCES sizes(id) ON DELETE CASCADE,
    price_adjustment_cents INTEGER,
    is_enabled BOOLEAN DEFAULT TRUE,
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, size_id)
);

CREATE INDEX idx_product_size_configs_product ON product_size_configs(product_id);

-- +goose Down
DROP INDEX IF EXISTS idx_product_size_configs_product;
DROP TABLE IF EXISTS product_size_configs;
