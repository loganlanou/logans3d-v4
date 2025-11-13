-- name: CreateSocialMediaPost :one
INSERT INTO social_media_posts (
    id,
    product_id,
    platform,
    post_copy,
    hashtags,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
RETURNING *;

-- name: GetSocialMediaPost :one
SELECT * FROM social_media_posts
WHERE id = ?;

-- name: GetSocialMediaPostByProductAndPlatform :one
SELECT * FROM social_media_posts
WHERE product_id = ? AND platform = ?;

-- name: GetSocialMediaPostsByProduct :many
SELECT * FROM social_media_posts
WHERE product_id = ?
ORDER BY platform;

-- name: UpdateSocialMediaPost :exec
UPDATE social_media_posts
SET post_copy = ?,
    hashtags = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteSocialMediaPost :exec
DELETE FROM social_media_posts
WHERE id = ?;

-- name: CreateSocialMediaTask :one
INSERT INTO social_media_tasks (
    id,
    product_id,
    platform,
    status,
    created_at
) VALUES (?, ?, ?, ?, datetime('now'))
RETURNING *;

-- name: GetSocialMediaTask :one
SELECT * FROM social_media_tasks
WHERE id = ?;

-- name: GetSocialMediaTaskByProductAndPlatform :one
SELECT * FROM social_media_tasks
WHERE product_id = ? AND platform = ?;

-- name: GetSocialMediaTasksByProduct :many
SELECT * FROM social_media_tasks
WHERE product_id = ?
ORDER BY platform;

-- name: UpdateSocialMediaTaskStatus :exec
UPDATE social_media_tasks
SET status = ?,
    posted_at = ?
WHERE id = ?;

-- name: DeleteSocialMediaTask :exec
DELETE FROM social_media_tasks
WHERE id = ?;

-- name: ListProductsWithPostingStatus :many
SELECT
    p.id,
    p.name,
    p.slug,
    p.price_cents,
    p.category_id,
    p.is_active,
    p.is_featured,
    c.name as category_name,
    COUNT(DISTINCT CASE WHEN smt.status = 'posted' THEN smt.platform END) as platforms_posted,
    COUNT(DISTINCT CASE WHEN smt.status = 'pending' THEN smt.platform END) as platforms_pending,
    COUNT(DISTINCT smt.platform) as total_platforms,
    COUNT(DISTINCT oi.id) as times_sold
FROM products p
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN social_media_tasks smt ON p.id = smt.product_id
LEFT JOIN order_items oi ON p.id = oi.product_id
LEFT JOIN orders o ON oi.order_id = o.id AND o.status IN ('paid', 'processing', 'shipped', 'delivered')
WHERE p.is_active = TRUE
GROUP BY p.id, p.name, p.slug, p.price_cents, p.category_id, p.is_active, p.is_featured, c.name
HAVING COUNT(DISTINCT CASE WHEN smt.status = 'posted' THEN smt.platform END) < 4
ORDER BY times_sold DESC, p.name;

-- name: GetBestSellingProducts :many
SELECT
    p.*,
    COUNT(oi.id) as times_sold,
    SUM(oi.quantity) as total_quantity_sold
FROM products p
LEFT JOIN order_items oi ON oi.product_id = p.id
LEFT JOIN orders o ON oi.order_id = o.id
WHERE p.is_active = TRUE
    AND (o.status IS NULL OR o.status IN ('paid', 'processing', 'shipped', 'delivered'))
GROUP BY p.id
ORDER BY times_sold DESC, total_quantity_sold DESC
LIMIT ?;

-- name: GetProductWithImages :one
SELECT
    p.*,
    GROUP_CONCAT(pi.image_url) as image_urls,
    GROUP_CONCAT(pi.is_primary) as image_primaries
FROM products p
LEFT JOIN product_images pi ON p.id = pi.product_id
WHERE p.id = ?
GROUP BY p.id;

-- name: DeleteAllPendingPosts :exec
DELETE FROM social_media_posts
WHERE (product_id, platform) IN (
    SELECT product_id, platform
    FROM social_media_tasks
    WHERE status = 'pending'
);

-- name: DeleteAllPendingTasks :exec
DELETE FROM social_media_tasks
WHERE status = 'pending';
