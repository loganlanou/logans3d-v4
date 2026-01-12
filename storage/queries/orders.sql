-- name: GetOrder :one
SELECT * FROM orders WHERE id = ?;

-- name: GetOrderWithItems :one
SELECT 
    o.*,
    GROUP_CONCAT(
        oi.id || ',' || oi.product_id || ',' || oi.quantity || ',' || 
        oi.unit_price_cents || ',' || oi.total_price_cents || ',' || 
        oi.product_name || ',' || COALESCE(oi.product_sku, '')
    ) as order_items
FROM orders o
LEFT JOIN order_items oi ON o.id = oi.order_id
WHERE o.id = ?
GROUP BY o.id;

-- name: ListOrders :many
SELECT * FROM orders 
ORDER BY created_at DESC;

-- name: ListOrdersByStatus :many
SELECT * FROM orders 
WHERE status = ? 
ORDER BY created_at DESC;

-- name: ListOrdersByUser :many
SELECT * FROM orders
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: GetOrderByStripeSessionID :one
SELECT * FROM orders
WHERE stripe_checkout_session_id = ?
LIMIT 1;

-- name: CreateOrder :one
INSERT INTO orders (
    id, user_id, customer_email, customer_name, customer_phone,
    shipping_address_line1, shipping_address_line2, shipping_city, shipping_state,
    shipping_postal_code, shipping_country,
    subtotal_cents, tax_cents, shipping_cents, total_cents,
    original_subtotal_cents, discount_cents, promotion_code, promotion_code_id,
    stripe_payment_intent_id, stripe_customer_id, stripe_checkout_session_id,
    easypost_shipment_id, status, notes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateOrderStatus :one
UPDATE orders 
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateOrderTracking :one
UPDATE orders
SET tracking_number = ?, tracking_url = ?, carrier = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateOrderLabel :one
UPDATE orders
SET easypost_label_url = ?, tracking_number = ?, carrier = ?, status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateOrderNotes :one
UPDATE orders 
SET notes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteOrder :exec
DELETE FROM orders WHERE id = ?;

-- name: GetOrderItems :many
SELECT
    oi.*,
    COALESCE(c.name, '') as category_name
FROM order_items oi
LEFT JOIN products p ON oi.product_id = p.id
LEFT JOIN categories c ON p.category_id = c.id
WHERE oi.order_id = ?;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    id, order_id, product_id, product_sku_id, quantity, unit_price_cents, 
    total_price_cents, product_name, product_sku
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetOrderStats :one
SELECT
    COUNT(*) as total_orders,
    COUNT(CASE WHEN status = 'received' THEN 1 END) as received_orders,
    COUNT(CASE WHEN status = 'in_production' THEN 1 END) as in_production_orders,
    COUNT(CASE WHEN status = 'shipped' THEN 1 END) as shipped_orders,
    COUNT(CASE WHEN status = 'delivered' THEN 1 END) as delivered_orders,
    COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_orders,
    SUM(total_cents) as total_revenue_cents,
    AVG(total_cents) as average_order_value_cents
FROM orders;

-- name: GetBuyAgainItems :many
-- Returns unique products from a user's past orders for "Buy It Again" feature
SELECT
    recent.product_id,
    recent.product_sku_id,
    recent.product_name,
    recent.product_sku,
    p.slug as product_slug,
    p.price_cents,
    p.is_active,
    p.has_variants,
    COALESCE(
        CASE WHEN psi.image_url IS NOT NULL THEN 'styles/' || psi.image_url END,
        pi.image_url,
        ''
    ) as image_url,
    recent.last_purchased_at
FROM (
    SELECT
        oi.product_id,
        oi.product_sku_id,
        oi.product_name,
        oi.product_sku,
        MAX(o.created_at) as last_purchased_at
    FROM order_items oi
    JOIN orders o ON oi.order_id = o.id
    WHERE o.user_id = ?
      AND o.status NOT IN ('cancelled', 'refunded')
    GROUP BY oi.product_id, oi.product_sku_id, oi.product_name, oi.product_sku
) recent
LEFT JOIN products p ON recent.product_id = p.id
LEFT JOIN product_skus ps ON recent.product_sku_id = ps.id
LEFT JOIN product_styles pst ON ps.product_style_id = pst.id
LEFT JOIN product_style_images psi ON psi.product_style_id = pst.id AND psi.is_primary = TRUE
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
WHERE p.is_active = TRUE
ORDER BY recent.last_purchased_at DESC
LIMIT 12;
