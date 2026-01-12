-- name: CreateImportJob :one
INSERT INTO import_jobs (id, designer_slug, platform, job_type, status, started_at)
VALUES (?, ?, ?, ?, 'running', CURRENT_TIMESTAMP)
RETURNING *;

-- name: UpdateImportJobProgress :exec
UPDATE import_jobs
SET processed_items = ?, total_items = ?
WHERE id = ?;

-- name: CompleteImportJob :exec
UPDATE import_jobs
SET status = 'completed', completed_at = CURRENT_TIMESTAMP, processed_items = total_items
WHERE id = ?;

-- name: FailImportJob :exec
UPDATE import_jobs
SET status = 'failed', completed_at = CURRENT_TIMESTAMP, error_message = ?
WHERE id = ?;

-- name: GetImportJob :one
SELECT * FROM import_jobs WHERE id = ?;

-- name: ListImportJobs :many
SELECT * FROM import_jobs
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListImportJobsByDesigner :many
SELECT * FROM import_jobs
WHERE designer_slug = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: GetLatestJobByDesigner :one
SELECT * FROM import_jobs
WHERE designer_slug = ? AND platform = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: UpsertScrapedProduct :one
INSERT INTO scraped_products (
    id, designer_slug, platform, source_url, name, description,
    original_price_cents, release_date, image_urls, tags, raw_html, scraped_at,
    original_description, generated_description, description_model, description_generated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?, ?, ?, ?)
ON CONFLICT(source_url) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    original_price_cents = excluded.original_price_cents,
    release_date = excluded.release_date,
    image_urls = excluded.image_urls,
    tags = excluded.tags,
    raw_html = excluded.raw_html,
    scraped_at = CURRENT_TIMESTAMP,
    original_description = COALESCE(excluded.original_description, scraped_products.original_description),
    generated_description = COALESCE(excluded.generated_description, scraped_products.generated_description),
    description_model = COALESCE(excluded.description_model, scraped_products.description_model),
    description_generated_at = COALESCE(excluded.description_generated_at, scraped_products.description_generated_at)
RETURNING *;

-- name: GetScrapedProduct :one
SELECT * FROM scraped_products WHERE id = ?;

-- name: GetScrapedProductBySourceURL :one
SELECT * FROM scraped_products WHERE source_url = ?;

-- name: ListScrapedProductsByDesigner :many
SELECT * FROM scraped_products
WHERE designer_slug = ?
ORDER BY scraped_at DESC
LIMIT ? OFFSET ?;

-- name: ListUnimportedProducts :many
SELECT * FROM scraped_products
WHERE designer_slug = ? AND imported_product_id IS NULL
ORDER BY scraped_at DESC
LIMIT ? OFFSET ?;

-- name: CountScrapedProductsByDesigner :one
SELECT COUNT(*) as count FROM scraped_products WHERE designer_slug = ?;

-- name: CountUnimportedProductsByDesigner :one
SELECT COUNT(*) as count FROM scraped_products
WHERE designer_slug = ? AND imported_product_id IS NULL;

-- name: MarkProductImported :exec
UPDATE scraped_products
SET imported_product_id = ?
WHERE id = ?;

-- name: UpdateScrapedProductAI :exec
UPDATE scraped_products
SET ai_category = ?, ai_price_cents = ?, ai_size = ?
WHERE id = ?;

-- name: DeleteScrapedProduct :exec
DELETE FROM scraped_products WHERE id = ?;

-- name: DeleteScrapedProductsByDesigner :exec
DELETE FROM scraped_products WHERE designer_slug = ?;

-- name: SkipScrapedProduct :exec
UPDATE scraped_products
SET is_skipped = true, skip_reason = ?
WHERE id = ?;

-- name: UnskipScrapedProduct :exec
UPDATE scraped_products
SET is_skipped = false, skip_reason = NULL
WHERE id = ?;

-- name: ListScrapedProductsByDesignerFiltered :many
SELECT * FROM scraped_products
WHERE designer_slug = ?
  AND (
    CASE ?
      WHEN 'unimported' THEN imported_product_id IS NULL AND is_skipped = false
      WHEN 'imported' THEN imported_product_id IS NOT NULL
      WHEN 'skipped' THEN is_skipped = true
      ELSE true
    END
  )
ORDER BY scraped_at DESC
LIMIT ? OFFSET ?;

-- name: CountScrapedProductsByDesignerFiltered :one
SELECT COUNT(*) as count FROM scraped_products
WHERE designer_slug = ?
  AND (
    CASE ?
      WHEN 'unimported' THEN imported_product_id IS NULL AND is_skipped = false
      WHEN 'imported' THEN imported_product_id IS NOT NULL
      WHEN 'skipped' THEN is_skipped = true
      ELSE true
    END
  );

