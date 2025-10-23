-- name: AddToCart :exec
INSERT INTO cart_items (id, session_id, user_id, product_id, product_variant_id, quantity)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetExistingCartItem :one
SELECT id, quantity FROM cart_items
WHERE (session_id = ? OR user_id = ?) AND product_id = ?
LIMIT 1;

-- name: UpdateCartItemQuantity :exec
UPDATE cart_items SET quantity = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: RemoveCartItem :exec
DELETE FROM cart_items WHERE id = ?;

-- name: GetCartBySession :many
SELECT 
    ci.id,
    ci.quantity,
    ci.product_id,
    p.name,
    p.price_cents,
    COALESCE(pi.image_url, '') as image_url
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
WHERE ci.session_id = ?
ORDER BY ci.created_at DESC;

-- name: GetCartByUser :many
SELECT
    ci.id,
    ci.quantity,
    ci.product_id,
    p.name,
    p.price_cents,
    COALESCE(pi.image_url, '') as image_url
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
WHERE ci.user_id = sqlc.arg(user_id)
ORDER BY ci.created_at DESC;

-- name: GetCartTotal :one
SELECT SUM(p.price_cents * ci.quantity) as total
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
WHERE (ci.session_id = ? OR ci.user_id = ?);

-- name: ClearCart :exec
DELETE FROM cart_items WHERE session_id = ? OR user_id = ?;

-- name: TransferCartToUser :exec
UPDATE cart_items
SET user_id = sqlc.arg(user_id), session_id = NULL, updated_at = CURRENT_TIMESTAMP
WHERE session_id = sqlc.arg(session_id) AND user_id IS NULL;