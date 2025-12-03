-- name: CreatePendingAIImage :one
INSERT INTO pending_ai_images (id, product_id, source_image_url, generated_image_url, model_used, status, created_at)
VALUES (?, ?, ?, ?, ?, 'pending', CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetPendingAIImage :one
SELECT * FROM pending_ai_images WHERE id = ?;

-- name: GetPendingAIImagesByProduct :many
SELECT * FROM pending_ai_images
WHERE product_id = ? AND status = 'pending'
ORDER BY created_at DESC;

-- name: GetAllPendingAIImages :many
SELECT pai.*, p.name as product_name
FROM pending_ai_images pai
JOIN products p ON pai.product_id = p.id
WHERE pai.status = 'pending'
ORDER BY pai.created_at DESC;

-- name: UpdatePendingAIImageStatus :exec
UPDATE pending_ai_images
SET status = ?, reviewed_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeletePendingAIImage :exec
DELETE FROM pending_ai_images WHERE id = ?;

-- name: CleanupOldRejectedImages :exec
DELETE FROM pending_ai_images
WHERE status = 'rejected' AND reviewed_at < datetime('now', '-7 days');
