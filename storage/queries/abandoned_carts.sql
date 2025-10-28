-- Abandoned Cart Management Queries

-- name: CreateAbandonedCart :one
INSERT INTO abandoned_carts (
    id, session_id, user_id, customer_email, customer_name,
    cart_value_cents, item_count, abandoned_at, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: CreateCartSnapshot :exec
INSERT INTO cart_snapshots (
    id, abandoned_cart_id, product_id, product_name, product_sku,
    product_image_url, quantity, unit_price_cents, total_price_cents
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAbandonedCartByID :one
SELECT * FROM abandoned_carts
WHERE id = ?;

-- name: GetAbandonedCartBySession :one
SELECT * FROM abandoned_carts
WHERE session_id = ? AND status = 'active'
ORDER BY abandoned_at DESC
LIMIT 1;

-- name: GetAbandonedCartByUser :one
SELECT * FROM abandoned_carts
WHERE user_id = ? AND status = 'active'
ORDER BY abandoned_at DESC
LIMIT 1;

-- name: GetAbandonedCartsByStatus :many
SELECT * FROM abandoned_carts
WHERE status = ?
ORDER BY abandoned_at DESC;

-- name: ListRecentAbandonedCarts :many
SELECT
    ac.*,
    COUNT(cs.id) as snapshot_item_count,
    COUNT(cra.id) as recovery_attempt_count
FROM abandoned_carts ac
LEFT JOIN cart_snapshots cs ON ac.id = cs.abandoned_cart_id
LEFT JOIN cart_recovery_attempts cra ON ac.id = cra.abandoned_cart_id
WHERE ac.abandoned_at >= datetime('now', '-24 hours')
GROUP BY ac.id
ORDER BY ac.abandoned_at DESC;

-- name: ListAbandonedCartsWithFilters :many
SELECT
    ac.*,
    COUNT(cs.id) as snapshot_item_count,
    COUNT(cra.id) as recovery_attempt_count
FROM abandoned_carts ac
LEFT JOIN cart_snapshots cs ON ac.id = cs.abandoned_cart_id
LEFT JOIN cart_recovery_attempts cra ON ac.id = cra.abandoned_cart_id
WHERE
    (sqlc.narg(status) IS NULL OR ac.status = sqlc.narg(status))
    AND (sqlc.narg(min_value_cents) IS NULL OR ac.cart_value_cents >= sqlc.narg(min_value_cents))
    AND (sqlc.narg(from_date) IS NULL OR ac.abandoned_at >= sqlc.narg(from_date))
    AND (sqlc.narg(to_date) IS NULL OR ac.abandoned_at <= sqlc.narg(to_date))
GROUP BY ac.id
ORDER BY ac.abandoned_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(offset);

-- name: GetCartSnapshotsByAbandonedCartID :many
SELECT * FROM cart_snapshots
WHERE abandoned_cart_id = ?
ORDER BY created_at;

-- name: UpdateAbandonedCartStatus :exec
UPDATE abandoned_carts
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: MarkCartAsRecovered :exec
UPDATE abandoned_carts
SET
    status = 'recovered',
    recovered_at = CURRENT_TIMESTAMP,
    recovery_method = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateAbandonedCartNotes :exec
UPDATE abandoned_carts
SET notes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: MarkCartAsContacted :exec
UPDATE abandoned_carts
SET
    status = 'contacted',
    last_contacted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- Recovery Attempts

-- name: CreateRecoveryAttempt :one
INSERT INTO cart_recovery_attempts (
    id, abandoned_cart_id, attempt_type, sent_at,
    email_subject, tracking_token, status
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetRecoveryAttemptsByCartID :many
SELECT * FROM cart_recovery_attempts
WHERE abandoned_cart_id = ?
ORDER BY sent_at DESC;

-- name: GetRecoveryAttemptByToken :one
SELECT * FROM cart_recovery_attempts
WHERE tracking_token = ?;

-- name: MarkRecoveryAttemptOpened :exec
UPDATE cart_recovery_attempts
SET
    opened_at = CURRENT_TIMESTAMP,
    status = 'opened'
WHERE tracking_token = ? AND opened_at IS NULL;

-- name: MarkRecoveryAttemptClicked :exec
UPDATE cart_recovery_attempts
SET
    clicked_at = CURRENT_TIMESTAMP,
    status = 'clicked'
WHERE tracking_token = ? AND clicked_at IS NULL;

-- name: GetCartsNeedingRecoveryEmail :many
SELECT ac.*
FROM abandoned_carts ac
LEFT JOIN cart_recovery_attempts cra ON ac.id = cra.abandoned_cart_id AND cra.attempt_type = sqlc.arg(attempt_type)
WHERE
    ac.status IN ('active', 'contacted')
    AND ac.customer_email IS NOT NULL
    AND ac.abandoned_at <= datetime('now', sqlc.arg(time_offset))
    AND ac.abandoned_at >= datetime('now', sqlc.arg(min_time_offset))
    AND cra.id IS NULL
ORDER BY ac.abandoned_at ASC;

-- Analytics Queries

-- name: GetAbandonmentRateByPeriod :one
SELECT
    COUNT(DISTINCT CASE WHEN ci.id IS NOT NULL THEN ci.session_id END) as total_carts_created,
    COUNT(DISTINCT CASE WHEN ac.id IS NOT NULL THEN ac.session_id END) as abandoned_carts_count,
    COALESCE(
        CAST(COUNT(DISTINCT CASE WHEN ac.id IS NOT NULL THEN ac.session_id END) AS FLOAT) /
        NULLIF(COUNT(DISTINCT CASE WHEN ci.id IS NOT NULL THEN ci.session_id END), 0) * 100,
        0
    ) as abandonment_rate_percent
FROM cart_items ci
LEFT JOIN abandoned_carts ac ON ci.session_id = ac.session_id
WHERE ci.created_at >= datetime('now', sqlc.arg(period_offset));

-- name: GetTotalAbandonedCartValue :one
SELECT
    COUNT(*) as total_count,
    SUM(cart_value_cents) as total_value_cents,
    AVG(cart_value_cents) as avg_value_cents,
    MAX(cart_value_cents) as max_value_cents
FROM abandoned_carts
WHERE
    status = 'active'
    AND abandoned_at >= datetime('now', sqlc.arg(period_offset));

-- name: GetRecoveryRateByPeriod :one
SELECT
    COUNT(*) as total_abandoned,
    COUNT(CASE WHEN status = 'recovered' THEN 1 END) as recovered_count,
    COALESCE(
        CAST(COUNT(CASE WHEN status = 'recovered' THEN 1 END) AS FLOAT) /
        NULLIF(COUNT(*), 0) * 100,
        0
    ) as recovery_rate_percent,
    SUM(CASE WHEN status = 'recovered' THEN cart_value_cents ELSE 0 END) as recovered_value_cents
FROM abandoned_carts
WHERE abandoned_at >= datetime('now', sqlc.arg(period_offset));

-- name: GetRecoveryEmailPerformance :many
SELECT
    attempt_type,
    COUNT(*) as total_sent,
    COUNT(CASE WHEN opened_at IS NOT NULL THEN 1 END) as opened_count,
    COUNT(CASE WHEN clicked_at IS NOT NULL THEN 1 END) as clicked_count,
    COALESCE(
        CAST(COUNT(CASE WHEN opened_at IS NOT NULL THEN 1 END) AS FLOAT) /
        NULLIF(COUNT(*), 0) * 100,
        0
    ) as open_rate_percent,
    COALESCE(
        CAST(COUNT(CASE WHEN clicked_at IS NOT NULL THEN 1 END) AS FLOAT) /
        NULLIF(COUNT(*), 0) * 100,
        0
    ) as click_rate_percent
FROM cart_recovery_attempts
WHERE sent_at >= datetime('now', sqlc.arg(period_offset))
GROUP BY attempt_type
ORDER BY attempt_type;

-- name: GetTopAbandonedProducts :many
SELECT
    cs.product_id,
    cs.product_name,
    COUNT(*) as abandoned_count,
    SUM(cs.total_price_cents) as total_value_cents,
    AVG(cs.total_price_cents) as avg_value_cents
FROM cart_snapshots cs
JOIN abandoned_carts ac ON cs.abandoned_cart_id = ac.id
WHERE ac.abandoned_at >= datetime('now', sqlc.arg(period_offset))
GROUP BY cs.product_id, cs.product_name
ORDER BY abandoned_count DESC
LIMIT sqlc.arg(limit_count);

-- name: GetAbandonmentTrendByDay :many
SELECT
    DATE(substr(abandoned_at, 1, 10)) as date,
    COUNT(*) as cart_count,
    SUM(cart_value_cents) as total_value_cents,
    AVG(cart_value_cents) as avg_value_cents
FROM abandoned_carts
WHERE abandoned_at >= datetime('now', sqlc.arg(period_offset))
GROUP BY DATE(substr(abandoned_at, 1, 10))
ORDER BY date DESC;

-- name: GetAbandonmentByHourOfDay :many
SELECT
    CAST(strftime('%H', abandoned_at) AS INTEGER) as hour_of_day,
    COUNT(*) as cart_count,
    AVG(cart_value_cents) as avg_value_cents
FROM abandoned_carts
WHERE abandoned_at >= datetime('now', sqlc.arg(period_offset))
GROUP BY hour_of_day
ORDER BY hour_of_day;

-- name: GetHighValueAbandonedCarts :many
SELECT * FROM abandoned_carts
WHERE
    status = 'active'
    AND cart_value_cents >= sqlc.arg(min_value_cents)
    AND abandoned_at >= datetime('now', '-7 days')
ORDER BY cart_value_cents DESC
LIMIT sqlc.arg(limit_count);

-- name: SearchAbandonedCarts :many
SELECT * FROM abandoned_carts
WHERE
    (customer_name LIKE '%' || sqlc.arg(search_query) || '%'
     OR customer_email LIKE '%' || sqlc.arg(search_query) || '%')
    AND abandoned_at >= datetime('now', '-30 days')
ORDER BY abandoned_at DESC
LIMIT 50;

-- Cleanup queries

-- name: MarkExpiredAbandonedCarts :exec
UPDATE abandoned_carts
SET status = 'expired', updated_at = CURRENT_TIMESTAMP
WHERE
    status IN ('active', 'contacted')
    AND abandoned_at < datetime('now', '-30 days');

-- name: DeleteOldAbandonedCarts :exec
DELETE FROM abandoned_carts
WHERE
    status = 'expired'
    AND abandoned_at < datetime('now', '-90 days');

-- Active Carts queries (carts not yet abandoned)

-- name: GetActiveCarts :many
SELECT
    COALESCE(ci.session_id, '') as session_id,
    COALESCE(ci.user_id, '') as user_id,
    MAX(ci.updated_at) as last_activity,
    COUNT(DISTINCT ci.id) as item_count,
    SUM(p.price_cents * ci.quantity) as cart_value_cents,
    u.email as customer_email,
    u.full_name as customer_name
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN users u ON ci.user_id = u.id
LEFT JOIN abandoned_carts ac ON (ci.session_id = ac.session_id OR ci.user_id = ac.user_id) AND ac.status = 'active'
WHERE ac.id IS NULL
GROUP BY COALESCE(ci.session_id, ''), COALESCE(ci.user_id, ''), u.email, u.full_name
HAVING item_count > 0
ORDER BY last_activity DESC
LIMIT 100;

-- name: GetActiveCartsMetrics :one
SELECT
    COUNT(DISTINCT COALESCE(ci.session_id, ci.user_id)) as total_active_carts,
    COALESCE(SUM(p.price_cents * ci.quantity), 0) as total_value_cents,
    COALESCE(AVG(p.price_cents * ci.quantity), 0) as avg_cart_value_cents
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN abandoned_carts ac ON (ci.session_id = ac.session_id OR ci.user_id = ac.user_id) AND ac.status = 'active'
WHERE ac.id IS NULL;

-- Promotion Code Support for Abandoned Carts

-- name: HasUserMadePurchase :one
SELECT COUNT(*) > 0 as has_purchased
FROM orders
WHERE (user_id = sqlc.narg(user_id) OR customer_email = sqlc.narg(customer_email))
  AND status NOT IN ('cancelled', 'failed');

-- name: UpdateAbandonedCartPromoCode :exec
UPDATE abandoned_carts
SET promotion_code_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetAbandonedCartWithPromoCode :one
SELECT
    ac.*,
    pc.code as promo_code,
    pc.expires_at as promo_expires_at,
    pcamp.discount_value as promo_discount_value,
    pcamp.discount_type as promo_discount_type
FROM abandoned_carts ac
LEFT JOIN promotion_codes pc ON ac.promotion_code_id = pc.id
LEFT JOIN promotion_campaigns pcamp ON pc.campaign_id = pcamp.id
WHERE ac.id = ?;
