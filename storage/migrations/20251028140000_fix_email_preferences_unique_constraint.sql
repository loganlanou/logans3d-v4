-- +goose Up
-- +goose StatementBegin
-- SQLite doesn't support modifying constraints directly, so we need to recreate the table
-- This migration fixes the UNIQUE constraint to be on email only, not (user_id, email)
-- This prevents duplicate email_preferences records for the same email address

-- Create new table with correct constraint
CREATE TABLE email_preferences_new (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    email TEXT NOT NULL UNIQUE,  -- Changed from UNIQUE(user_id, email) to UNIQUE on email
    transactional INTEGER DEFAULT 1,
    abandoned_cart INTEGER DEFAULT 1,
    promotional INTEGER DEFAULT 0,
    newsletter INTEGER DEFAULT 0,
    product_updates INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    unsubscribe_token TEXT UNIQUE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy data from old table, handling duplicates by keeping the most recent record per email
INSERT INTO email_preferences_new
SELECT
    ep.id,
    ep.user_id,
    ep.email,
    ep.transactional,
    ep.abandoned_cart,
    ep.promotional,
    ep.newsletter,
    ep.product_updates,
    ep.created_at,
    ep.updated_at,
    ep.unsubscribe_token
FROM email_preferences ep
INNER JOIN (
    -- Get the most recent record for each email
    SELECT email, MAX(created_at) as max_created
    FROM email_preferences
    GROUP BY email
) latest ON ep.email = latest.email AND ep.created_at = latest.max_created;

-- Drop old table
DROP TABLE email_preferences;

-- Rename new table
ALTER TABLE email_preferences_new RENAME TO email_preferences;

-- Recreate indexes
CREATE INDEX idx_email_preferences_user_id ON email_preferences(user_id);
CREATE INDEX idx_email_preferences_email ON email_preferences(email);
CREATE INDEX idx_email_preferences_unsubscribe_token ON email_preferences(unsubscribe_token);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Recreate table with old constraint
CREATE TABLE email_preferences_old (
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

-- Copy data back
INSERT INTO email_preferences_old SELECT * FROM email_preferences;

-- Drop new table
DROP TABLE email_preferences;

-- Rename old table back
ALTER TABLE email_preferences_old RENAME TO email_preferences;

-- Recreate indexes
CREATE INDEX idx_email_preferences_user_id ON email_preferences(user_id);
CREATE INDEX idx_email_preferences_email ON email_preferences(email);
CREATE INDEX idx_email_preferences_unsubscribe_token ON email_preferences(unsubscribe_token);
-- +goose StatementEnd
