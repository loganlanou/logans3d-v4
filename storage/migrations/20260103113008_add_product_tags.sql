-- +goose Up
-- +goose StatementBegin
CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_tags (
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, tag_id)
);

CREATE INDEX idx_product_tags_product ON product_tags(product_id);
CREATE INDEX idx_product_tags_tag ON product_tags(tag_id);
CREATE INDEX idx_tags_slug ON tags(slug);

-- Seed common tags for 3D print models
INSERT INTO tags (id, name, slug) VALUES
    ('tag-articulated', 'Articulated', 'articulated'),
    ('tag-print-in-place', 'Print-in-Place', 'print-in-place'),
    ('tag-no-supports', 'No Supports', 'no-supports'),
    ('tag-flexi', 'Flexi', 'flexi'),
    ('tag-multipart', 'Multi-Part', 'multipart'),
    ('tag-easy-print', 'Easy Print', 'easy-print');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tags_slug;
DROP INDEX IF EXISTS idx_product_tags_tag;
DROP INDEX IF EXISTS idx_product_tags_product;
DROP TABLE IF EXISTS product_tags;
DROP TABLE IF EXISTS tags;
-- +goose StatementEnd
