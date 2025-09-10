-- name: GetQuoteRequest :one
SELECT * FROM quote_requests WHERE id = ?;

-- name: GetQuoteRequestWithFiles :one
SELECT 
    q.*,
    GROUP_CONCAT(
        qf.id || ',' || qf.filename || ',' || qf.original_filename || ',' || 
        qf.file_path || ',' || qf.file_size || ',' || qf.mime_type
    ) as quote_files
FROM quote_requests q
LEFT JOIN quote_files qf ON q.id = qf.quote_request_id
WHERE q.id = ?
GROUP BY q.id;

-- name: ListQuoteRequests :many
SELECT * FROM quote_requests 
ORDER BY created_at DESC;

-- name: ListQuoteRequestsByStatus :many
SELECT * FROM quote_requests 
WHERE status = ? 
ORDER BY created_at DESC;

-- name: CreateQuoteRequest :one
INSERT INTO quote_requests (
    id, customer_name, customer_email, customer_phone, project_description,
    quantity, material_preference, finish_preference, deadline_date, budget_range,
    status, admin_notes, quoted_price_cents
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateQuoteRequestStatus :one
UPDATE quote_requests 
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateQuoteRequestPrice :one
UPDATE quote_requests 
SET quoted_price_cents = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateQuoteRequestNotes :one
UPDATE quote_requests 
SET admin_notes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateQuoteRequest :one
UPDATE quote_requests 
SET customer_name = ?, customer_email = ?, customer_phone = ?, project_description = ?,
    quantity = ?, material_preference = ?, finish_preference = ?, deadline_date = ?, 
    budget_range = ?, status = ?, admin_notes = ?, quoted_price_cents = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteQuoteRequest :exec
DELETE FROM quote_requests WHERE id = ?;

-- name: GetQuoteFiles :many
SELECT * FROM quote_files WHERE quote_request_id = ?;

-- name: CreateQuoteFile :one
INSERT INTO quote_files (
    id, quote_request_id, filename, original_filename, file_path, file_size, mime_type
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteQuoteFile :exec
DELETE FROM quote_files WHERE id = ?;

-- name: GetQuoteStats :one
SELECT 
    COUNT(*) as total_quotes,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_quotes,
    COUNT(CASE WHEN status = 'reviewing' THEN 1 END) as reviewing_quotes,
    COUNT(CASE WHEN status = 'quoted' THEN 1 END) as quoted_quotes,
    COUNT(CASE WHEN status = 'approved' THEN 1 END) as approved_quotes,
    COUNT(CASE WHEN status = 'rejected' THEN 1 END) as rejected_quotes,
    SUM(CASE WHEN quoted_price_cents IS NOT NULL THEN quoted_price_cents ELSE 0 END) as total_quoted_value_cents,
    AVG(CASE WHEN quoted_price_cents IS NOT NULL THEN quoted_price_cents ELSE 0 END) as average_quote_value_cents
FROM quote_requests;