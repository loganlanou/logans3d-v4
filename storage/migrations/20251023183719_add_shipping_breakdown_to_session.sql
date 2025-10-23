-- +goose Up
-- +goose StatementBegin

-- Add breakdown columns to session_shipping_selection
ALTER TABLE session_shipping_selection ADD COLUMN shipping_amount_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE session_shipping_selection ADD COLUMN box_cost_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE session_shipping_selection ADD COLUMN handling_cost_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE session_shipping_selection ADD COLUMN box_sku TEXT NOT NULL DEFAULT 'UNKNOWN';

-- Migrate existing data: set shipping_amount_cents to current price_cents
UPDATE session_shipping_selection SET shipping_amount_cents = price_cents WHERE shipping_amount_cents = 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
CREATE TABLE session_shipping_selection_new (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,
    rate_id TEXT NOT NULL,
    shipment_id TEXT NOT NULL,
    carrier_name TEXT NOT NULL,
    service_name TEXT NOT NULL,
    price_cents INTEGER NOT NULL,
    delivery_days INTEGER,
    estimated_date TEXT,
    cart_snapshot_json TEXT NOT NULL,
    shipping_address_json TEXT NOT NULL,
    is_valid BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Copy data back (excluding new columns)
INSERT INTO session_shipping_selection_new
SELECT id, session_id, rate_id, shipment_id, carrier_name, service_name, price_cents,
       delivery_days, estimated_date, cart_snapshot_json, shipping_address_json,
       is_valid, created_at, updated_at
FROM session_shipping_selection;

-- Drop old table and rename
DROP TABLE session_shipping_selection;
ALTER TABLE session_shipping_selection_new RENAME TO session_shipping_selection;

-- Recreate indexes
CREATE INDEX idx_session_shipping_selection_session_id ON session_shipping_selection(session_id);
CREATE INDEX idx_session_shipping_selection_shipment_id ON session_shipping_selection(shipment_id);

-- +goose StatementEnd
