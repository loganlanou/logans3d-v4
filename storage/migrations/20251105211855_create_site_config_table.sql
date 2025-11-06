-- +goose Up
-- +goose StatementBegin
CREATE TABLE site_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Seed default site-wide SEO configuration
INSERT INTO site_config (key, value) VALUES
    ('site_name', 'Logan''s 3D Creations'),
    ('site_url', 'https://www.logans3dcreations.com'),
    ('site_description', 'Custom 3D printed collectibles, dinosaurs, and more'),
    ('default_og_image', '/public/images/social/default-og.jpg'),
    ('twitter_handle', ''),
    ('facebook_page_id', ''),
    ('facebook_app_id', '');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS site_config;
-- +goose StatementEnd
