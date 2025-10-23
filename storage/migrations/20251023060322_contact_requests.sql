-- +goose Up
-- +goose StatementBegin

CREATE TABLE contact_requests (
    id TEXT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    subject TEXT NOT NULL,
    message TEXT NOT NULL,
    newsletter_subscribe BOOLEAN DEFAULT FALSE,

    -- Metadata
    ip_address TEXT,
    user_agent TEXT,
    referrer TEXT,

    -- Status and Priority
    status TEXT DEFAULT 'new' CHECK(status IN ('new', 'in_progress', 'responded', 'resolved', 'spam')),
    priority TEXT DEFAULT 'normal' CHECK(priority IN ('low', 'normal', 'high', 'urgent')),

    -- Assignment
    assigned_to_user_id TEXT REFERENCES users(id),
    assigned_at DATETIME,

    -- Response tracking
    responded_at DATETIME,
    response_notes TEXT,

    -- Tags for categorization (comma-separated)
    tags TEXT,

    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Ensure at least email or phone is provided
    CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

-- Create indexes for better performance
CREATE INDEX idx_contact_requests_status ON contact_requests(status);
CREATE INDEX idx_contact_requests_priority ON contact_requests(priority);
CREATE INDEX idx_contact_requests_created_at ON contact_requests(created_at);
CREATE INDEX idx_contact_requests_email ON contact_requests(email);
CREATE INDEX idx_contact_requests_assigned_to ON contact_requests(assigned_to_user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_contact_requests_assigned_to;
DROP INDEX IF EXISTS idx_contact_requests_email;
DROP INDEX IF EXISTS idx_contact_requests_created_at;
DROP INDEX IF EXISTS idx_contact_requests_priority;
DROP INDEX IF EXISTS idx_contact_requests_status;
DROP TABLE IF EXISTS contact_requests;

-- +goose StatementEnd
