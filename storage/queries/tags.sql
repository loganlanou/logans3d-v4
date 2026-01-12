-- name: CreateTag :one
INSERT INTO tags (id, name, slug)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetTag :one
SELECT * FROM tags WHERE id = ?;

-- name: GetTagBySlug :one
SELECT * FROM tags WHERE slug = ?;

-- name: ListTags :many
SELECT * FROM tags ORDER BY name ASC;

-- name: UpdateTag :one
UPDATE tags
SET name = ?, slug = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: AddProductTag :exec
INSERT OR IGNORE INTO product_tags (product_id, tag_id)
VALUES (?, ?);

-- name: RemoveProductTag :exec
DELETE FROM product_tags
WHERE product_id = ? AND tag_id = ?;

-- name: GetProductTags :many
SELECT t.*
FROM tags t
JOIN product_tags pt ON pt.tag_id = t.id
WHERE pt.product_id = ?
ORDER BY t.name ASC;

-- name: GetProductsByTag :many
SELECT p.*
FROM products p
JOIN product_tags pt ON pt.product_id = p.id
WHERE pt.tag_id = ?
ORDER BY p.name ASC;

-- name: GetProductsByTagSlug :many
SELECT p.*
FROM products p
JOIN product_tags pt ON pt.product_id = p.id
JOIN tags t ON t.id = pt.tag_id
WHERE t.slug = ?
ORDER BY p.name ASC;

-- name: ClearProductTags :exec
DELETE FROM product_tags WHERE product_id = ?;

-- name: CountProductsWithTag :one
SELECT COUNT(*) as count
FROM product_tags
WHERE tag_id = ?;

-- name: GetTagsWithProductCounts :many
SELECT
    t.*,
    COUNT(pt.product_id) as product_count
FROM tags t
LEFT JOIN product_tags pt ON pt.tag_id = t.id
GROUP BY t.id
ORDER BY t.name ASC;
