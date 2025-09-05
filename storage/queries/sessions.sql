-- name: CreateSession :one
INSERT INTO user_sessions (id, user_id, session_token, expires_at, created_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetSessionByToken :one
SELECT 
    s.*,
    u.id AS user_id,
    u.email,
    u.name,
    u.google_id,
    u.avatar_url
FROM user_sessions s
INNER JOIN users u ON s.user_id = u.id
WHERE s.session_token = ? AND s.expires_at > CURRENT_TIMESTAMP;

-- name: DeleteSession :exec
DELETE FROM user_sessions
WHERE session_token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM user_sessions
WHERE expires_at < CURRENT_TIMESTAMP;

-- name: DeleteUserSessions :exec
DELETE FROM user_sessions
WHERE user_id = ?;