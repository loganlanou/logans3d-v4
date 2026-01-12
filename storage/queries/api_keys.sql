-- name: CreateAPIKey :one
INSERT INTO api_keys (
    id,
    name,
    key_hash,
    key_prefix,
    permissions,
    is_active
)
VALUES (?, ?, ?, ?, ?, 1)
RETURNING *;

-- name: GetAPIKey :one
SELECT * FROM api_keys WHERE id = ?;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = ? AND is_active = 1;

-- name: ListAPIKeys :many
SELECT * FROM api_keys ORDER BY created_at DESC;

-- name: ListActiveAPIKeys :many
SELECT * FROM api_keys WHERE is_active = 1 ORDER BY created_at DESC;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys
SET last_used_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeactivateAPIKey :exec
UPDATE api_keys
SET is_active = 0
WHERE id = ?;

-- name: ActivateAPIKey :exec
UPDATE api_keys
SET is_active = 1
WHERE id = ?;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = ?;

-- name: UpdateAPIKeyName :one
UPDATE api_keys
SET name = ?
WHERE id = ?
RETURNING *;

-- name: CountActiveAPIKeys :one
SELECT COUNT(*) as count FROM api_keys WHERE is_active = 1;
