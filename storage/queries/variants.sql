-- Product Style and SKU management
-- Styles are product-specific (not global)

-- ============================================
-- Product Styles
-- ============================================

-- name: CreateProductStyle :one
INSERT INTO product_styles (id, product_id, name, is_primary, display_order)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProductStyle :one
SELECT * FROM product_styles WHERE id = ?;

-- name: GetProductStyles :many
SELECT * FROM product_styles
WHERE product_id = ?
ORDER BY is_primary DESC, display_order ASC, created_at ASC;

-- name: UpdateProductStyle :one
UPDATE product_styles
SET name = ?, is_primary = ?, display_order = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteProductStyle :exec
DELETE FROM product_styles WHERE id = ?;

-- name: ClearPrimaryProductStyles :exec
UPDATE product_styles
SET is_primary = FALSE
WHERE product_id = ?;

-- name: SetPrimaryProductStyle :exec
UPDATE product_styles
SET is_primary = TRUE
WHERE id = ?;

-- ============================================
-- Product Style Images
-- ============================================

-- name: CreateProductStyleImage :one
INSERT INTO product_style_images (id, product_style_id, image_url, is_primary, display_order)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProductStyleImage :one
SELECT * FROM product_style_images WHERE id = ?;

-- name: GetProductStyleImages :many
SELECT * FROM product_style_images
WHERE product_style_id = ?
ORDER BY is_primary DESC, display_order ASC, created_at ASC;

-- name: DeleteProductStyleImage :exec
DELETE FROM product_style_images WHERE id = ?;

-- name: ClearPrimaryStyleImages :exec
UPDATE product_style_images
SET is_primary = FALSE
WHERE product_style_id = ?;

-- name: SetPrimaryStyleImage :exec
UPDATE product_style_images
SET is_primary = TRUE
WHERE id = ?;

-- name: GetPrimaryStyleImage :one
SELECT * FROM product_style_images
WHERE product_style_id = ?
ORDER BY is_primary DESC, display_order ASC, created_at ASC
LIMIT 1;

-- ============================================
-- Product Styles with Images (for display)
-- ============================================

-- name: GetProductStylesWithPrimaryImage :many
SELECT
    ps.*,
    COALESCE(psi.image_url, '') as primary_image
FROM product_styles ps
LEFT JOIN product_style_images psi ON psi.product_style_id = ps.id AND psi.is_primary = TRUE
WHERE ps.product_id = ?
ORDER BY ps.is_primary DESC, ps.display_order ASC, ps.created_at ASC;

-- ============================================
-- Sizes (global)
-- ============================================

-- name: GetAllSizes :many
SELECT * FROM sizes
ORDER BY display_order ASC;

-- name: GetSize :one
SELECT * FROM sizes WHERE id = ?;

-- ============================================
-- Size Charts (global shipping/price defaults)
-- ============================================

-- name: GetSizeChart :one
SELECT * FROM size_charts WHERE size_id = ?;

-- name: GetSizeCharts :many
SELECT
    s.id as size_id,
    s.name,
    s.display_name,
    s.display_order,
    sc.id as chart_id,
    sc.default_shipping_class,
    sc.default_shipping_weight_oz,
    sc.default_price_adjustment_cents
FROM sizes s
LEFT JOIN size_charts sc ON sc.size_id = s.id
ORDER BY s.display_order ASC;

-- name: UpsertSizeChart :one
INSERT INTO size_charts (id, size_id, default_shipping_class, default_shipping_weight_oz, default_price_adjustment_cents)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(size_id) DO UPDATE SET
    default_shipping_class = excluded.default_shipping_class,
    default_shipping_weight_oz = excluded.default_shipping_weight_oz,
    default_price_adjustment_cents = excluded.default_price_adjustment_cents,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateSizeChartPriceAdjustment :exec
UPDATE size_charts
SET default_price_adjustment_cents = ?, updated_at = CURRENT_TIMESTAMP
WHERE size_id = ?;

-- ============================================
-- Product Size Configs (product-level size settings)
-- ============================================

-- name: GetProductSizeConfigs :many
SELECT
    psc.*,
    s.name as size_name,
    s.display_name as size_display_name,
    COALESCE(sc.default_price_adjustment_cents, 0) as chart_default_adjustment
FROM product_size_configs psc
JOIN sizes s ON s.id = psc.size_id
LEFT JOIN size_charts sc ON sc.size_id = psc.size_id
WHERE psc.product_id = ? AND psc.is_enabled = TRUE
ORDER BY psc.display_order, s.display_order;

-- name: GetAllProductSizeConfigs :many
SELECT
    psc.*,
    s.name as size_name,
    s.display_name as size_display_name,
    COALESCE(sc.default_price_adjustment_cents, 0) as chart_default_adjustment
FROM product_size_configs psc
JOIN sizes s ON s.id = psc.size_id
LEFT JOIN size_charts sc ON sc.size_id = psc.size_id
WHERE psc.product_id = ?
ORDER BY psc.display_order, s.display_order;

-- name: UpsertProductSizeConfig :one
INSERT INTO product_size_configs (id, product_id, size_id, price_adjustment_cents, is_enabled, display_order)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(product_id, size_id) DO UPDATE SET
    price_adjustment_cents = excluded.price_adjustment_cents,
    is_enabled = excluded.is_enabled,
    display_order = excluded.display_order
RETURNING *;

-- name: DeleteProductSizeConfig :exec
DELETE FROM product_size_configs WHERE product_id = ? AND size_id = ?;

-- name: DeleteAllProductSizeConfigs :exec
DELETE FROM product_size_configs WHERE product_id = ?;

-- ============================================
-- Product SKUs (Style + Size = purchasable item)
-- ============================================

