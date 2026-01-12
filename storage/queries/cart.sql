-- name: AddToCart :exec
INSERT INTO cart_items (id, session_id, user_id, product_id, product_sku_id, quantity)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetExistingCartItem :one
SELECT id, quantity FROM cart_items
WHERE (session_id = ? OR user_id = ?)
AND product_id = ?
AND COALESCE(product_sku_id, '') = COALESCE(?, '')
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
    ci.product_sku_id,
    p.name,
    (p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) AS price_cents,
    COALESCE(
        CASE WHEN psi.image_url IS NOT NULL THEN 'styles/' || psi.image_url END,
        pi.image_url,
        ''
    ) as image_url,
    COALESCE(ps.sku, '') as variant_sku,
    COALESCE(pst.name || ' - ' || sz.display_name, '') as variant_name,
    COALESCE(ps.stock_quantity, p.stock_quantity, 0) as stock_quantity,
    COALESCE(c.name, '') as category_name
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
LEFT JOIN product_styles pst ON ps.product_style_id = pst.id
LEFT JOIN sizes sz ON ps.size_id = sz.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
LEFT JOIN product_style_images psi ON psi.product_style_id = pst.id AND psi.is_primary = TRUE
WHERE ci.session_id = ?
ORDER BY ci.created_at DESC;

-- name: GetCartByUser :many
SELECT
    ci.id,
    ci.quantity,
    ci.product_id,
    ci.product_sku_id,
    p.name,
    (p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) AS price_cents,
    COALESCE(
        CASE WHEN psi.image_url IS NOT NULL THEN 'styles/' || psi.image_url END,
        pi.image_url,
        ''
    ) as image_url,
    COALESCE(ps.sku, '') as variant_sku,
    COALESCE(pst.name || ' - ' || sz.display_name, '') as variant_name,
    COALESCE(ps.stock_quantity, p.stock_quantity, 0) as stock_quantity,
    COALESCE(c.name, '') as category_name
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
LEFT JOIN product_styles pst ON ps.product_style_id = pst.id
LEFT JOIN sizes sz ON ps.size_id = sz.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
LEFT JOIN product_style_images psi ON psi.product_style_id = pst.id AND psi.is_primary = TRUE
WHERE ci.user_id = sqlc.arg(user_id)
ORDER BY ci.created_at DESC;

-- name: GetCartTotal :one
SELECT SUM((p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) * ci.quantity) as total
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
WHERE (ci.session_id = ? OR ci.user_id = ?);

-- name: ClearCart :exec
DELETE FROM cart_items WHERE session_id = ? OR user_id = ?;

-- name: TransferCartToUser :exec
UPDATE cart_items
SET user_id = sqlc.arg(user_id), session_id = NULL, updated_at = CURRENT_TIMESTAMP
WHERE session_id = sqlc.arg(session_id) AND user_id IS NULL;
