-- +goose Up
-- +goose StatementBegin
-- Drop the old unique index that includes both product_id and is_primary in the index columns
DROP INDEX IF EXISTS idx_product_images_primary;

-- Create a partial unique index that only enforces uniqueness on product_id for rows where is_primary = TRUE
-- This allows the CASE-based UPDATE to work without constraint violations
CREATE UNIQUE INDEX idx_product_images_primary ON product_images(product_id) WHERE is_primary = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Restore the original index that included is_primary in the index columns
DROP INDEX IF EXISTS idx_product_images_primary;

-- Recreate the original index structure
CREATE UNIQUE INDEX idx_product_images_primary ON product_images(product_id, is_primary) WHERE is_primary = TRUE;
-- +goose StatementEnd
