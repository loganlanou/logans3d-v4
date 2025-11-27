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
    quoted_shipping_amount_cents, quoted_box_cost_cents, quoted_handling_cost_cents, quoted_total_cents,
    delivery_days, estimated_delivery_date, packing_solution_json, shipment_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

WITH resolved AS (
    SELECT
        COALESCE(sc.default_shipping_class, p.shipping_category, 'unknown') AS category,
        COALESCE(sc.default_shipping_weight_oz, p.weight_grams / 28.35) as weight_oz,
        p.dimensions_length_mm / 25.4 as length_in,
        p.dimensions_width_mm / 25.4 as width_in,
        p.dimensions_height_mm / 25.4 as height_in,
        oi.quantity
    FROM order_items oi
    JOIN products p ON oi.product_id = p.id
    LEFT JOIN product_skus ps ON oi.product_sku_id = ps.id
    LEFT JOIN size_charts sc ON sc.size_id = ps.size_id
    WHERE oi.order_id = ?
)
SELECT
    SUM(CASE WHEN category = 'small' THEN quantity ELSE 0 END) as small_items,
    SUM(CASE WHEN category = 'medium' THEN quantity ELSE 0 END) as medium_items,
    SUM(CASE WHEN category = 'large' THEN quantity ELSE 0 END) as large_items,
    SUM(CASE WHEN category = 'xlarge' THEN quantity ELSE 0 END) as xlarge_items,
    SUM(CASE WHEN category = 'unknown' THEN quantity ELSE 0 END) as unknown_items,
    SUM(CASE WHEN category = 'small' THEN COALESCE(weight_oz,0) * quantity ELSE 0 END) as small_weight_oz,
    SUM(CASE WHEN category = 'medium' THEN COALESCE(weight_oz,0) * quantity ELSE 0 END) as medium_weight_oz,
    SUM(CASE WHEN category = 'large' THEN COALESCE(weight_oz,0) * quantity ELSE 0 END) as large_weight_oz,
    SUM(CASE WHEN category = 'xlarge' THEN COALESCE(weight_oz,0) * quantity ELSE 0 END) as xlarge_weight_oz
FROM resolved;

