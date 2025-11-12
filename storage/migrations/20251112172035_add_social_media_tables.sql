-- +goose Up
-- +goose StatementBegin
CREATE TABLE social_media_posts (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    platform TEXT NOT NULL,
    post_copy TEXT NOT NULL,
    hashtags TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

CREATE INDEX idx_social_media_posts_product_id ON social_media_posts(product_id);
CREATE INDEX idx_social_media_posts_platform ON social_media_posts(platform);
CREATE UNIQUE INDEX idx_social_media_posts_product_platform ON social_media_posts(product_id, platform);

CREATE TABLE social_media_tasks (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    platform TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    posted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CHECK (status IN ('pending', 'posted', 'skipped'))
);

CREATE INDEX idx_social_media_tasks_product_id ON social_media_tasks(product_id);
CREATE INDEX idx_social_media_tasks_platform ON social_media_tasks(platform);
CREATE INDEX idx_social_media_tasks_status ON social_media_tasks(status);
CREATE UNIQUE INDEX idx_social_media_tasks_product_platform ON social_media_tasks(product_id, platform);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS social_media_tasks;
DROP TABLE IF EXISTS social_media_posts;
-- +goose StatementEnd
