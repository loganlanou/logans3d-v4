-- name: GetExistingCartItem :one
SELECT id, quantity FROM cart_items 
WHERE (session_id = ? OR user_id = ?) AND product_id = ?
LIMIT 1;

-- name: AddToCart :exec
INSERT INTO cart_items (id, session_id, user_id, product_id, quantity, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- name: GetCartBySession :many
SELECT c.id, c.session_id, c.user_id, c.product_id, c.product_variant_id, c.quantity, c.created_at, c.updated_at, 
       p.name, p.price_cents, p.slug, 
       COALESCE(pi.image_url, '') as image_url
FROM cart_items c
JOIN products p ON c.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND (pi.is_primary = 1 OR pi.display_order = 1)
WHERE c.session_id = ?
GROUP BY c.id
ORDER BY c.created_at DESC;

-- name: GetCartByUser :many
SELECT c.id, c.session_id, c.user_id, c.product_id, c.product_variant_id, c.quantity, c.created_at, c.updated_at, 
       p.name, p.price_cents, p.slug, 
       COALESCE(pi.image_url, '') as image_url
FROM cart_items c
JOIN products p ON c.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND (pi.is_primary = 1 OR pi.display_order = 1)
WHERE c.user_id = ?
GROUP BY c.id
ORDER BY c.created_at DESC;

-- name: UpdateCartItemQuantity :exec
UPDATE cart_items 
SET quantity = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: RemoveCartItem :exec
DELETE FROM cart_items WHERE id = ?;

-- name: ClearCart :exec
DELETE FROM cart_items WHERE session_id = ? OR user_id = ?;

-- name: GetCartItemCount :one
SELECT COALESCE(SUM(quantity), 0) as count
FROM cart_items 
WHERE session_id = ? OR user_id = ?;

-- name: GetCartTotal :one
SELECT COALESCE(SUM(c.quantity * p.price_cents), 0) as total_cents
FROM cart_items c
JOIN products p ON c.product_id = p.id
WHERE c.session_id = ? OR c.user_id = ?;

-- name: MergeCartOnLogin :exec
UPDATE cart_items 
SET user_id = ?, session_id = NULL, updated_at = CURRENT_TIMESTAMP
WHERE session_id = ?
  AND NOT EXISTS (
    SELECT 1 FROM cart_items c2 
    WHERE c2.user_id = ? AND c2.product_id = cart_items.product_id
  );