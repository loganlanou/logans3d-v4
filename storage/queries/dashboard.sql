-- Dashboard Metrics Queries

-- name: GetDashboardRevenueToday :one
SELECT
    COALESCE(SUM(total_cents), 0) as revenue_cents,
    COUNT(*) as order_count
FROM orders
WHERE DATE(substr(created_at, 1, 10)) = DATE('now')
    AND status NOT IN ('cancelled', 'refunded');

-- name: GetDashboardRevenueWeek :one
SELECT
    COALESCE(SUM(total_cents), 0) as revenue_cents,
    COUNT(*) as order_count
FROM orders
WHERE created_at >= datetime('now', '-7 days')
    AND status NOT IN ('cancelled', 'refunded');

-- name: GetDashboardRevenueMonth :one
SELECT
    COALESCE(SUM(total_cents), 0) as revenue_cents,
    COUNT(*) as order_count
FROM orders
WHERE created_at >= datetime('now', 'start of month')
    AND status NOT IN ('cancelled', 'refunded');

-- name: GetDashboardRevenuePreviousMonth :one
SELECT
    COALESCE(SUM(total_cents), 0) as revenue_cents,
    COUNT(*) as order_count
FROM orders
WHERE created_at >= datetime('now', 'start of month', '-1 month')
    AND created_at < datetime('now', 'start of month')
    AND status NOT IN ('cancelled', 'refunded');

-- name: GetDashboardOrdersByStatus :one
SELECT
    COUNT(*) as total_orders,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_orders,
    COUNT(CASE WHEN status = 'received' THEN 1 END) as received_orders,
    COUNT(CASE WHEN status = 'in_production' THEN 1 END) as in_production_orders,
    COUNT(CASE WHEN status = 'ready_to_ship' THEN 1 END) as ready_to_ship_orders,
    COUNT(CASE WHEN status = 'shipped' THEN 1 END) as shipped_orders,
    COUNT(CASE WHEN status = 'delivered' THEN 1 END) as delivered_orders,
    COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_orders,
    COUNT(CASE WHEN status = 'refunded' THEN 1 END) as refunded_orders
FROM orders;

-- name: GetDashboardAverageOrderValue :one
SELECT
    COALESCE(AVG(total_cents), 0) as avg_order_value_cents
FROM orders
WHERE status NOT IN ('cancelled', 'refunded')
    AND created_at >= datetime('now', '-30 days');

-- name: GetDashboardProductStats :one
SELECT
    COUNT(*) as total_products,
    COUNT(CASE WHEN is_active = TRUE THEN 1 END) as active_products,
    COUNT(CASE WHEN is_featured = TRUE THEN 1 END) as featured_products,
    COUNT(CASE WHEN is_premium = TRUE THEN 1 END) as premium_products,
    COUNT(CASE WHEN stock_quantity IS NOT NULL AND stock_quantity <= 5 AND stock_quantity > 0 THEN 1 END) as low_stock_products,
    COUNT(CASE WHEN stock_quantity IS NOT NULL AND stock_quantity = 0 THEN 1 END) as out_of_stock_products
FROM products;

-- name: GetDashboardLowStockProducts :many
SELECT id, name, sku, stock_quantity, price_cents
FROM products
WHERE stock_quantity IS NOT NULL
    AND stock_quantity <= 5
    AND stock_quantity > 0
    AND is_active = TRUE
ORDER BY stock_quantity ASC
LIMIT 10;

-- name: GetDashboardCustomerStats :one
SELECT
    COUNT(DISTINCT id) as total_customers,
    COUNT(DISTINCT CASE WHEN created_at >= datetime('now', '-7 days') THEN id END) as new_customers_week,
    COUNT(DISTINCT CASE WHEN created_at >= datetime('now', '-30 days') THEN id END) as new_customers_month
FROM users;

-- name: GetDashboardCustomerOrderStats :one
SELECT
    COUNT(DISTINCT user_id) as customers_with_orders,
    COUNT(DISTINCT CASE
        WHEN user_id IN (
            SELECT user_id FROM orders GROUP BY user_id HAVING COUNT(*) > 1
        ) THEN user_id
    END) as returning_customers
FROM orders
WHERE user_id IS NOT NULL AND user_id != '';

