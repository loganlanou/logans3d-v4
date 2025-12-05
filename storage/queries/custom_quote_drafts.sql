-- Custom Quote Draft Management Queries

-- name: GetDraftBySessionID :one
SELECT * FROM custom_quote_drafts
WHERE session_id = ? AND completed_at IS NULL
ORDER BY updated_at DESC
LIMIT 1;

-- name: GetDraftByID :one
SELECT * FROM custom_quote_drafts
WHERE id = ?;

-- name: CreateDraft :one
INSERT INTO custom_quote_drafts (session_id, user_id, current_step, project_type)
VALUES (?, ?, 1, ?)
RETURNING *;

-- name: UpdateDraftStep1 :exec
UPDATE custom_quote_drafts
SET project_type = ?,
    current_step = CASE WHEN current_step < 2 THEN 2 ELSE current_step END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateDraftStep2 :exec
UPDATE custom_quote_drafts
SET name = ?,
    email = ?,
    current_step = CASE WHEN current_step < 3 THEN 3 ELSE current_step END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateDraftStep3 :exec
UPDATE custom_quote_drafts
SET material = ?,
    size = ?,
    budget = ?,
    color = ?,
    current_step = CASE WHEN current_step < 4 THEN 4 ELSE current_step END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateDraftStep4 :exec
UPDATE custom_quote_drafts
SET timeline = ?,
    description = ?,
    current_step = CASE WHEN current_step < 5 THEN 5 ELSE current_step END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateDraftOptions :exec
UPDATE custom_quote_drafts
SET finishing = ?,
    painting = ?,
    rush = ?,
    need_design = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: LinkDraftToQuoteRequest :exec
UPDATE custom_quote_drafts
SET quote_request_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: MarkDraftCompleted :exec
UPDATE custom_quote_drafts
SET completed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetDraftByQuoteRequestID :one
SELECT * FROM custom_quote_drafts
WHERE quote_request_id = ?;

-- name: MarkDraftAbandoned :exec
-- Clears the session_id so a new draft can be created for this session
-- The draft remains in the system for admin follow-up
UPDATE custom_quote_drafts
SET session_id = 'abandoned_' || session_id,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- File management

-- name: GetDraftFiles :many
SELECT * FROM custom_quote_draft_files
WHERE draft_id = ?
ORDER BY created_at;

-- name: AddDraftFile :one
INSERT INTO custom_quote_draft_files (draft_id, filename, file_path, file_size, file_type)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteDraftFile :exec
DELETE FROM custom_quote_draft_files
WHERE id = ?;

-- name: DeleteDraftFilesByDraftID :exec
DELETE FROM custom_quote_draft_files
WHERE draft_id = ?;

-- Recovery email queries

-- name: GetAbandonedDrafts :many
SELECT * FROM custom_quote_drafts
WHERE completed_at IS NULL
  AND recovery_email_sent_at IS NULL
  AND email IS NOT NULL
  AND email != ''
  AND updated_at < datetime('now', '-24 hours')
  AND updated_at > datetime('now', '-7 days')
ORDER BY updated_at DESC;

-- name: MarkDraftRecoveryEmailSent :exec
UPDATE custom_quote_drafts
SET recovery_email_sent_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- Analytics queries

-- name: GetDraftStats :one
SELECT
    COUNT(*) as total_drafts,
    COUNT(CASE WHEN completed_at IS NOT NULL THEN 1 END) as completed_count,
    COUNT(CASE WHEN completed_at IS NULL THEN 1 END) as abandoned_count,
    COUNT(CASE WHEN email IS NOT NULL AND email != '' THEN 1 END) as with_email_count
FROM custom_quote_drafts
WHERE created_at >= datetime('now', sqlc.arg(period_offset));

-- name: GetDraftsByStep :many
SELECT
    current_step,
    COUNT(*) as count
FROM custom_quote_drafts
WHERE completed_at IS NULL
  AND created_at >= datetime('now', sqlc.arg(period_offset))
GROUP BY current_step
ORDER BY current_step;

-- Cleanup

-- name: DeleteOldCompletedDrafts :exec
DELETE FROM custom_quote_drafts
WHERE completed_at IS NOT NULL
  AND completed_at < datetime('now', '-90 days');

-- name: DeleteOldAbandonedDrafts :exec
DELETE FROM custom_quote_drafts
WHERE completed_at IS NULL
  AND updated_at < datetime('now', '-30 days');

-- Admin listing queries (exclude archived by default)

-- name: ListAllDrafts :many
SELECT * FROM custom_quote_drafts
WHERE archived_at IS NULL
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: ListCompletedDrafts :many
SELECT * FROM custom_quote_drafts
WHERE completed_at IS NOT NULL
  AND archived_at IS NULL
ORDER BY completed_at DESC
LIMIT ? OFFSET ?;

-- name: ListAbandonedDrafts :many
SELECT * FROM custom_quote_drafts
WHERE completed_at IS NULL
  AND email IS NOT NULL
  AND email != ''
  AND updated_at < datetime('now', '-24 hours')
  AND archived_at IS NULL
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: ListInProgressDrafts :many
SELECT * FROM custom_quote_drafts
WHERE completed_at IS NULL
  AND updated_at >= datetime('now', '-24 hours')
  AND archived_at IS NULL
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: ListArchivedDrafts :many
SELECT * FROM custom_quote_drafts
WHERE archived_at IS NOT NULL
ORDER BY archived_at DESC
LIMIT ? OFFSET ?;

-- name: SearchDrafts :many
SELECT * FROM custom_quote_drafts
WHERE (name LIKE ? OR email LIKE ? OR description LIKE ?)
  AND archived_at IS NULL
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: SearchDraftsIncludeArchived :many
SELECT * FROM custom_quote_drafts
WHERE (name LIKE ? OR email LIKE ? OR description LIKE ?)
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: CountDraftsByStatus :one
SELECT
    COUNT(CASE WHEN archived_at IS NULL THEN 1 END) as total,
    COUNT(CASE WHEN completed_at IS NOT NULL AND archived_at IS NULL THEN 1 END) as completed,
    COUNT(CASE WHEN completed_at IS NULL AND email IS NOT NULL AND email != ''
               AND updated_at < datetime('now', '-24 hours') AND archived_at IS NULL THEN 1 END) as abandoned,
    COUNT(CASE WHEN completed_at IS NULL
               AND updated_at >= datetime('now', '-24 hours') AND archived_at IS NULL THEN 1 END) as in_progress,
    COUNT(CASE WHEN email IS NOT NULL AND email != '' AND archived_at IS NULL THEN 1 END) as with_email,
    COUNT(CASE WHEN archived_at IS NOT NULL THEN 1 END) as archived
FROM custom_quote_drafts;

-- Archive management

-- name: ArchiveDraft :exec
UPDATE custom_quote_drafts
SET archived_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UnarchiveDraft :exec
UPDATE custom_quote_drafts
SET archived_at = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
