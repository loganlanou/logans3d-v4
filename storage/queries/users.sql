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

-- name: ListUsersWithStats :many
SELECT
    u.id,
    u.email,
    u.full_name,
    u.first_name,
    u.last_name,
    u.username,
    u.profile_image_url,
    u.clerk_id,
    u.is_admin,
    u.created_at,
    u.last_synced_at,
    COUNT(DISTINCT o.id) as order_count,
    COALESCE(SUM(o.total_cents), 0) as lifetime_spend_cents,
    MAX(o.created_at) as last_order_date
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE
    u.clerk_id IS NOT NULL
    AND (
        sqlc.narg(search) IS NULL
        OR u.full_name LIKE '%' || sqlc.narg(search) || '%'
        OR u.email LIKE '%' || sqlc.narg(search) || '%'
        OR u.first_name LIKE '%' || sqlc.narg(search) || '%'
        OR u.last_name LIKE '%' || sqlc.narg(search) || '%'
    )
    AND (
        sqlc.narg(date_from) IS NULL
        OR u.created_at >= sqlc.narg(date_from)
    )
    AND (
        sqlc.narg(date_to) IS NULL
        OR u.created_at <= sqlc.narg(date_to)
    )
GROUP BY u.id
ORDER BY u.created_at DESC;

-- name: GetUserDetailWithStats :one
SELECT
    u.id,
    u.email,
    u.full_name,
    u.first_name,
    u.last_name,
    u.username,
    u.profile_image_url,
    u.clerk_id,
    u.is_admin,
    u.created_at,
    u.updated_at,
    u.last_synced_at,
    COUNT(DISTINCT o.id) as order_count,
    COALESCE(SUM(o.total_cents), 0) as lifetime_spend_cents,
    MAX(o.created_at) as last_order_date,
    COUNT(DISTINCT f.id) as favorites_count,
    COUNT(DISTINCT c.id) as collections_count,
    COUNT(DISTINCT CASE WHEN ci.updated_at > datetime('now', '-7 days') THEN ci.id END) as active_carts_count,
    COUNT(DISTINCT CASE WHEN ci.updated_at <= datetime('now', '-7 days') THEN ci.id END) as abandoned_carts_count
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
LEFT JOIN user_favorites f ON f.user_id = u.id
LEFT JOIN user_collections c ON c.user_id = u.id
LEFT JOIN cart_items ci ON ci.user_id = u.id
WHERE u.id = ?
GROUP BY u.id;

-- name: GetUserOrders :many
SELECT
    id,
    user_id,
    customer_email,
    customer_name,
    customer_phone,
    status,
    total_cents,
    created_at,
    updated_at
FROM orders
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: GetUserActiveCarts :many
SELECT
    ci.id,
    ci.session_id,
    ci.user_id,
    COUNT(*) as item_count,
    SUM(p.price_cents * ci.quantity) as total_cents,
    MAX(ci.updated_at) as last_activity
FROM cart_items ci
JOIN products p ON p.id = ci.product_id
WHERE
    ci.user_id = ?
    AND ci.updated_at > datetime('now', '-7 days')
GROUP BY ci.session_id, ci.user_id
ORDER BY last_activity DESC;

-- name: GetUserAbandonedCarts :many
SELECT
    ci.id,
    ci.session_id,
    ci.user_id,
    COUNT(*) as item_count,
    SUM(p.price_cents * ci.quantity) as total_cents,
    MAX(ci.updated_at) as last_activity
FROM cart_items ci
JOIN products p ON p.id = ci.product_id
WHERE
    ci.user_id = ?
    AND ci.updated_at <= datetime('now', '-7 days')
GROUP BY ci.session_id, ci.user_id
ORDER BY last_activity DESC;

-- name: GetUserRecentFavorites :many
SELECT
    p.id,
    p.name,
    p.slug,
    p.price_cents,
    pi.image_url,
    f.created_at as favorited_at
FROM user_favorites f
JOIN products p ON p.id = f.product_id
LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = true
WHERE f.user_id = ?
ORDER BY f.created_at DESC
LIMIT 10;

-- name: GetUserCollectionsList :many
SELECT
    id,
    name,
    description,
    is_quote_requested,
    created_at,
    updated_at
FROM user_collections
WHERE user_id = ?
ORDER BY created_at DESC;