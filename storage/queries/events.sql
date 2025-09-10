-- name: GetEvent :one
SELECT * FROM events WHERE id = ?;

-- name: ListEvents :many
SELECT * FROM events 
ORDER BY start_date DESC;

-- name: ListActiveEvents :many
SELECT * FROM events 
WHERE is_active = TRUE 
ORDER BY start_date ASC;

-- name: ListUpcomingEvents :many
SELECT * FROM events 
WHERE is_active = TRUE AND start_date >= DATE('now')
ORDER BY start_date ASC;

-- name: ListPastEvents :many
SELECT * FROM events 
WHERE is_active = TRUE AND start_date < DATE('now')
ORDER BY start_date DESC;

-- name: CreateEvent :one
INSERT INTO events (
    id, title, description, location, address, start_date, end_date, url, is_active
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateEvent :one
UPDATE events 
SET title = ?, description = ?, location = ?, address = ?, 
    start_date = ?, end_date = ?, url = ?, is_active = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateEventActiveStatus :one
UPDATE events 
SET is_active = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = ?;

-- name: GetEventStats :one
SELECT 
    COUNT(*) as total_events,
    COUNT(CASE WHEN is_active = TRUE THEN 1 END) as active_events,
    COUNT(CASE WHEN is_active = TRUE AND start_date >= DATE('now') THEN 1 END) as upcoming_events,
    COUNT(CASE WHEN is_active = TRUE AND start_date < DATE('now') THEN 1 END) as past_events
FROM events;