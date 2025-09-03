-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

-- name: GetProductBySlug :one
SELECT * FROM products WHERE slug = ? AND is_active = TRUE;

-- name: ListProducts :many
SELECT * FROM products 
WHERE is_active = TRUE 
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
    is_active, is_featured
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products 
SET name = ?, slug = ?, description = ?, short_description = ?, 
    price_cents = ?, category_id = ?, sku = ?, stock_quantity = ?, 
    weight_grams = ?, lead_time_days = ?, is_active = ?, is_featured = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = ?;

-- name: UpdateProductStock :exec
UPDATE products 
SET stock_quantity = ?, updated_at = CURRENT_TIMESTAMP 
WHERE id = ?;