-- +goose Up
-- +goose StatementBegin
CREATE TABLE email_history (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    recipient_email TEXT NOT NULL,
    email_type TEXT NOT NULL,
    subject TEXT NOT NULL,
    template_name TEXT NOT NULL,
    sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    opened_at DATETIME,
    clicked_at DATETIME,
    tracking_token TEXT UNIQUE,
    metadata TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_email_history_user_id ON email_history(user_id);
CREATE INDEX idx_email_history_sent_at ON email_history(sent_at);
CREATE INDEX idx_email_history_type ON email_history(email_type);
CREATE INDEX idx_email_history_tracking_token ON email_history(tracking_token);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_email_history_tracking_token;
DROP INDEX IF EXISTS idx_email_history_type;
DROP INDEX IF EXISTS idx_email_history_sent_at;
DROP INDEX IF EXISTS idx_email_history_user_id;
DROP TABLE IF EXISTS email_history;
-- +goose StatementEnd