-- name: CountCartItemsByShippingCategory :one
WITH resolved AS (
    SELECT
        COALESCE(sc.default_shipping_class, p.shipping_category, 'unknown') AS category,
        COALESCE(sc.default_shipping_weight_oz, p.weight_grams / 28.35) as weight_oz,
        p.dimensions_length_mm / 25.4 as length_in,
        p.dimensions_width_mm / 25.4 as width_in,
        p.dimensions_height_mm / 25.4 as height_in,
        ci.quantity
    FROM cart_items ci
    JOIN products p ON ci.product_id = p.id
    LEFT JOIN product_skus ps ON ci.product_sku_id = ps.id
    LEFT JOIN size_charts sc ON sc.size_id = ps.size_id
    WHERE (ci.session_id = ? OR ci.user_id = ?)
)
SELECT
    SUM(CASE WHEN category = 'small' THEN quantity ELSE 0 END) as small_items,
    SUM(CASE WHEN category = 'medium' THEN quantity ELSE 0 END) as medium_items,
    SUM(CASE WHEN category = 'large' THEN quantity ELSE 0 END) as large_items,
    SUM(CASE WHEN category = 'xlarge' THEN quantity ELSE 0 END) as xlarge_items,
    SUM(CASE WHEN category = 'unknown' THEN quantity ELSE 0 END) as unknown_items,
    SUM(CASE WHEN category = 'small' THEN COALESCE(weight_oz,0) * quantity ELSE 0 END) as small_weight_oz,
    SUM(CASE WHEN category = 'medium' THEN COALESCE(weight_oz,0) * quantity ELSE 0 END) as medium_weight_oz,
    SUM(CASE WHEN category = 'large' THEN COALESCE(weight_oz,0) * quantity ELSE 0 END) as large_weight_oz,
    SUM(CASE WHEN category = 'xlarge' THEN COALESCE(weight_oz,0) * quantity ELSE 0 END) as xlarge_weight_oz,
    -- Missing weight counts (only items missing weight data)
    SUM(CASE WHEN category = 'small' AND (weight_oz IS NULL OR weight_oz <= 0) THEN quantity ELSE 0 END) as small_missing_weight,
    SUM(CASE WHEN category = 'medium' AND (weight_oz IS NULL OR weight_oz <= 0) THEN quantity ELSE 0 END) as medium_missing_weight,
    SUM(CASE WHEN category = 'large' AND (weight_oz IS NULL OR weight_oz <= 0) THEN quantity ELSE 0 END) as large_missing_weight,
    SUM(CASE WHEN category = 'xlarge' AND (weight_oz IS NULL OR weight_oz <= 0) THEN quantity ELSE 0 END) as xlarge_missing_weight,
    -- Missing dimension counts (items missing any dimension)
    SUM(CASE WHEN category = 'small' AND (length_in IS NULL OR length_in <= 0 OR width_in IS NULL OR width_in <= 0 OR height_in IS NULL OR height_in <= 0) THEN quantity ELSE 0 END) as small_missing_dims,
    SUM(CASE WHEN category = 'medium' AND (length_in IS NULL OR length_in <= 0 OR width_in IS NULL OR width_in <= 0 OR height_in IS NULL OR height_in <= 0) THEN quantity ELSE 0 END) as medium_missing_dims,
    SUM(CASE WHEN category = 'large' AND (length_in IS NULL OR length_in <= 0 OR width_in IS NULL OR width_in <= 0 OR height_in IS NULL OR height_in <= 0) THEN quantity ELSE 0 END) as large_missing_dims,
    SUM(CASE WHEN category = 'xlarge' AND (length_in IS NULL OR length_in <= 0 OR width_in IS NULL OR width_in <= 0 OR height_in IS NULL OR height_in <= 0) THEN quantity ELSE 0 END) as xlarge_missing_dims,
    MAX(CAST(CASE WHEN category = 'small' THEN COALESCE(length_in, 0.0) ELSE 0.0 END AS REAL)) as small_max_length_in,
    MAX(CAST(CASE WHEN category = 'small' THEN COALESCE(width_in, 0.0) ELSE 0.0 END AS REAL)) as small_max_width_in,
    MAX(CAST(CASE WHEN category = 'small' THEN COALESCE(height_in, 0.0) ELSE 0.0 END AS REAL)) as small_max_height_in,
    MAX(CAST(CASE WHEN category = 'medium' THEN COALESCE(length_in, 0.0) ELSE 0.0 END AS REAL)) as medium_max_length_in,
    MAX(CAST(CASE WHEN category = 'medium' THEN COALESCE(width_in, 0.0) ELSE 0.0 END AS REAL)) as medium_max_width_in,
    MAX(CAST(CASE WHEN category = 'medium' THEN COALESCE(height_in, 0.0) ELSE 0.0 END AS REAL)) as medium_max_height_in,
    MAX(CAST(CASE WHEN category = 'large' THEN COALESCE(length_in, 0.0) ELSE 0.0 END AS REAL)) as large_max_length_in,
    MAX(CAST(CASE WHEN category = 'large' THEN COALESCE(width_in, 0.0) ELSE 0.0 END AS REAL)) as large_max_width_in,
    MAX(CAST(CASE WHEN category = 'large' THEN COALESCE(height_in, 0.0) ELSE 0.0 END AS REAL)) as large_max_height_in,
    MAX(CAST(CASE WHEN category = 'xlarge' THEN COALESCE(length_in, 0.0) ELSE 0.0 END AS REAL)) as xlarge_max_length_in,
    MAX(CAST(CASE WHEN category = 'xlarge' THEN COALESCE(width_in, 0.0) ELSE 0.0 END AS REAL)) as xlarge_max_width_in,
    MAX(CAST(CASE WHEN category = 'xlarge' THEN COALESCE(height_in, 0.0) ELSE 0.0 END AS REAL)) as xlarge_max_height_in
FROM resolved;

-- name: UpdateProductShippingCategory :one
UPDATE products
SET shipping_category = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- Session Shipping Selection Queries

-- name: CreateSessionShippingSelection :one
INSERT INTO session_shipping_selection (
    id, session_id, rate_id, shipment_id, carrier_name, service_name,
    price_cents, shipping_amount_cents, box_cost_cents, handling_cost_cents, box_sku,
    delivery_days, estimated_date,
    cart_snapshot_json, shipping_address_json, is_valid
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSessionShippingSelection :one
SELECT * FROM session_shipping_selection WHERE session_id = ? LIMIT 1;

-- name: UpdateSessionShippingSelection :one
UPDATE session_shipping_selection
SET rate_id = ?, shipment_id = ?, carrier_name = ?, service_name = ?,
    price_cents = ?, shipping_amount_cents = ?, box_cost_cents = ?, handling_cost_cents = ?, box_sku = ?,
    delivery_days = ?, estimated_date = ?,
    cart_snapshot_json = ?, shipping_address_json = ?, is_valid = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE session_id = ?
RETURNING *;

-- name: InvalidateSessionShipping :exec
UPDATE session_shipping_selection
SET is_valid = FALSE, updated_at = CURRENT_TIMESTAMP
WHERE session_id = ?;

-- name: DeleteSessionShippingSelection :exec
DELETE FROM session_shipping_selection WHERE session_id = ?;

-- Carrier Account Queries

-- name: GetCarrierAccountsByLocation :many
SELECT * FROM carrier_accounts WHERE origin_zip = ?;

-- name: GetCarrierAccountByType :one
SELECT * FROM carrier_accounts WHERE carrier_type = ? LIMIT 1;

-- name: ListAllCarrierAccounts :many
SELECT * FROM carrier_accounts ORDER BY carrier_type;
