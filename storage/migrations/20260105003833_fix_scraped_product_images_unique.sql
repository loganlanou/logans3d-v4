-- +goose Up
-- +goose StatementBegin

-- Add unique constraint to prevent duplicate images for the same product+URL
-- Must clean up any existing duplicates before this can be applied
CREATE UNIQUE INDEX IF NOT EXISTS idx_scraped_product_images_unique
ON scraped_product_images(scraped_product_id, source_url);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_scraped_product_images_unique;

-- +goose StatementEnd
