-- name: GetSiteConfig :one
SELECT value FROM site_config WHERE key = ?;

-- name: GetAllSiteConfig :many
SELECT key, value, updated_at FROM site_config
ORDER BY key;

-- name: SetSiteConfig :exec
INSERT INTO site_config (key, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(key) DO UPDATE SET
    value = excluded.value,
    updated_at = CURRENT_TIMESTAMP;

-- name: DeleteSiteConfig :exec
DELETE FROM site_config WHERE key = ?;