-- name: GetDashboardCartStats :one
WITH cart_summary AS (
    SELECT
        COALESCE(ci.session_id, '') as session_id,
        COALESCE(ci.user_id, '') as user_id,
        MAX(ci.updated_at) as last_activity,
        COUNT(DISTINCT ci.id) as item_count,
        SUM(p.price_cents * ci.quantity) as cart_value_cents
    FROM cart_items ci
    JOIN products p ON ci.product_id = p.id
    GROUP BY COALESCE(ci.session_id, ''), COALESCE(ci.user_id, '')
    HAVING item_count > 0
)
SELECT
    COUNT(*) as total_carts,
    COUNT(CASE WHEN last_activity >= datetime('now', '-15 minutes') THEN 1 END) as active_carts,
    COUNT(CASE WHEN last_activity < datetime('now', '-30 minutes') THEN 1 END) as abandoned_carts,
    COALESCE(SUM(cart_value_cents), 0) as total_cart_value_cents,
    COALESCE(SUM(CASE WHEN last_activity >= datetime('now', '-15 minutes') THEN cart_value_cents END), 0) as active_cart_value_cents
FROM cart_summary;

-- name: GetDashboardAbandonedCartStats :one
SELECT
    COUNT(*) as total_abandoned,
    COUNT(CASE WHEN abandoned_at >= datetime('now', '-24 hours') THEN 1 END) as abandoned_24h,
    COUNT(CASE WHEN abandoned_at >= datetime('now', '-7 days') THEN 1 END) as abandoned_7d,
    COUNT(CASE WHEN abandoned_at >= datetime('now', '-30 days') THEN 1 END) as abandoned_30d,
    COUNT(CASE WHEN status = 'recovered' THEN 1 END) as recovered_carts,
    COALESCE(SUM(cart_value_cents), 0) as total_value_cents,
    COALESCE(SUM(CASE WHEN status = 'recovered' THEN cart_value_cents END), 0) as recovered_value_cents
FROM abandoned_carts
WHERE abandoned_at >= datetime('now', '-30 days');

-- name: GetDashboardQuoteStats :one
SELECT
    COUNT(*) as total_quotes,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_quotes,
    COUNT(CASE WHEN status = 'responded' THEN 1 END) as responded_quotes,
    COUNT(CASE WHEN status = 'converted' THEN 1 END) as converted_quotes,
    COUNT(CASE WHEN created_at >= datetime('now', '-7 days') THEN 1 END) as quotes_week,
    COUNT(CASE WHEN created_at >= datetime('now', '-30 days') THEN 1 END) as quotes_month
FROM quote_requests;

-- name: GetDashboardContactStats :one
SELECT
    COUNT(*) as total_contacts,
    COUNT(CASE WHEN status = 'new' THEN 1 END) as new_contacts,
    COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress_contacts,
    COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved_contacts,
    COUNT(CASE WHEN priority = 'high' THEN 1 END) as high_priority_contacts,
    COUNT(CASE WHEN created_at >= datetime('now', '-7 days') THEN 1 END) as contacts_week
FROM contact_requests;

-- name: GetDashboardRecentOrders :many
SELECT
    o.id,
    o.customer_name,
    o.customer_email,
    o.total_cents,
    o.status,
    o.created_at,
    COUNT(oi.id) as item_count
FROM orders o
LEFT JOIN order_items oi ON o.id = oi.order_id
GROUP BY o.id
ORDER BY o.created_at DESC
LIMIT 10;

-- name: GetDashboardRevenueByDay :many
SELECT
    DATE(substr(created_at, 1, 10)) as date,
    COALESCE(SUM(total_cents), 0) as revenue_cents,
    COUNT(*) as order_count
FROM orders
WHERE created_at >= datetime('now', '-30 days')
    AND status NOT IN ('cancelled', 'refunded')
GROUP BY DATE(substr(created_at, 1, 10))
ORDER BY date ASC;

-- name: GetDashboardTopProducts :many
SELECT
    p.id,
    p.name,
    p.price_cents,
    COUNT(oi.id) as times_ordered,
    SUM(oi.quantity) as total_quantity_sold,
    SUM(oi.total_price_cents) as total_revenue_cents
FROM products p
JOIN order_items oi ON p.id = oi.product_id
JOIN orders o ON oi.order_id = o.id
WHERE o.status NOT IN ('cancelled', 'refunded')
    AND o.created_at >= datetime('now', '-30 days')
GROUP BY p.id
ORDER BY total_revenue_cents DESC
LIMIT 10;
