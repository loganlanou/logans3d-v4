-- +goose Up
-- +goose StatementBegin

-- Add recaptcha_score column to store Google reCAPTCHA v3 scores (0.0 to 1.0)
-- Higher scores indicate more likely human, lower scores indicate more likely bot
ALTER TABLE contact_requests ADD COLUMN recaptcha_score REAL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN in older versions
-- We need to recreate the table without the recaptcha_score column

-- Create temporary table with original schema
CREATE TABLE contact_requests_temp (
    id TEXT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    subject TEXT NOT NULL,
    message TEXT NOT NULL,
    newsletter_subscribe BOOLEAN DEFAULT FALSE,
    ip_address TEXT,
    user_agent TEXT,
    referrer TEXT,
    status TEXT DEFAULT 'new' CHECK(status IN ('new', 'in_progress', 'responded', 'resolved', 'spam')),
    priority TEXT DEFAULT 'normal' CHECK(priority IN ('low', 'normal', 'high', 'urgent')),
    assigned_to_user_id TEXT REFERENCES users(id),
    assigned_at DATETIME,
    responded_at DATETIME,
    response_notes TEXT,
    tags TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

-- Copy data from original table (excluding recaptcha_score)
INSERT INTO contact_requests_temp (
    id, first_name, last_name, email, phone, subject, message,
    newsletter_subscribe, ip_address, user_agent, referrer,
    status, priority, assigned_to_user_id, assigned_at,
    responded_at, response_notes, tags, created_at, updated_at
)
SELECT
    id, first_name, last_name, email, phone, subject, message,
    newsletter_subscribe, ip_address, user_agent, referrer,
    status, priority, assigned_to_user_id, assigned_at,
    responded_at, response_notes, tags, created_at, updated_at
FROM contact_requests;

-- Drop original table
DROP TABLE contact_requests;

-- Rename temp table to original name
ALTER TABLE contact_requests_temp RENAME TO contact_requests;

-- Recreate indexes
CREATE INDEX idx_contact_requests_status ON contact_requests(status);
CREATE INDEX idx_contact_requests_priority ON contact_requests(priority);
CREATE INDEX idx_contact_requests_created_at ON contact_requests(created_at);
CREATE INDEX idx_contact_requests_email ON contact_requests(email);
CREATE INDEX idx_contact_requests_assigned_to ON contact_requests(assigned_to_user_id);

-- +goose StatementEnd
