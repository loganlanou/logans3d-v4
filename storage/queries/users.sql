-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByClerkID :one
SELECT * FROM users WHERE clerk_id = ?;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = ?;

-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    clerk_id,
    first_name,
    last_name,
    full_name,
    username,
    profile_image_url,
    google_id,
    legacy_avatar_url,
    last_synced_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET
    email = ?,
    first_name = ?,
    last_name = ?,
    full_name = ?,
    username = ?,
    profile_image_url = ?,
    last_synced_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateUserByClerkID :one
UPDATE users
SET
    email = ?,
    first_name = ?,
    last_name = ?,
    full_name = ?,
    username = ?,
    profile_image_url = ?,
    last_synced_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE clerk_id = ?
RETURNING *;

-- name: UpsertUserByClerkID :one
INSERT INTO users (
    id,
    clerk_id,
    email,
    first_name,
    last_name,
    full_name,
    username,
    profile_image_url,
    last_synced_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(clerk_id) DO UPDATE SET
    email = excluded.email,
    first_name = excluded.first_name,
    last_name = excluded.last_name,
    full_name = excluded.full_name,
    username = excluded.username,
    profile_image_url = excluded.profile_image_url,
    last_synced_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;