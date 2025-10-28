-- Promotion Campaigns

-- name: CreatePromotionCampaign :one
INSERT INTO promotion_campaigns (
    id, name, description, discount_type, discount_value,
    stripe_promotion_id, start_date, end_date, max_uses, active
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPromotionCampaignByID :one
SELECT * FROM promotion_campaigns
WHERE id = ?;

-- name: GetAllPromotionCampaigns :many
SELECT * FROM promotion_campaigns
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: GetActivePromotionCampaigns :many
SELECT * FROM promotion_campaigns
WHERE active = 1
ORDER BY created_at DESC;

-- name: UpdatePromotionCampaign :exec
UPDATE promotion_campaigns
SET name = ?,
    description = ?,
    discount_type = ?,
    discount_value = ?,
    end_date = ?,
    max_uses = ?,
    active = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeletePromotionCampaign :exec
DELETE FROM promotion_campaigns
WHERE id = ?;

-- Promotion Codes

-- name: CreatePromotionCode :one
INSERT INTO promotion_codes (
    id, campaign_id, code, stripe_promotion_code_id,
    email, user_id, max_uses, expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPromotionCodeByID :one
SELECT * FROM promotion_codes
WHERE id = ?;

-- name: GetPromotionCodeByCode :one
SELECT * FROM promotion_codes
WHERE code = ?;

-- name: GetPromotionCodesByCampaign :many
SELECT * FROM promotion_codes
WHERE campaign_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: GetPromotionCodesByEmail :many
SELECT * FROM promotion_codes
WHERE email = ?
ORDER BY created_at DESC;

-- name: MarkPromotionCodeUsed :exec
UPDATE promotion_codes
SET current_uses = current_uses + 1,
    first_used_at = COALESCE(first_used_at, CURRENT_TIMESTAMP),
    last_used_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetPromotionCodeStats :one
SELECT
    COUNT(*) as total_codes,
    SUM(CASE WHEN current_uses > 0 THEN 1 ELSE 0 END) as used_codes,
    SUM(current_uses) as total_uses
FROM promotion_codes
WHERE campaign_id = ?;

-- Marketing Contacts

-- name: CreateMarketingContact :one
INSERT INTO marketing_contacts (
    id, email, first_name, last_name, source, opted_in, promotion_code_id
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMarketingContactByEmail :one
SELECT * FROM marketing_contacts
WHERE email = ?;

-- name: GetAllMarketingContacts :many
SELECT * FROM marketing_contacts
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: GetOptedInContacts :many
SELECT * FROM marketing_contacts
WHERE opted_in = 1
ORDER BY created_at DESC;

-- name: UpdateMarketingContactOptIn :exec
UPDATE marketing_contacts
SET opted_in = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CountMarketingContacts :one
SELECT COUNT(*) FROM marketing_contacts;

-- name: CountOptedInContacts :one
SELECT COUNT(*) FROM marketing_contacts
WHERE opted_in = 1;
