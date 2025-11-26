-- +goose Up
-- Product SKUs (Style + Size combination = purchasable item)
CREATE TABLE product_skus (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    product_style_id TEXT NOT NULL REFERENCES product_styles(id) ON DELETE CASCADE,
    size_id TEXT NOT NULL REFERENCES sizes(id) ON DELETE CASCADE,
    sku TEXT NOT NULL UNIQUE,
    price_adjustment_cents INTEGER,
    stock_quantity INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, product_style_id, size_id)
);

CREATE INDEX idx_product_skus_product ON product_skus(product_id);
CREATE INDEX idx_product_skus_style ON product_skus(product_style_id);
CREATE INDEX idx_product_skus_sku ON product_skus(sku);

-- +goose Down
DROP INDEX IF EXISTS idx_product_skus_sku;
DROP INDEX IF EXISTS idx_product_skus_style;
DROP INDEX IF EXISTS idx_product_skus_product;
DROP TABLE IF EXISTS product_skus;
