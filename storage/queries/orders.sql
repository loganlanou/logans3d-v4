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

-- name: CreateOrder :one
INSERT INTO orders (
    id, user_id, customer_email, customer_name, customer_phone,
    billing_address_line1, billing_address_line2, billing_city, billing_state, 
    billing_postal_code, billing_country,
    shipping_address_line1, shipping_address_line2, shipping_city, shipping_state, 
    shipping_postal_code, shipping_country,
    subtotal_cents, tax_cents, shipping_cents, total_cents,
    stripe_payment_intent_id, stripe_customer_id, status, fulfillment_status, payment_status, notes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateOrderStatus :one
UPDATE orders 
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateOrderFulfillmentStatus :one
UPDATE orders 
SET fulfillment_status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateOrderPaymentStatus :one
UPDATE orders 
SET payment_status = ?, updated_at = CURRENT_TIMESTAMP
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
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_orders,
    COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing_orders,
    COUNT(CASE WHEN status = 'shipped' THEN 1 END) as shipped_orders,
    COUNT(CASE WHEN status = 'delivered' THEN 1 END) as delivered_orders,
    COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_orders,
    SUM(total_cents) as total_revenue_cents,
    AVG(total_cents) as average_order_value_cents
FROM orders;