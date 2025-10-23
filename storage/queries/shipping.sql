-- name: GetShippingConfig :one
SELECT * FROM shipping_config WHERE id = 1;

-- name: UpdateShippingConfig :one
UPDATE shipping_config
SET config_json = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: InsertShippingConfig :one
INSERT INTO shipping_config (id, config_json)
VALUES (1, ?)
ON CONFLICT(id) DO UPDATE SET
    config_json = excluded.config_json,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListBoxCatalog :many
SELECT * FROM box_catalog
WHERE is_active = TRUE
ORDER BY unit_cost_usd;

-- name: ListAllBoxCatalog :many
SELECT * FROM box_catalog
ORDER BY unit_cost_usd;

-- name: GetBoxBySKU :one
SELECT * FROM box_catalog WHERE sku = ? AND is_active = TRUE;

-- name: GetBoxByID :one
SELECT * FROM box_catalog WHERE id = ?;

-- name: CreateBoxCatalogItem :one
INSERT INTO box_catalog (
    id, sku, name, length_inches, width_inches, height_inches,
    box_weight_oz, unit_cost_usd, is_active
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateBoxCatalogItem :one
UPDATE box_catalog
SET name = ?, length_inches = ?, width_inches = ?, height_inches = ?,
    box_weight_oz = ?, unit_cost_usd = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
WHERE sku = ?
RETURNING *;

-- name: DeleteBoxCatalogItem :exec
UPDATE box_catalog SET is_active = FALSE WHERE sku = ?;

-- name: CreateOrderShippingSelection :one
INSERT INTO order_shipping_selection (
    id, order_id, candidate_box_sku, rate_id, carrier_id, service_code, service_name,
    quoted_shipping_amount_cents, quoted_box_cost_cents, quoted_total_cents,
    delivery_days, estimated_delivery_date, packing_solution_json, shipment_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetOrderShippingSelection :one
SELECT * FROM order_shipping_selection WHERE order_id = ?;

-- name: UpdateOrderShippingSelection :one
UPDATE order_shipping_selection
SET candidate_box_sku = ?, rate_id = ?, carrier_id = ?, service_code = ?, service_name = ?,
    quoted_shipping_amount_cents = ?, quoted_box_cost_cents = ?, quoted_total_cents = ?,
    delivery_days = ?, estimated_delivery_date = ?, packing_solution_json = ?, shipment_id = ?
WHERE order_id = ?
RETURNING *;

-- name: DeleteOrderShippingSelection :exec
DELETE FROM order_shipping_selection WHERE order_id = ?;

-- name: CreateShippingLabel :one
INSERT INTO shipping_labels (
    id, order_id, label_id, tracking_number, carrier_id, service_code,
    shipping_amount_cents, label_pdf_url, label_pdf_path, status, shipstation_created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetShippingLabel :one
SELECT * FROM shipping_labels WHERE order_id = ?;

-- name: GetShippingLabelByLabelID :one
SELECT * FROM shipping_labels WHERE label_id = ?;

-- name: GetShippingLabelByTrackingNumber :one
SELECT * FROM shipping_labels WHERE tracking_number = ?;

-- name: UpdateShippingLabelStatus :one
UPDATE shipping_labels
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE label_id = ?
RETURNING *;

-- name: VoidShippingLabel :one
UPDATE shipping_labels
SET status = 'voided', voided_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE label_id = ?
RETURNING *;

-- name: UpdateShippingLabelPDFPath :one
UPDATE shipping_labels
SET label_pdf_path = ?, updated_at = CURRENT_TIMESTAMP
WHERE label_id = ?
RETURNING *;

-- name: ListShippingLabels :many
SELECT * FROM shipping_labels
ORDER BY created_at DESC;

-- name: ListShippingLabelsByStatus :many
SELECT * FROM shipping_labels
WHERE status = ?
ORDER BY created_at DESC;

-- name: GetOrderWithShippingInfo :one
SELECT
    o.*,
    oss.candidate_box_sku,
    oss.rate_id,
    oss.carrier_id,
    oss.service_code,
    oss.service_name,
    oss.quoted_shipping_amount_cents,
    oss.quoted_box_cost_cents,
    oss.quoted_total_cents,
    oss.delivery_days,
    oss.estimated_delivery_date,
    sl.label_id,
    sl.tracking_number,
    sl.status as label_status,
    sl.label_pdf_path
FROM orders o
LEFT JOIN order_shipping_selection oss ON o.id = oss.order_id
LEFT JOIN shipping_labels sl ON o.id = sl.order_id
WHERE o.id = ?;

-- name: CountItemsByShippingCategory :one
SELECT
    SUM(CASE WHEN p.shipping_category = 'small' THEN oi.quantity ELSE 0 END) as small_items,
    SUM(CASE WHEN p.shipping_category = 'medium' THEN oi.quantity ELSE 0 END) as medium_items,
    SUM(CASE WHEN p.shipping_category = 'large' THEN oi.quantity ELSE 0 END) as large_items,
    SUM(CASE WHEN p.shipping_category = 'xlarge' THEN oi.quantity ELSE 0 END) as xlarge_items
FROM order_items oi
JOIN products p ON oi.product_id = p.id
WHERE oi.order_id = ?;

-- name: CountCartItemsByShippingCategory :one
SELECT
    SUM(CASE WHEN p.shipping_category = 'small' THEN ci.quantity ELSE 0 END) as small_items,
    SUM(CASE WHEN p.shipping_category = 'medium' THEN ci.quantity ELSE 0 END) as medium_items,
    SUM(CASE WHEN p.shipping_category = 'large' THEN ci.quantity ELSE 0 END) as large_items,
    SUM(CASE WHEN p.shipping_category = 'xlarge' THEN ci.quantity ELSE 0 END) as xlarge_items
FROM cart_items ci
JOIN products p ON ci.product_id = p.id
WHERE (ci.session_id = ? OR ci.user_id = ?);

-- name: UpdateProductShippingCategory :one
UPDATE products
SET shipping_category = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;