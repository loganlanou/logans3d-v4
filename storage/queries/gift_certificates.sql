-- name: CreateGiftCertificate :one
INSERT INTO gift_certificates (id, amount, reference, created_by_user_id)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetGiftCertificate :one
SELECT * FROM gift_certificates WHERE id = ?;

-- name: GetGiftCertificateWithCreator :one
SELECT
    gc.*,
    u1.full_name as created_by_name,
    u2.full_name as redeemed_by_admin_name,
    u3.full_name as voided_by_admin_name
FROM gift_certificates gc
LEFT JOIN users u1 ON gc.created_by_user_id = u1.id
LEFT JOIN users u2 ON gc.redeemed_by_user_id = u2.id
LEFT JOIN users u3 ON gc.voided_by_user_id = u3.id
WHERE gc.id = ?;

-- name: ListGiftCertificates :many
SELECT
    gc.*,
    u.full_name as created_by_name
FROM gift_certificates gc
LEFT JOIN users u ON gc.created_by_user_id = u.id
ORDER BY gc.issued_at DESC
LIMIT ? OFFSET ?;

-- name: ListAllGiftCertificates :many
SELECT
    gc.*,
    u1.full_name as created_by_name,
    u2.full_name as redeemed_by_admin_name,
    u3.full_name as voided_by_admin_name
FROM gift_certificates gc
LEFT JOIN users u1 ON gc.created_by_user_id = u1.id
LEFT JOIN users u2 ON gc.redeemed_by_user_id = u2.id
LEFT JOIN users u3 ON gc.voided_by_user_id = u3.id
ORDER BY gc.issued_at DESC;

-- name: ListActiveGiftCertificates :many
SELECT
    gc.*,
    u.full_name as created_by_name
FROM gift_certificates gc
LEFT JOIN users u ON gc.created_by_user_id = u.id
WHERE gc.redeemed_at IS NULL AND gc.voided_at IS NULL
ORDER BY gc.issued_at DESC;

-- name: ListRedeemedGiftCertificates :many
SELECT
    gc.*,
    u1.full_name as created_by_name,
    u2.full_name as redeemed_by_admin_name
FROM gift_certificates gc
LEFT JOIN users u1 ON gc.created_by_user_id = u1.id
LEFT JOIN users u2 ON gc.redeemed_by_user_id = u2.id
WHERE gc.redeemed_at IS NOT NULL
ORDER BY gc.redeemed_at DESC;

-- name: SearchGiftCertificates :many
SELECT
    gc.*,
    u.full_name as created_by_name
FROM gift_certificates gc
LEFT JOIN users u ON gc.created_by_user_id = u.id
WHERE gc.id LIKE ?
   OR gc.reference LIKE ?
   OR gc.redeemer_name LIKE ?
ORDER BY gc.issued_at DESC
LIMIT ?;

-- name: RedeemGiftCertificate :exec
UPDATE gift_certificates
SET redeemed_at = CURRENT_TIMESTAMP,
    redeemed_by_user_id = ?,
    redeemer_name = ?,
    redemption_notes = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND redeemed_at IS NULL AND voided_at IS NULL;

-- name: VoidGiftCertificate :exec
UPDATE gift_certificates
SET voided_at = CURRENT_TIMESTAMP,
    voided_by_user_id = ?,
    void_reason = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND redeemed_at IS NULL AND voided_at IS NULL;

-- name: UpdateGiftCertificateImages :exec
UPDATE gift_certificates
SET image_png_path = ?,
    image_pdf_path = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteGiftCertificate :exec
DELETE FROM gift_certificates WHERE id = ?;

-- name: CountGiftCertificates :one
SELECT COUNT(*) FROM gift_certificates;

-- name: CountActiveGiftCertificates :one
SELECT COUNT(*) FROM gift_certificates WHERE redeemed_at IS NULL AND voided_at IS NULL;

-- name: CountVoidedGiftCertificates :one
SELECT COUNT(*) FROM gift_certificates WHERE voided_at IS NOT NULL;

-- name: SumVoidedGiftCertificates :one
SELECT COALESCE(SUM(amount), 0) FROM gift_certificates WHERE voided_at IS NOT NULL;

-- name: CountRedeemedGiftCertificates :one
SELECT COUNT(*) FROM gift_certificates WHERE redeemed_at IS NOT NULL;

-- name: SumActiveGiftCertificates :one
SELECT COALESCE(SUM(amount), 0) FROM gift_certificates WHERE redeemed_at IS NULL AND voided_at IS NULL;

-- name: SumRedeemedGiftCertificates :one
SELECT COALESCE(SUM(amount), 0) FROM gift_certificates WHERE redeemed_at IS NOT NULL;
