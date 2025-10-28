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
SELECT * FROM order_items WHERE order_id = ?;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    id, order_id, product_id, product_variant_id, quantity, unit_price_cents, 
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