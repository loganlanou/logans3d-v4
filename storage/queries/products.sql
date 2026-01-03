-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

-- name: GetProductWithImage :one
SELECT
    p.*,
    pi.image_url as primary_image_url
FROM products p
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = TRUE
WHERE p.id = ?;

-- name: GetProductBySlug :one
SELECT * FROM products WHERE slug = ? AND is_active = TRUE;

-- name: ListProducts :many
SELECT * FROM products
WHERE is_active = TRUE
ORDER BY created_at DESC;

-- name: ListAllProducts :many
SELECT * FROM products
ORDER BY created_at DESC;

-- name: ListProductsByCategory :many
SELECT * FROM products 
WHERE category_id = ? AND is_active = TRUE 
ORDER BY created_at DESC;

-- name: ListFeaturedProducts :many
SELECT * FROM products 
WHERE is_featured = TRUE AND is_active = TRUE 
ORDER BY created_at DESC;

-- name: SearchProducts :many
SELECT * FROM products 
WHERE (name LIKE '%' || ? || '%' OR description LIKE '%' || ? || '%') 
AND is_active = TRUE 
ORDER BY created_at DESC;

-- name: CreateProduct :one
INSERT INTO products (
    id, name, slug, description, short_description, price_cents,
    category_id, sku, stock_quantity, has_variants, weight_grams, lead_time_days,
    is_active, is_featured, is_premium, disclaimer, seo_title, seo_description, seo_keywords, og_image_url
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET name = ?, slug = ?, description = ?, short_description = ?,
    price_cents = ?, category_id = ?, sku = ?, stock_quantity = ?, has_variants = ?,
    weight_grams = ?, lead_time_days = ?, is_active = ?, is_featured = ?, is_premium = ?, disclaimer = ?,
    seo_title = ?, seo_description = ?, seo_keywords = ?, og_image_url = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateProductFields :one
UPDATE products
SET name = ?, slug = ?, description = ?, short_description = ?,
    price_cents = ?, category_id = ?, sku = ?, stock_quantity = ?, has_variants = ?,
    weight_grams = ?, lead_time_days = ?, disclaimer = ?,
    seo_title = ?, seo_description = ?, seo_keywords = ?, og_image_url = ?,
    shipping_category = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = ?;

-- name: UpdateProductStock :exec
UPDATE products
SET stock_quantity = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DecrementProductStock :exec
UPDATE products
SET stock_quantity = stock_quantity - sqlc.arg(delta), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND stock_quantity >= sqlc.arg(delta);

-- name: GetProductImages :many
SELECT * FROM product_images 
WHERE product_id = ? 
ORDER BY display_order ASC, is_primary DESC;

-- name: GetProductByName :one
SELECT * FROM products WHERE name = ?;

-- name: UpsertProduct :one
INSERT INTO products (
    id, name, slug, description, price_cents, category_id,
    stock_quantity, has_variants, is_active, is_featured, is_premium, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, FALSE, TRUE, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON CONFLICT(name) DO UPDATE SET
    slug = excluded.slug,
    description = excluded.description,
    price_cents = excluded.price_cents,
    category_id = excluded.category_id,
    stock_quantity = excluded.stock_quantity,
    is_featured = excluded.is_featured,
    is_premium = excluded.is_premium,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetPrimaryProductImage :one
SELECT * FROM product_images 
WHERE product_id = ? AND is_primary = TRUE;

-- name: CreateProductImage :one
INSERT INTO product_images (
    id, product_id, image_url, alt_text, display_order, is_primary, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP
)
RETURNING *;

-- name: UpdateProductImage :exec
UPDATE product_images
SET image_url = ?, alt_text = ?
WHERE id = ?;

-- name: DeleteProductImage :exec
DELETE FROM product_images WHERE id = ?;

-- name: UnsetAllPrimaryProductImages :exec
UPDATE product_images
SET is_primary = FALSE
WHERE product_id = ?;

-- name: SetPrimaryProductImage :exec
UPDATE product_images
SET is_primary = TRUE
WHERE id = ?;

-- name: UpdateProductPrice :exec
UPDATE products
SET price_cents = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ToggleProductFeatured :one
UPDATE products
SET is_featured = NOT COALESCE(is_featured, FALSE), updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: ToggleProductPremium :one
UPDATE products
SET is_premium = NOT COALESCE(is_premium, FALSE), updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: ToggleProductActive :one
UPDATE products
SET is_active = NOT COALESCE(is_active, TRUE), updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: ToggleProductNew :one
UPDATE products
SET is_new = NOT COALESCE(is_new, FALSE), updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: SetProductVariantsFlag :exec
UPDATE products
SET has_variants = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListRelatedProducts :many
SELECT * FROM products
WHERE category_id = ? AND id != ? AND is_active = TRUE
ORDER BY RANDOM()
LIMIT ?;

-- name: UpdateProductInline :one
UPDATE products
SET name = ?, slug = ?, price_cents = ?, stock_quantity = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- Source tracking queries for importer

-- name: GetProductBySourceURL :one
SELECT * FROM products WHERE source_url = ?;

-- name: ListProductsBySourcePlatform :many
SELECT * FROM products
WHERE source_platform = ?
ORDER BY created_at DESC;

-- name: ListProductsByDesigner :many
SELECT * FROM products
WHERE designer_name = ?
ORDER BY created_at DESC;

-- name: UpdateProductSource :exec
UPDATE products
SET source_url = ?, source_platform = ?, designer_name = ?, release_date = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CreateProductWithSource :one
INSERT INTO products (
    id, name, slug, description, short_description, price_cents,
    category_id, sku, stock_quantity, has_variants, weight_grams, lead_time_days,
    is_active, is_featured, is_premium, is_new, disclaimer,
    seo_title, seo_description, seo_keywords, og_image_url,
    source_url, source_platform, designer_name, release_date
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListNewProducts :many
SELECT * FROM products
WHERE is_new = TRUE AND is_active = TRUE
ORDER BY release_date DESC, created_at DESC;

-- name: ClearExpiredNewFlags :exec
UPDATE products
SET is_new = FALSE, updated_at = CURRENT_TIMESTAMP
WHERE is_new = TRUE
  AND release_date IS NOT NULL
  AND release_date < datetime('now', '-6 months');

-- name: CountProductsByDesigner :one
SELECT COUNT(*) as count FROM products WHERE designer_name = ?;

-- name: ListDesigners :many
SELECT DISTINCT designer_name, COUNT(*) as product_count
FROM products
WHERE designer_name IS NOT NULL
GROUP BY designer_name
ORDER BY designer_name;
