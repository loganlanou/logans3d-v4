-- +goose Up
-- +goose StatementBegin

-- Table to track abandoned cart events
CREATE TABLE abandoned_carts (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    user_id TEXT REFERENCES users(id),
    customer_email TEXT,
    customer_name TEXT,
    cart_value_cents INTEGER NOT NULL DEFAULT 0,
    item_count INTEGER NOT NULL DEFAULT 0,
    abandoned_at DATETIME NOT NULL,
    recovered_at DATETIME,
    recovery_method TEXT, -- email_1hr, email_24hr, email_72hr, manual, organic
    status TEXT DEFAULT 'active', -- active, recovered, expired, contacted
    last_contacted_at DATETIME,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Ensure either session_id or user_id is set
    CHECK ((session_id IS NULL) != (user_id IS NULL) OR (session_id IS NOT NULL AND user_id IS NOT NULL))
);

-- Table to track cart recovery attempts (emails, manual outreach)
CREATE TABLE cart_recovery_attempts (
    id TEXT PRIMARY KEY,
    abandoned_cart_id TEXT NOT NULL REFERENCES abandoned_carts(id) ON DELETE CASCADE,
    attempt_type TEXT NOT NULL, -- email_1hr, email_24hr, email_72hr, manual
    sent_at DATETIME NOT NULL,
    opened_at DATETIME,
    clicked_at DATETIME,
    status TEXT DEFAULT 'sent', -- sent, opened, clicked, bounced, failed
    email_subject TEXT,
    tracking_token TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Table to store cart contents at time of abandonment
CREATE TABLE cart_snapshots (
    id TEXT PRIMARY KEY,
    abandoned_cart_id TEXT NOT NULL REFERENCES abandoned_carts(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id),
    product_name TEXT NOT NULL,
    product_sku TEXT,
    product_image_url TEXT,
    quantity INTEGER NOT NULL,
    unit_price_cents INTEGER NOT NULL,
    total_price_cents INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_abandoned_carts_session_id ON abandoned_carts(session_id);
CREATE INDEX idx_abandoned_carts_user_id ON abandoned_carts(user_id);
CREATE INDEX idx_abandoned_carts_status ON abandoned_carts(status);
CREATE INDEX idx_abandoned_carts_abandoned_at ON abandoned_carts(abandoned_at);
CREATE INDEX idx_abandoned_carts_customer_email ON abandoned_carts(customer_email);

CREATE INDEX idx_cart_recovery_attempts_abandoned_cart_id ON cart_recovery_attempts(abandoned_cart_id);
CREATE INDEX idx_cart_recovery_attempts_attempt_type ON cart_recovery_attempts(attempt_type);
CREATE INDEX idx_cart_recovery_attempts_sent_at ON cart_recovery_attempts(sent_at);
CREATE INDEX idx_cart_recovery_attempts_tracking_token ON cart_recovery_attempts(tracking_token);

CREATE INDEX idx_cart_snapshots_abandoned_cart_id ON cart_snapshots(abandoned_cart_id);
CREATE INDEX idx_cart_snapshots_product_id ON cart_snapshots(product_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes first
DROP INDEX IF EXISTS idx_cart_snapshots_product_id;
DROP INDEX IF EXISTS idx_cart_snapshots_abandoned_cart_id;

DROP INDEX IF EXISTS idx_cart_recovery_attempts_tracking_token;
DROP INDEX IF EXISTS idx_cart_recovery_attempts_sent_at;
DROP INDEX IF EXISTS idx_cart_recovery_attempts_attempt_type;
DROP INDEX IF EXISTS idx_cart_recovery_attempts_abandoned_cart_id;

DROP INDEX IF EXISTS idx_abandoned_carts_customer_email;
DROP INDEX IF EXISTS idx_abandoned_carts_abandoned_at;
DROP INDEX IF EXISTS idx_abandoned_carts_status;
DROP INDEX IF EXISTS idx_abandoned_carts_user_id;
DROP INDEX IF EXISTS idx_abandoned_carts_session_id;

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS cart_snapshots;
DROP TABLE IF EXISTS cart_recovery_attempts;
DROP TABLE IF EXISTS abandoned_carts;

-- +goose StatementEnd
