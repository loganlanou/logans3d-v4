-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

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
    category_id, sku, stock_quantity, weight_grams, lead_time_days,
    is_active, is_featured, is_premium
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET name = ?, slug = ?, description = ?, short_description = ?,
    price_cents = ?, category_id = ?, sku = ?, stock_quantity = ?,
    weight_grams = ?, lead_time_days = ?, is_active = ?, is_featured = ?, is_premium = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = ?;

-- name: UpdateProductStock :exec
UPDATE products 
SET stock_quantity = ?, updated_at = CURRENT_TIMESTAMP 
WHERE id = ?;

-- name: GetProductImages :many
SELECT * FROM product_images 
WHERE product_id = ? 
ORDER BY display_order ASC, is_primary DESC;

-- name: GetProductByName :one
SELECT * FROM products WHERE name = ?;

-- name: UpsertProduct :one
INSERT INTO products (
    id, name, slug, description, price_cents, category_id,
    stock_quantity, is_active, is_featured, is_premium, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, TRUE, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
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

-- name: SetPrimaryProductImage :exec
UPDATE product_images
SET is_primary = CASE WHEN id = ? THEN TRUE ELSE FALSE END
WHERE product_id = ?;

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

-- name: ListRelatedProducts :many
SELECT * FROM products
WHERE category_id = ? AND id != ? AND is_active = TRUE
ORDER BY RANDOM()
LIMIT ?;