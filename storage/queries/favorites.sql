-- name: GetUserFavorites :many
SELECT 
    f.id,
    f.user_id,
    f.product_id,
    f.created_at,
    p.name AS product_name,
    p.slug AS product_slug,
    p.price_cents,
    p.short_description,
    pi.image_url AS primary_image_url
FROM user_favorites f
INNER JOIN products p ON f.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
WHERE f.user_id = ?
ORDER BY f.created_at DESC;

-- name: AddFavorite :one
INSERT INTO user_favorites (id, user_id, product_id, created_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP)
RETURNING *;

-- name: RemoveFavorite :exec
DELETE FROM user_favorites
WHERE user_id = ? AND product_id = ?;

-- name: IsFavorite :one
SELECT COUNT(*) as is_favorite
FROM user_favorites
WHERE user_id = ? AND product_id = ?;

-- name: GetFavoriteCount :one
SELECT COUNT(*) as count
FROM user_favorites
WHERE user_id = ?;