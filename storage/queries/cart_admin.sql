-- Admin Cart Management Queries

-- name: GetAllCartsWithDetails :many
WITH cart_summary AS (
    SELECT
        COALESCE(ci.session_id, '') as session_id,
        COALESCE(ci.user_id, '') as user_id,
        MIN(ci.created_at) as created_at,
        MAX(ci.updated_at) as last_activity,
        COUNT(DISTINCT ci.id) as item_count,
        SUM((p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) * ci.quantity) as cart_value_cents,
        ci.user_id as raw_user_id
    FROM cart_items ci
    JOIN products p ON ci.product_id = p.id
    LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
    GROUP BY COALESCE(ci.session_id, ''), COALESCE(ci.user_id, ''), ci.user_id
    HAVING item_count > 0
)
SELECT
    cs.session_id,
    cs.user_id,
    cs.created_at,
    cs.last_activity,
    cs.item_count,
    cs.cart_value_cents,
    u.email as customer_email,
    u.full_name as customer_name,
    u.profile_image_url as customer_avatar,
    CASE
        WHEN cs.last_activity < datetime('now', '-30 minutes') THEN 'abandoned'
        WHEN cs.last_activity < datetime('now', '-25 minutes') THEN 'at_risk'
        WHEN cs.last_activity < datetime('now', '-15 minutes') THEN 'idle'
        ELSE 'active'
    END as status
FROM cart_summary cs
LEFT JOIN users u ON cs.raw_user_id = u.id
WHERE 1=1
    AND (sqlc.narg(status) IS NULL OR
        (sqlc.narg(status) = 'abandoned' AND cs.last_activity < datetime('now', '-30 minutes')) OR
        (sqlc.narg(status) = 'at_risk' AND cs.last_activity >= datetime('now', '-30 minutes') AND cs.last_activity < datetime('now', '-25 minutes')) OR
        (sqlc.narg(status) = 'idle' AND cs.last_activity >= datetime('now', '-25 minutes') AND cs.last_activity < datetime('now', '-15 minutes')) OR
        (sqlc.narg(status) = 'active' AND cs.last_activity >= datetime('now', '-15 minutes')))
    AND (sqlc.narg(customer_type) IS NULL OR
        (sqlc.narg(customer_type) = 'guest' AND cs.raw_user_id IS NULL) OR
        (sqlc.narg(customer_type) = 'registered' AND cs.raw_user_id IS NOT NULL))
    AND (sqlc.narg(search) IS NULL OR
        u.email LIKE '%' || sqlc.narg(search) || '%' OR
        u.full_name LIKE '%' || sqlc.narg(search) || '%')
ORDER BY cs.last_activity DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(offset);

-- name: GetCartDetailsBySession :one
SELECT
    ci.session_id,
    MIN(ci.created_at) as created_at,
    MAX(ci.updated_at) as last_activity,
    COUNT(DISTINCT ci.id) as item_count,
    SUM((p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) * ci.quantity) as cart_value_cents
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
WHERE ci.session_id = sqlc.arg(session_id)
GROUP BY ci.session_id;

-- name: GetCartDetailsByUser :one
SELECT
    ci.user_id,
    u.email as customer_email,
    u.full_name as customer_name,
    u.profile_image_url as customer_avatar,
    MIN(ci.created_at) as created_at,
    MAX(ci.updated_at) as last_activity,
    COUNT(DISTINCT ci.id) as item_count,
    SUM((p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) * ci.quantity) as cart_value_cents
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
LEFT JOIN users u ON ci.user_id = u.id
WHERE ci.user_id = sqlc.arg(user_id)
GROUP BY ci.user_id, u.email, u.full_name, u.profile_image_url;

