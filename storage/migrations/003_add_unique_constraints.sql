-- +goose Up
-- Add unique constraints for seeding/upsert operations

-- Add unique constraint on category name
CREATE UNIQUE INDEX idx_categories_name ON categories(name);

-- Add unique constraint on product name  
CREATE UNIQUE INDEX idx_products_name ON products(name);

-- Add unique constraint on product_images for primary image per product
CREATE UNIQUE INDEX idx_product_images_primary ON product_images(product_id, is_primary) 
WHERE is_primary = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_categories_name;
DROP INDEX IF EXISTS idx_products_name;
DROP INDEX IF EXISTS idx_product_images_primary;