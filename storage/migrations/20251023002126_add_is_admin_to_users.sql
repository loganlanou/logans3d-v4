-- +goose Up
-- +goose StatementBegin

-- Add is_admin column to users table with default FALSE
ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove is_admin column from users table
ALTER TABLE users DROP COLUMN is_admin;

-- +goose StatementEnd
