-- +goose Up
-- Sizes (global, standardized across all products)
CREATE TABLE sizes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Size Charts (global shipping defaults per size)
CREATE TABLE size_charts (
    id TEXT PRIMARY KEY,
    size_id TEXT NOT NULL REFERENCES sizes(id) ON DELETE CASCADE,
    default_shipping_class TEXT,
    default_shipping_weight_oz REAL,
    default_price_adjustment_cents INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(size_id)
);

CREATE INDEX idx_size_charts_size ON size_charts(size_id);

-- Seed default sizes
INSERT INTO sizes (id, name, display_name, display_order) VALUES
    ('size_small', 'small', 'Small', 1),
    ('size_medium', 'medium', 'Medium', 2),
    ('size_large', 'large', 'Large', 3),
    ('size_xlarge', 'x-large', 'X-Large', 4);

-- Seed default size charts with shipping defaults
INSERT INTO size_charts (id, size_id, default_shipping_class, default_shipping_weight_oz, default_price_adjustment_cents) VALUES
    ('chart_small', 'size_small', 'First', 4.0, 0),
    ('chart_medium', 'size_medium', 'First', 8.0, 200),
    ('chart_large', 'size_large', 'Priority', 16.0, 500),
    ('chart_xlarge', 'size_xlarge', 'Priority', 32.0, 800);

-- +goose Down
DROP INDEX IF EXISTS idx_size_charts_size;
DROP TABLE IF EXISTS size_charts;
DROP TABLE IF EXISTS sizes;
