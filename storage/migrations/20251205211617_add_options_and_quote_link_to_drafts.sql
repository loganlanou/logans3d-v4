-- +goose Up
-- Add checkbox option fields and link to quote_requests
ALTER TABLE custom_quote_drafts ADD COLUMN finishing INTEGER DEFAULT 0;
ALTER TABLE custom_quote_drafts ADD COLUMN painting INTEGER DEFAULT 0;
ALTER TABLE custom_quote_drafts ADD COLUMN rush INTEGER DEFAULT 0;
ALTER TABLE custom_quote_drafts ADD COLUMN need_design INTEGER DEFAULT 0;
ALTER TABLE custom_quote_drafts ADD COLUMN quote_request_id TEXT REFERENCES quote_requests(id);

-- Index for looking up draft by quote request
CREATE INDEX idx_custom_quote_drafts_quote_request ON custom_quote_drafts(quote_request_id);

-- +goose Down
DROP INDEX IF EXISTS idx_custom_quote_drafts_quote_request;

-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
CREATE TABLE custom_quote_drafts_backup AS SELECT
    id, session_id, user_id, name, email, current_step, project_type,
    material, size, budget, timeline, description, created_at, updated_at,
    completed_at, recovery_email_sent_at, color, archived_at
FROM custom_quote_drafts;

DROP TABLE custom_quote_drafts;

CREATE TABLE custom_quote_drafts (
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
    recovery_email_sent_at DATETIME,
    color TEXT,
    archived_at DATETIME
);

INSERT INTO custom_quote_drafts SELECT * FROM custom_quote_drafts_backup;
DROP TABLE custom_quote_drafts_backup;

CREATE INDEX idx_custom_quote_drafts_session ON custom_quote_drafts(session_id);
CREATE INDEX idx_custom_quote_drafts_email ON custom_quote_drafts(email);
CREATE INDEX idx_custom_quote_drafts_recovery ON custom_quote_drafts(completed_at, recovery_email_sent_at, updated_at);
