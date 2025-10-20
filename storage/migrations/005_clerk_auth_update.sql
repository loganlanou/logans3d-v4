-- +goose Up
-- +goose StatementBegin

-- Update users table for Clerk integration
ALTER TABLE users ADD COLUMN clerk_id TEXT;
ALTER TABLE users ADD COLUMN first_name TEXT;
ALTER TABLE users ADD COLUMN last_name TEXT;
ALTER TABLE users ADD COLUMN username TEXT;
ALTER TABLE users ADD COLUMN profile_image_url TEXT;
ALTER TABLE users ADD COLUMN last_synced_at DATETIME;

-- Rename old columns
ALTER TABLE users RENAME COLUMN name TO full_name;
ALTER TABLE users RENAME COLUMN avatar_url TO legacy_avatar_url;

-- Create unique index on clerk_id for fast lookups and uniqueness constraint
CREATE UNIQUE INDEX idx_users_clerk_id ON users(clerk_id);

-- Update user_sessions table to reference clerk session tokens
ALTER TABLE user_sessions ADD COLUMN clerk_session_id TEXT;
CREATE INDEX idx_user_sessions_clerk_session_id ON user_sessions(clerk_session_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_user_sessions_clerk_session_id;
DROP INDEX IF EXISTS idx_users_clerk_id;

ALTER TABLE user_sessions DROP COLUMN clerk_session_id;

ALTER TABLE users RENAME COLUMN legacy_avatar_url TO avatar_url;
ALTER TABLE users RENAME COLUMN full_name TO name;

ALTER TABLE users DROP COLUMN last_synced_at;
ALTER TABLE users DROP COLUMN profile_image_url;
ALTER TABLE users DROP COLUMN username;
ALTER TABLE users DROP COLUMN last_name;
ALTER TABLE users DROP COLUMN first_name;
ALTER TABLE users DROP COLUMN clerk_id;

-- +goose StatementEnd
