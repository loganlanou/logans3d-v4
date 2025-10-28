-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders ADD COLUMN original_subtotal_cents INTEGER DEFAULT 0;
ALTER TABLE orders ADD COLUMN discount_cents INTEGER DEFAULT 0;
ALTER TABLE orders ADD COLUMN promotion_code TEXT;
ALTER TABLE orders ADD COLUMN promotion_code_id TEXT;

-- Create index on promotion_code_id for efficient lookups
CREATE INDEX idx_orders_promotion_code_id ON orders(promotion_code_id);

-- For existing orders, copy current subtotal to original_subtotal_cents
-- (since they don't have discounts tracked yet)
UPDATE orders SET original_subtotal_cents = subtotal_cents WHERE original_subtotal_cents = 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
-- First, drop the promotion_code_id index
DROP INDEX IF EXISTS idx_orders_promotion_code_id;

-- Create new orders table without discount fields
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
    easypost_shipment_id TEXT,
    easypost_label_url TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy data from old table to new table (excluding discount fields)
INSERT INTO orders_new (
    id, user_id, customer_name, customer_email, customer_phone,
    shipping_address_line1, shipping_address_line2, shipping_city,
    shipping_state, shipping_postal_code, shipping_country,
    subtotal_cents, tax_cents, shipping_cents, total_cents, status,
    notes, stripe_payment_intent_id, stripe_customer_id,
    stripe_checkout_session_id, tracking_number, tracking_url,
    carrier, created_at, updated_at, easypost_shipment_id,
    easypost_label_url
)
SELECT
    id, user_id, customer_name, customer_email, customer_phone,
    shipping_address_line1, shipping_address_line2, shipping_city,
    shipping_state, shipping_postal_code, shipping_country,
    subtotal_cents, tax_cents, shipping_cents, total_cents, status,
    notes, stripe_payment_intent_id, stripe_customer_id,
    stripe_checkout_session_id, tracking_number, tracking_url,
    carrier, created_at, updated_at, easypost_shipment_id,
    easypost_label_url
FROM orders;

-- Drop old table and rename new table
DROP TABLE orders;
ALTER TABLE orders_new RENAME TO orders;

-- Recreate indexes
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_stripe_payment_intent ON orders(stripe_payment_intent_id);
CREATE INDEX idx_orders_stripe_checkout_session ON orders(stripe_checkout_session_id);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_orders_easypost_shipment_id ON orders(easypost_shipment_id);
CREATE INDEX idx_orders_easypost_label_url ON orders(easypost_label_url);
-- +goose StatementEnd