-- name: CountUnimportedNonSkippedByDesigner :one
SELECT COUNT(*) as count FROM scraped_products
WHERE designer_slug = ? AND imported_product_id IS NULL AND is_skipped = false;

-- name: GetPreviousScrapedProduct :one
-- Get the previous product (newer by scraped_at) in the same designer's list
SELECT sp.id FROM scraped_products sp
WHERE sp.designer_slug = ?
  AND sp.scraped_at > (SELECT sp2.scraped_at FROM scraped_products sp2 WHERE sp2.id = ?)
ORDER BY sp.scraped_at ASC
LIMIT 1;

-- name: GetNextScrapedProduct :one
-- Get the next product (older by scraped_at) in the same designer's list
SELECT sp.id FROM scraped_products sp
WHERE sp.designer_slug = ?
  AND sp.scraped_at < (SELECT sp2.scraped_at FROM scraped_products sp2 WHERE sp2.id = ?)
ORDER BY sp.scraped_at DESC
LIMIT 1;

-- Scraped Product Images

-- name: CreateScrapedProductImage :one
INSERT INTO scraped_product_images (
    id, scraped_product_id, source_url, local_filename,
    download_status, display_order, created_at
) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
RETURNING *;

-- name: CreateOrGetScrapedProductImage :one
-- Upsert query - creates a new image record or returns existing one if source_url already exists
INSERT INTO scraped_product_images (
    id, scraped_product_id, source_url, download_status, display_order, created_at
) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(scraped_product_id, source_url) DO UPDATE SET
    scraped_product_id = scraped_product_id
RETURNING *;

-- name: GetScrapedProductImageBySourceURL :one
SELECT * FROM scraped_product_images
WHERE scraped_product_id = ? AND source_url = ?;

-- name: GetScrapedProductImage :one
SELECT * FROM scraped_product_images WHERE id = ?;

-- name: ListScrapedProductImages :many
SELECT * FROM scraped_product_images
WHERE scraped_product_id = ?
ORDER BY display_order, created_at;

-- name: UpdateScrapedProductImageStatus :exec
UPDATE scraped_product_images
SET download_status = ?, download_error = ?, local_filename = ?
WHERE id = ?;

-- name: UpdateScrapedProductImageSelection :exec
UPDATE scraped_product_images
SET is_selected_for_import = ?
WHERE id = ?;

-- name: DeleteScrapedProductImages :exec
DELETE FROM scraped_product_images WHERE scraped_product_id = ?;

-- name: CountScrapedProductImagesByStatus :one
SELECT COUNT(*) as count FROM scraped_product_images
WHERE scraped_product_id = ? AND download_status = ?;

-- Scraped Product AI Images

-- name: CreateScrapedProductAIImage :one
INSERT INTO scraped_product_ai_images (
    id, scraped_product_id, source_image_id, local_filename,
    prompt_used, model_used, status, display_order, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
RETURNING *;

-- name: GetScrapedProductAIImage :one
SELECT * FROM scraped_product_ai_images WHERE id = ?;

-- name: ListScrapedProductAIImages :many
SELECT * FROM scraped_product_ai_images
WHERE scraped_product_id = ?
ORDER BY display_order, created_at;

-- name: UpdateScrapedProductAIImageStatus :exec
UPDATE scraped_product_ai_images
SET status = ?
WHERE id = ?;

-- name: UpdateScrapedProductAIImageSelection :exec
UPDATE scraped_product_ai_images
SET is_selected_for_import = ?
WHERE id = ?;

-- name: DeleteScrapedProductAIImages :exec
DELETE FROM scraped_product_ai_images WHERE scraped_product_id = ?;

-- Description Generation Queries

-- name: UpdateScrapedProductGeneratedContent :exec
UPDATE scraped_products
SET generated_name = ?,
    generated_description = ?,
    description_model = ?,
    description_generated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateScrapedProductGeneratedName :exec
-- For manual editing of the generated name
UPDATE scraped_products
SET generated_name = ?
WHERE id = ?;

-- name: UpdateScrapedProductGeneratedDescription :exec
-- For manual editing of the generated description
UPDATE scraped_products
SET generated_description = ?
WHERE id = ?;

-- name: ClearScrapedProductGeneratedContent :exec
UPDATE scraped_products
SET generated_name = NULL,
    generated_description = NULL,
    description_model = NULL,
    description_generated_at = NULL
WHERE id = ?;
