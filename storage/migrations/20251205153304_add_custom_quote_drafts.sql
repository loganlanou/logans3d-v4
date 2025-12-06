-- +goose Up
-- +goose StatementBegin
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
    recovery_email_sent_at DATETIME
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_custom_quote_drafts_session ON custom_quote_drafts(session_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_custom_quote_drafts_email ON custom_quote_drafts(email);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_custom_quote_drafts_recovery ON custom_quote_drafts(completed_at, recovery_email_sent_at, updated_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE custom_quote_draft_files (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    draft_id TEXT NOT NULL REFERENCES custom_quote_drafts(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    file_type TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_custom_quote_draft_files_draft ON custom_quote_draft_files(draft_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_custom_quote_draft_files_draft;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS custom_quote_draft_files;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_custom_quote_drafts_recovery;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_custom_quote_drafts_email;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_custom_quote_drafts_session;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS custom_quote_drafts;
-- +goose StatementEnd
