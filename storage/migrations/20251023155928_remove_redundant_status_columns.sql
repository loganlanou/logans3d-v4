-- +goose Up
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN directly for multiple columns
-- We need to recreate the table without those columns

-- Create new table without fulfillment_status and payment_status
CREATE TABLE orders_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    customer_name TEXT NOT NULL,
    customer_email TEXT NOT NULL,
    customer_phone TEXT,
    shipping_address_line1 TEXT NOT NULL,
    shipping_address_line2 TEXT,
    shipping_city TEXT NOT NULL,
    shipping_state TEXT NOT NULL,
    shipping_postal_code TEXT NOT NULL,
    shipping_country TEXT NOT NULL DEFAULT 'US',
    subtotal_cents INTEGER NOT NULL,
    tax_cents INTEGER NOT NULL DEFAULT 0,
    shipping_cents INTEGER NOT NULL DEFAULT 0,
    total_cents INTEGER NOT NULL,
    status TEXT DEFAULT 'received',
    notes TEXT,
    stripe_payment_intent_id TEXT,
    stripe_customer_id TEXT,
    stripe_checkout_session_id TEXT,
    tracking_number TEXT,
    tracking_url TEXT,
    carrier TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy data from old table, converting statuses
INSERT INTO orders_new (
    id, user_id, customer_name, customer_email, customer_phone,
    shipping_address_line1, shipping_address_line2, shipping_city,
    shipping_state, shipping_postal_code, shipping_country,
    subtotal_cents, tax_cents, shipping_cents, total_cents,
    status, notes,
    stripe_payment_intent_id, stripe_customer_id, stripe_checkout_session_id,
    tracking_number, tracking_url, carrier,
    created_at, updated_at
)
SELECT
    id, user_id, customer_name, customer_email, customer_phone,
    shipping_address_line1, shipping_address_line2, shipping_city,
    shipping_state, shipping_postal_code, shipping_country,
    subtotal_cents, tax_cents, shipping_cents, total_cents,
    CASE
        WHEN status = 'processing' THEN 'received'
        ELSE status
    END as status,
    notes,
    stripe_payment_intent_id, stripe_customer_id, stripe_checkout_session_id,
    tracking_number, tracking_url, carrier,
    created_at, updated_at
FROM orders;

-- Drop old table
DROP TABLE orders;

-- Rename new table
ALTER TABLE orders_new RENAME TO orders;

-- Recreate indexes
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_stripe_payment_intent ON orders(stripe_payment_intent_id);
CREATE INDEX idx_orders_stripe_checkout_session ON orders(stripe_checkout_session_id);
CREATE INDEX idx_orders_created_at ON orders(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Recreate old table structure with fulfillment_status and payment_status
CREATE TABLE orders_old (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    customer_name TEXT NOT NULL,
    customer_email TEXT NOT NULL,
    customer_phone TEXT,
    shipping_address_line1 TEXT NOT NULL,
    shipping_address_line2 TEXT,
    shipping_city TEXT NOT NULL,
    shipping_state TEXT NOT NULL,
    shipping_postal_code TEXT NOT NULL,
    shipping_country TEXT NOT NULL DEFAULT 'US',
    subtotal_cents INTEGER NOT NULL,
    tax_cents INTEGER NOT NULL DEFAULT 0,
    shipping_cents INTEGER NOT NULL DEFAULT 0,
    total_cents INTEGER NOT NULL,
    status TEXT DEFAULT 'pending',
    fulfillment_status TEXT DEFAULT 'unfulfilled',
    payment_status TEXT DEFAULT 'unpaid',
    notes TEXT,
    stripe_payment_intent_id TEXT,
    stripe_customer_id TEXT,
    stripe_checkout_session_id TEXT,
    tracking_number TEXT,
    tracking_url TEXT,
    carrier TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy data back, converting statuses
INSERT INTO orders_old (
    id, user_id, customer_name, customer_email, customer_phone,
    shipping_address_line1, shipping_address_line2, shipping_city,
    shipping_state, shipping_postal_code, shipping_country,
    subtotal_cents, tax_cents, shipping_cents, total_cents,
    status, fulfillment_status, payment_status, notes,
    stripe_payment_intent_id, stripe_customer_id, stripe_checkout_session_id,
    tracking_number, tracking_url, carrier,
    created_at, updated_at
)
SELECT
    id, user_id, customer_name, customer_email, customer_phone,
    shipping_address_line1, shipping_address_line2, shipping_city,
    shipping_state, shipping_postal_code, shipping_country,
    subtotal_cents, tax_cents, shipping_cents, total_cents,
    CASE
        WHEN status = 'received' THEN 'processing'
        ELSE status
    END as status,
    'unfulfilled' as fulfillment_status,
    'paid' as payment_status,
    notes,
    stripe_payment_intent_id, stripe_customer_id, stripe_checkout_session_id,
    tracking_number, tracking_url, carrier,
    created_at, updated_at
FROM orders;

-- Drop new table
DROP TABLE orders;

-- Rename old table back
ALTER TABLE orders_old RENAME TO orders;

-- Recreate indexes
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_stripe_payment_intent ON orders(stripe_payment_intent_id);
CREATE INDEX idx_orders_stripe_checkout_session ON orders(stripe_checkout_session_id);
CREATE INDEX idx_orders_created_at ON orders(created_at);
-- +goose StatementEnd
