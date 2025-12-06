-- +goose Up
-- +goose StatementBegin
ALTER TABLE custom_quote_drafts ADD COLUMN archived_at DATETIME;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN directly, need to recreate table
CREATE TABLE custom_quote_drafts_backup (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    user_id TEXT,
    name TEXT,
    email TEXT,
    current_step INTEGER NOT NULL DEFAULT 1,
    project_type TEXT,
    material TEXT,
    size TEXT,
    color TEXT,
    budget TEXT,
    timeline TEXT,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    recovery_email_sent_at DATETIME
);

INSERT INTO custom_quote_drafts_backup
SELECT id, session_id, user_id, name, email, current_step, project_type, material, size, color, budget, timeline, description, created_at, updated_at, completed_at, recovery_email_sent_at
FROM custom_quote_drafts;

DROP TABLE custom_quote_drafts;

ALTER TABLE custom_quote_drafts_backup RENAME TO custom_quote_drafts;

CREATE INDEX idx_custom_quote_drafts_session_id ON custom_quote_drafts(session_id);
CREATE INDEX idx_custom_quote_drafts_email ON custom_quote_drafts(email);
CREATE INDEX idx_custom_quote_drafts_updated_at ON custom_quote_drafts(updated_at);
-- +goose StatementEnd
