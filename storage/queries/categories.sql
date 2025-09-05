-- name: GetCategory :one
SELECT * FROM categories WHERE id = ?;

-- name: GetCategoryBySlug :one
SELECT * FROM categories WHERE slug = ?;

-- name: ListCategories :many
SELECT * FROM categories ORDER BY display_order ASC, name ASC;

-- name: ListRootCategories :many
SELECT * FROM categories WHERE parent_id IS NULL ORDER BY display_order ASC, name ASC;

-- name: ListCategoriesByParent :many
SELECT * FROM categories WHERE parent_id = ? ORDER BY display_order ASC, name ASC;

-- name: CreateCategory :one
INSERT INTO categories (id, name, slug, description, parent_id, display_order)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateCategory :one
UPDATE categories 
SET name = ?, slug = ?, description = ?, parent_id = ?, display_order = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = ?;

-- name: GetCategoryByName :one
SELECT * FROM categories WHERE name = ?;

-- name: UpsertCategory :one
INSERT INTO categories (id, name, slug, created_at, updated_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(name) DO UPDATE SET
    slug = excluded.slug,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;