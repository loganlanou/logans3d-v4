-- +goose Up
-- +goose StatementBegin

-- Drop existing index
DROP INDEX IF EXISTS idx_users_clerk_id;

-- Create UNIQUE index on clerk_id (without WHERE clause so ON CONFLICT works)
CREATE UNIQUE INDEX idx_users_clerk_id_unique ON users(clerk_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop unique index
DROP INDEX IF EXISTS idx_users_clerk_id_unique;

-- Recreate non-unique index
CREATE INDEX idx_users_clerk_id ON users(clerk_id);

-- +goose StatementEnd
