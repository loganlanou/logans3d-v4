-- +goose Up
-- +goose StatementBegin
CREATE TABLE email_preferences (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    email TEXT NOT NULL,
    transactional INTEGER DEFAULT 1,
    abandoned_cart INTEGER DEFAULT 1,
    promotional INTEGER DEFAULT 0,
    newsletter INTEGER DEFAULT 0,
    product_updates INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    unsubscribe_token TEXT UNIQUE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, email)
);

CREATE INDEX idx_email_preferences_user_id ON email_preferences(user_id);
CREATE INDEX idx_email_preferences_email ON email_preferences(email);
CREATE INDEX idx_email_preferences_unsubscribe_token ON email_preferences(unsubscribe_token);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_email_preferences_unsubscribe_token;
DROP INDEX IF EXISTS idx_email_preferences_email;
DROP INDEX IF EXISTS idx_email_preferences_user_id;
DROP TABLE IF EXISTS email_preferences;
-- +goose StatementEnd
