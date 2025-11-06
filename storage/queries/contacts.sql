-- name: CreateContactRequest :one
INSERT INTO contact_requests (
    id,
    first_name,
    last_name,
    email,
    phone,
    subject,
    message,
    newsletter_subscribe,
    ip_address,
    user_agent,
    referrer,
    status,
    priority,
    recaptcha_score
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetContactRequest :one
SELECT * FROM contact_requests WHERE id = ?;

-- name: ListContactRequests :many
SELECT * FROM contact_requests
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListActiveContactRequests :many
SELECT * FROM contact_requests
WHERE status NOT IN ('resolved', 'spam')
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: SearchContactRequests :many
SELECT * FROM contact_requests
WHERE
    (first_name LIKE ? OR
     last_name LIKE ? OR
     email LIKE ? OR
     message LIKE ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: FilterContactRequests :many
SELECT * FROM contact_requests
WHERE
    (sqlc.arg('status') IS NULL OR status = sqlc.arg('status'))
    AND (sqlc.arg('priority') IS NULL OR priority = sqlc.arg('priority'))
    AND (sqlc.arg('subject') IS NULL OR subject = sqlc.arg('subject'))
    AND (sqlc.arg('assigned_to') IS NULL OR assigned_to_user_id = sqlc.arg('assigned_to'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateContactRequestStatus :exec
UPDATE contact_requests
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateContactRequestPriority :exec
UPDATE contact_requests
SET priority = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: AssignContactRequest :exec
UPDATE contact_requests
SET assigned_to_user_id = ?, assigned_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: AddContactRequestResponse :exec
UPDATE contact_requests
SET responded_at = CURRENT_TIMESTAMP,
    response_notes = ?,
    status = 'responded',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateContactRequestTags :exec
UPDATE contact_requests
SET tags = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CountContactRequests :one
SELECT COUNT(*) FROM contact_requests;

-- name: CountContactRequestsByStatus :one
SELECT COUNT(*) FROM contact_requests WHERE status = ?;

-- name: GetContactRequestStats :one
SELECT
    COUNT(*) as total,
    SUM(CASE WHEN status = 'new' THEN 1 ELSE 0 END) as new_count,
    SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) as in_progress_count,
    SUM(CASE WHEN status = 'responded' THEN 1 ELSE 0 END) as responded_count,
    SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END) as resolved_count,
    SUM(CASE WHEN priority = 'urgent' THEN 1 ELSE 0 END) as urgent_count,
    SUM(CASE WHEN priority = 'high' THEN 1 ELSE 0 END) as high_count
FROM contact_requests;

-- name: DeleteContactRequest :exec
DELETE FROM contact_requests WHERE id = ?;
