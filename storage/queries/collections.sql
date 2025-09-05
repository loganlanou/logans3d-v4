-- name: GetUserCollections :many
SELECT 
    c.id,
    c.user_id,
    c.name,
    c.description,
    c.is_quote_requested,
    c.quote_request_id,
    c.created_at,
    c.updated_at,
    COUNT(ci.id) as item_count
FROM user_collections c
LEFT JOIN collection_items ci ON c.id = ci.collection_id
WHERE c.user_id = ?
GROUP BY c.id
ORDER BY c.updated_at DESC;

-- name: GetCollection :one
SELECT * FROM user_collections
WHERE id = ? AND user_id = ?;

-- name: CreateCollection :one
INSERT INTO user_collections (id, user_id, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING *;

-- name: UpdateCollection :one
UPDATE user_collections 
SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
RETURNING *;

-- name: DeleteCollection :exec
DELETE FROM user_collections
WHERE id = ? AND user_id = ?;

-- name: GetCollectionItems :many
SELECT 
    ci.id,
    ci.collection_id,
    ci.product_id,
    ci.quantity,
    ci.notes,
    ci.created_at,
    p.name AS product_name,
    p.slug AS product_slug,
    p.price_cents,
    p.short_description,
    pi.image_url AS primary_image_url
FROM collection_items ci
INNER JOIN products p ON ci.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
WHERE ci.collection_id = ?
ORDER BY ci.created_at DESC;

-- name: AddToCollection :one
INSERT INTO collection_items (id, collection_id, product_id, quantity, notes, created_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(collection_id, product_id) 
DO UPDATE SET quantity = excluded.quantity, notes = excluded.notes
RETURNING *;

-- name: RemoveFromCollection :exec
DELETE FROM collection_items
WHERE collection_id = ? AND product_id = ?;

-- name: UpdateCollectionItem :one
UPDATE collection_items
SET quantity = ?, notes = ?
WHERE id = ?
RETURNING *;

-- name: MarkCollectionAsQuoted :exec
UPDATE user_collections
SET is_quote_requested = TRUE, quote_request_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?;