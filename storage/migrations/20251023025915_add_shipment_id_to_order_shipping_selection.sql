-- +goose Up
-- +goose StatementBegin

-- Add shipment_id to store EasyPost shipment ID for label purchase
ALTER TABLE order_shipping_selection ADD COLUMN shipment_id TEXT;

-- Add index for faster lookups
CREATE INDEX idx_order_shipping_selection_shipment_id ON order_shipping_selection(shipment_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop index first
DROP INDEX IF EXISTS idx_order_shipping_selection_shipment_id;

-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
CREATE TABLE order_shipping_selection_new (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    candidate_box_sku TEXT NOT NULL,
    rate_id TEXT NOT NULL,
    carrier_id TEXT NOT NULL,
    service_code TEXT NOT NULL,
    service_name TEXT NOT NULL,
    quoted_shipping_amount_cents INTEGER NOT NULL,
    quoted_box_cost_cents INTEGER NOT NULL,
    quoted_total_cents INTEGER NOT NULL,
    delivery_days INTEGER,
    estimated_delivery_date TEXT,
    packing_solution_json TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(order_id)
);

-- Copy data (excluding shipment_id)
INSERT INTO order_shipping_selection_new
SELECT id, order_id, candidate_box_sku, rate_id, carrier_id, service_code, service_name,
       quoted_shipping_amount_cents, quoted_box_cost_cents, quoted_total_cents,
       delivery_days, estimated_delivery_date, packing_solution_json, created_at
FROM order_shipping_selection;

-- Drop old table and rename new one
DROP TABLE order_shipping_selection;
ALTER TABLE order_shipping_selection_new RENAME TO order_shipping_selection;

-- Recreate index
CREATE INDEX idx_order_shipping_selection_order_id ON order_shipping_selection(order_id);

-- +goose StatementEnd