-- name: GetCartItemsWithDetails :many
SELECT
    ci.id,
    ci.product_id,
    ci.product_sku_id,
    ci.quantity,
    ci.created_at,
    ci.updated_at,
    p.name as product_name,
    (p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) as price_cents,
    COALESCE(psi.image_url, pi.image_url) as product_image,
    ps.sku as variant_sku,
    COALESCE(pst.name || ' - ' || sz.display_name, '') as variant_name,
    ((p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) * ci.quantity) as line_total_cents
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
LEFT JOIN product_styles pst ON ps.product_style_id = pst.id
LEFT JOIN sizes sz ON ps.size_id = sz.id
LEFT JOIN product_images pi ON ci.product_id = pi.product_id AND pi.is_primary = TRUE
LEFT JOIN product_style_images psi ON psi.product_style_id = pst.id AND psi.is_primary = TRUE
WHERE (ci.session_id = sqlc.narg(session_id) OR ci.user_id = sqlc.narg(user_id))
ORDER BY ci.created_at DESC;

-- name: GetCartMetrics :one
SELECT
    COUNT(DISTINCT cart_activity.cart_id) as total_carts,
    COUNT(DISTINCT CASE WHEN cart_activity.is_guest = 1 THEN cart_activity.cart_id END) as guest_carts,
    COUNT(DISTINCT CASE WHEN cart_activity.is_guest = 0 THEN cart_activity.cart_id END) as registered_carts,
    COALESCE(SUM(cart_totals.cart_value), 0) as total_value_cents,
    COALESCE(AVG(cart_totals.cart_value), 0) as avg_cart_value_cents,
    COUNT(DISTINCT CASE WHEN cart_activity.last_activity < datetime('now', '-30 minutes') THEN cart_activity.cart_id END) as abandoned_count,
    COUNT(DISTINCT CASE WHEN cart_activity.last_activity >= datetime('now', '-30 minutes') THEN cart_activity.cart_id END) as active_count
FROM (
    SELECT
        COALESCE(ci.session_id, ci.user_id) as cart_id,
        MAX(ci.updated_at) as last_activity,
        CASE WHEN ci.user_id IS NULL THEN 1 ELSE 0 END as is_guest
    FROM cart_items ci
    GROUP BY COALESCE(ci.session_id, ci.user_id), is_guest
) cart_activity
LEFT JOIN (
    SELECT
        COALESCE(ci2.session_id, ci2.user_id) as cart_id,
        SUM((p2.price_cents + COALESCE(ps2.price_adjustment_cents, 0)) * ci2.quantity) as cart_value
    FROM cart_items ci2
    JOIN products p2 ON ci2.product_id = p2.id
    LEFT JOIN product_skus ps2 ON ci2.product_sku_id = ps2.id
    GROUP BY COALESCE(ci2.session_id, ci2.user_id)
) cart_totals ON cart_activity.cart_id = cart_totals.cart_id;

-- name: GetRecentCartActivity :many
SELECT
    ci.id,
    ci.session_id,
    ci.user_id,
    ci.product_id,
    ci.quantity,
    ci.created_at,
    ci.updated_at,
    p.name as product_name,
    u.email as customer_email,
    u.full_name as customer_name
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN users u ON ci.user_id = u.id
WHERE ci.created_at >= datetime('now', sqlc.arg(time_period))
ORDER BY ci.created_at DESC
LIMIT sqlc.arg(limit_count);

-- name: SearchCarts :many
SELECT
    COALESCE(ci.session_id, '') as session_id,
    COALESCE(ci.user_id, '') as user_id,
    MIN(ci.created_at) as created_at,
    MAX(ci.updated_at) as last_activity,
    COUNT(DISTINCT ci.id) as item_count,
    SUM((p.price_cents + COALESCE(ps.price_adjustment_cents, 0)) * ci.quantity) as cart_value_cents,
    u.email as customer_email,
    u.full_name as customer_name
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
LEFT JOIN users u ON ci.user_id = u.id
WHERE (
    u.email LIKE '%' || sqlc.arg(search_query) || '%' OR
    u.full_name LIKE '%' || sqlc.arg(search_query) || '%' OR
    ci.session_id LIKE '%' || sqlc.arg(search_query) || '%'
)
GROUP BY COALESCE(ci.session_id, ''), COALESCE(ci.user_id, ''), u.email, u.full_name
HAVING item_count > 0
ORDER BY last_activity DESC
LIMIT 50;
