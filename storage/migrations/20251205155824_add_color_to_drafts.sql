-- +goose Up
-- +goose StatementBegin
ALTER TABLE custom_quote_drafts ADD COLUMN color TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- First, create a new table without the color column
CREATE TABLE custom_quote_drafts_new (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    session_id TEXT NOT NULL,
    user_id TEXT,

    name TEXT,
    email TEXT,

    current_step INTEGER NOT NULL DEFAULT 1,
    project_type TEXT,
    material TEXT,
    size TEXT,
    budget TEXT,
    timeline TEXT,
    description TEXT,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    recovery_email_sent_at DATETIME
);

-- Copy data (excluding color column)
INSERT INTO custom_quote_drafts_new
SELECT id, session_id, user_id, name, email, current_step, project_type,
       material, size, budget, timeline, description, created_at, updated_at,
       completed_at, recovery_email_sent_at
FROM custom_quote_drafts;

-- Drop old table
DROP TABLE custom_quote_drafts;

-- Rename new table to original name
ALTER TABLE custom_quote_drafts_new RENAME TO custom_quote_drafts;

-- Recreate indexes
CREATE INDEX idx_custom_quote_drafts_session ON custom_quote_drafts(session_id);
CREATE INDEX idx_custom_quote_drafts_email ON custom_quote_drafts(email);
CREATE INDEX idx_custom_quote_drafts_recovery ON custom_quote_drafts(completed_at, recovery_email_sent_at, updated_at);
-- +goose StatementEnd
