-- name: CreateEmailHistory :one
INSERT INTO email_history (
    id,
    user_id,
    recipient_email,
    email_type,
    subject,
    template_name,
    tracking_token,
    metadata
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetEmailHistoryByID :one
SELECT * FROM email_history
WHERE id = ?;

-- name: GetEmailHistoryByTrackingToken :one
SELECT * FROM email_history
WHERE tracking_token = ?;

-- name: UpdateEmailOpened :exec
UPDATE email_history
SET opened_at = CURRENT_TIMESTAMP
WHERE tracking_token = ? AND opened_at IS NULL;

-- name: UpdateEmailClicked :exec
UPDATE email_history
SET clicked_at = CURRENT_TIMESTAMP
WHERE tracking_token = ? AND clicked_at IS NULL;

-- name: GetAllEmailHistory :many
SELECT * FROM email_history
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: GetEmailHistoryByUser :many
SELECT * FROM email_history
WHERE user_id = ?
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: GetEmailHistoryByEmail :many
SELECT * FROM email_history
WHERE recipient_email = ?
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: GetEmailHistoryByType :many
SELECT * FROM email_history
WHERE email_type = ?
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: SearchEmailHistory :many
SELECT * FROM email_history
WHERE
    (recipient_email LIKE '%' || ? || '%' OR subject LIKE '%' || ? || '%')
    AND (? = '' OR email_type = ?)
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: GetEmailHistoryStats :one
SELECT
    COUNT(*) as total_sent,
    SUM(CASE WHEN opened_at IS NOT NULL THEN 1 ELSE 0 END) as total_opened,
    SUM(CASE WHEN clicked_at IS NOT NULL THEN 1 ELSE 0 END) as total_clicked
FROM email_history
WHERE email_type = ?;

-- name: GetEmailHistoryStatsByDateRange :one
SELECT
    COUNT(*) as total_sent,
    SUM(CASE WHEN opened_at IS NOT NULL THEN 1 ELSE 0 END) as total_opened,
    SUM(CASE WHEN clicked_at IS NOT NULL THEN 1 ELSE 0 END) as total_clicked
FROM email_history
WHERE sent_at >= ? AND sent_at <= ?;

-- name: GetEmailHistoryByTypeAndDateRange :many
SELECT * FROM email_history
WHERE email_type = ?
    AND sent_at >= ?
    AND sent_at <= ?
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: CountEmailHistory :one
SELECT COUNT(*) FROM email_history;

-- name: CountEmailHistoryByType :one
SELECT COUNT(*) FROM email_history
WHERE email_type = ?;

-- name: CountEmailHistoryByDateRange :one
SELECT COUNT(*) FROM email_history
WHERE sent_at >= ? AND sent_at <= ?;

-- name: GetRecentEmailHistory :many
SELECT * FROM email_history
ORDER BY sent_at DESC
LIMIT ?;

-- name: DeleteOldEmailHistory :exec
DELETE FROM email_history
WHERE sent_at < ?;
