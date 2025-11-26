-- +goose Up
-- Add has_variants flag to products table
ALTER TABLE products ADD COLUMN has_variants BOOLEAN DEFAULT FALSE;

-- Add product_sku_id to cart_items for SKU-based cart items
ALTER TABLE cart_items ADD COLUMN product_sku_id TEXT REFERENCES product_skus(id) ON DELETE CASCADE;

-- Add product_sku_id to order_items for SKU-based orders
ALTER TABLE order_items ADD COLUMN product_sku_id TEXT REFERENCES product_skus(id) ON DELETE SET NULL;

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we recreate tables

-- Recreate cart_items without product_sku_id
CREATE TABLE cart_items_new (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    user_id TEXT REFERENCES users(id),
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    product_variant_id TEXT REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK ((session_id IS NULL) != (user_id IS NULL))
);
INSERT INTO cart_items_new (id, session_id, user_id, product_id, product_variant_id, quantity, created_at, updated_at)
SELECT id, session_id, user_id, product_id, product_variant_id, quantity, created_at, updated_at FROM cart_items;
DROP TABLE cart_items;
ALTER TABLE cart_items_new RENAME TO cart_items;

-- Recreate order_items without product_sku_id
CREATE TABLE order_items_new (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    product_variant_id TEXT REFERENCES product_variants(id) ON DELETE SET NULL,
    quantity INTEGER NOT NULL,
    price_cents INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO order_items_new (id, order_id, product_id, product_variant_id, quantity, price_cents, created_at)
SELECT id, order_id, product_id, product_variant_id, quantity, price_cents, created_at FROM order_items;
DROP TABLE order_items;
ALTER TABLE order_items_new RENAME TO order_items;

-- Recreate products without has_variants (need full schema)
CREATE TABLE products_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    short_description TEXT,
    price_cents INTEGER NOT NULL,
    category_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    sku TEXT,
    stock_quantity INTEGER DEFAULT 0,
    weight_grams INTEGER,
    lead_time_days INTEGER,
    is_active BOOLEAN DEFAULT TRUE,
    is_featured BOOLEAN DEFAULT FALSE,
    is_new BOOLEAN DEFAULT FALSE,
    is_premium BOOLEAN DEFAULT FALSE,
    disclaimer TEXT,
    seo_title TEXT,
    seo_description TEXT,
    seo_keywords TEXT,
    og_image_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO products_new SELECT
    id, name, slug, description, short_description, price_cents, category_id,
    sku, stock_quantity, weight_grams, lead_time_days, is_active, is_featured,
    is_new, is_premium, disclaimer, seo_title, seo_description, seo_keywords, og_image_url,
    created_at, updated_at
FROM products;
DROP TABLE products;
ALTER TABLE products_new RENAME TO products;