-- name: CreateProductSku :one
INSERT INTO product_skus (id, product_id, product_style_id, size_id, sku, price_adjustment_cents, stock_quantity, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProductSku :one
SELECT * FROM product_skus WHERE id = ?;

-- name: GetProductSkuForProduct :one
SELECT * FROM product_skus WHERE id = ? AND product_id = ?;

-- name: GetProductSkuBySku :one
SELECT * FROM product_skus WHERE sku = ?;

-- name: GetProductSkus :many
SELECT
    ps.*,
    pst.name as style_name,
    pst.is_primary as style_is_primary,
    s.name as size_name,
    s.display_name as size_display_name,
    COALESCE(psi.image_url, '') as style_primary_image
FROM product_skus ps
JOIN product_styles pst ON pst.id = ps.product_style_id
JOIN sizes s ON s.id = ps.size_id
LEFT JOIN product_style_images psi ON psi.product_style_id = pst.id AND psi.is_primary = TRUE
WHERE ps.product_id = ?
ORDER BY pst.display_order, s.display_order;

-- name: GetProductSkuByStyleAndSize :one
SELECT * FROM product_skus
WHERE product_id = ? AND product_style_id = ? AND size_id = ?
LIMIT 1;

-- name: UpdateProductSku :one
UPDATE product_skus
SET sku = ?, price_adjustment_cents = ?, stock_quantity = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteProductSku :exec
DELETE FROM product_skus WHERE id = ?;

-- name: DeleteProductSkusByStyle :exec
DELETE FROM product_skus WHERE product_style_id = ?;

-- name: CheckSkuExists :one
SELECT EXISTS(
    SELECT 1 FROM product_skus WHERE UPPER(sku) = UPPER(sqlc.arg(sku))
) as sku_exists;

-- name: DecrementProductSkuStock :exec
UPDATE product_skus
SET stock_quantity = stock_quantity - sqlc.arg(delta), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND stock_quantity >= sqlc.arg(delta);

-- ============================================
-- Shop-facing queries (for product pages)
-- ============================================

-- name: GetProductVariantStyles :many
SELECT
    ps.id,
    ps.name,
    ps.is_primary,
    ps.display_order,
    COALESCE(psi.image_url, '') as primary_image
FROM product_styles ps
LEFT JOIN product_style_images psi ON psi.product_style_id = ps.id AND psi.is_primary = TRUE
WHERE ps.product_id = ?
ORDER BY ps.is_primary DESC, ps.display_order ASC, ps.name ASC;

-- name: GetProductVariantSizesForStyle :many
SELECT
    psku.id as product_sku_id,
    psku.sku,
    psku.price_adjustment_cents,
    psku.stock_quantity,
    psku.is_active,
    s.id as size_id,
    s.name as size_name,
    s.display_name as size_display_name,
    s.display_order as size_display_order
FROM product_skus psku
JOIN sizes s ON s.id = psku.size_id
WHERE psku.product_id = ? AND psku.product_style_id = ? AND psku.is_active = TRUE
ORDER BY s.display_order, s.display_name;

-- name: GetSkuByStyleAndSize :one
SELECT psku.*
FROM product_skus psku
WHERE psku.product_id = ?
    AND psku.product_style_id = ?
    AND psku.size_id = ?
LIMIT 1;

-- ============================================
-- Style Panel queries (for admin UI)
-- ============================================

-- name: GetStyleSkus :many
-- Get all SKUs for a specific style (for style detail panel)
SELECT
    psku.*,
    s.name as size_name,
    s.display_name as size_display_name,
    s.display_order as size_display_order
FROM product_skus psku
JOIN sizes s ON s.id = psku.size_id
WHERE psku.product_style_id = ?
ORDER BY s.display_order;

-- name: GetAvailableSizesForStyle :many
-- Get enabled sizes that DON'T have SKUs for this style yet
SELECT s.* FROM sizes s
JOIN product_size_configs psc ON psc.size_id = s.id AND psc.product_id = @product_id
LEFT JOIN product_skus psku ON psku.product_style_id = @product_style_id AND psku.size_id = s.id
WHERE psc.is_enabled = TRUE AND psku.id IS NULL
ORDER BY s.display_order;

-- name: UpdateSkuPrice :exec
UPDATE product_skus
SET price_adjustment_cents = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateSkuStock :exec
UPDATE product_skus
SET stock_quantity = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateSkuActive :exec
UPDATE product_skus
SET is_active = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CountStyleSkus :one
-- Count SKUs for a specific style
SELECT COUNT(*) as sku_count
FROM product_skus
WHERE product_style_id = ?;

-- ============================================
-- Multi-Variant Sharing queries (for admin sharing)
-- ============================================

-- name: GetProductPriceRange :one
-- Get price range across all SKUs for a product
SELECT
    p.price_cents as base_price,
    MIN(p.price_cents + COALESCE(psku.price_adjustment_cents, 0)) as min_price,
    MAX(p.price_cents + COALESCE(psku.price_adjustment_cents, 0)) as max_price,
    COUNT(DISTINCT ps.id) as style_count
FROM products p
LEFT JOIN product_styles ps ON ps.product_id = p.id
LEFT JOIN product_skus psku ON psku.product_id = p.id AND psku.is_active = TRUE
WHERE p.id = ?
GROUP BY p.id;

-- name: GetAllStylePrimaryImages :many
-- Get primary image for each style (for OG grid and carousel)
SELECT
    ps.id as style_id,
    ps.name as style_name,
    psi.image_url
FROM product_styles ps
JOIN product_style_images psi ON psi.product_style_id = ps.id AND psi.is_primary = TRUE
WHERE ps.product_id = ?
ORDER BY ps.is_primary DESC, ps.display_order ASC
LIMIT 10;
