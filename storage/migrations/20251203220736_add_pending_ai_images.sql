-- +goose Up
-- +goose StatementBegin
CREATE TABLE pending_ai_images (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    source_image_url TEXT NOT NULL,
    generated_image_url TEXT NOT NULL,
    model_used TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at DATETIME
);

CREATE INDEX idx_pending_ai_images_product ON pending_ai_images(product_id);
CREATE INDEX idx_pending_ai_images_status ON pending_ai_images(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_pending_ai_images_status;
DROP INDEX IF EXISTS idx_pending_ai_images_product;
DROP TABLE IF EXISTS pending_ai_images;
-- +goose StatementEnd
