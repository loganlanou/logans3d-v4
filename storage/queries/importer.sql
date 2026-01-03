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
    original_price_cents, release_date, image_urls, tags, raw_html, scraped_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(source_url) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    original_price_cents = excluded.original_price_cents,
    release_date = excluded.release_date,
    image_urls = excluded.image_urls,
    tags = excluded.tags,
    raw_html = excluded.raw_html,
    scraped_at = CURRENT_TIMESTAMP
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
