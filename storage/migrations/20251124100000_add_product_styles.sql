-- +goose Up
-- Product Styles (product-specific, NOT global)
-- Each product has its own styles with its own images
CREATE TABLE product_styles (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, name)
);

CREATE INDEX idx_product_styles_product ON product_styles(product_id);

-- Style Images (linked to product_styles, NOT global)
CREATE TABLE product_style_images (
    id TEXT PRIMARY KEY,
    product_style_id TEXT NOT NULL REFERENCES product_styles(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_product_style_images_style ON product_style_images(product_style_id);

-- +goose Down
DROP INDEX IF EXISTS idx_product_style_images_style;
DROP TABLE IF EXISTS product_style_images;
DROP INDEX IF EXISTS idx_product_styles_product;
DROP TABLE IF EXISTS product_styles;
