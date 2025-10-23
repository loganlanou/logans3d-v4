-- +goose Up
-- +goose StatementBegin

-- Add EasyPost shipment ID to link orders back to EasyPost shipments created during checkout
ALTER TABLE orders ADD COLUMN easypost_shipment_id TEXT;

-- Add EasyPost label URL to store the purchased shipping label PDF
ALTER TABLE orders ADD COLUMN easypost_label_url TEXT;

-- Add indexes for faster lookups
CREATE INDEX idx_orders_easypost_shipment_id ON orders(easypost_shipment_id);
CREATE INDEX idx_orders_easypost_label_url ON orders(easypost_label_url);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes first
DROP INDEX IF EXISTS idx_orders_easypost_label_url;
DROP INDEX IF EXISTS idx_orders_easypost_shipment_id;

-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
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

-- Copy data (excluding easypost fields)
INSERT INTO orders_new
SELECT id, user_id, customer_name, customer_email, customer_phone,
       shipping_address_line1, shipping_address_line2, shipping_city, shipping_state,
       shipping_postal_code, shipping_country,
       subtotal_cents, tax_cents, shipping_cents, total_cents,
       status, notes,
       stripe_payment_intent_id, stripe_customer_id, stripe_checkout_session_id,
       tracking_number, tracking_url, carrier,
       created_at, updated_at
FROM orders;

-- Drop old table and rename new one
DROP TABLE orders;
ALTER TABLE orders_new RENAME TO orders;

-- +goose StatementEnd
