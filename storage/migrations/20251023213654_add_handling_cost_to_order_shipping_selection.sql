-- +goose Up
-- +goose StatementBegin

ALTER TABLE order_shipping_selection ADD COLUMN quoted_handling_cost_cents INTEGER NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
CREATE TABLE order_shipping_selection_new (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
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
    shipment_id TEXT,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

INSERT INTO order_shipping_selection_new
SELECT id, order_id, candidate_box_sku, rate_id, carrier_id, service_code, service_name,
       quoted_shipping_amount_cents, quoted_box_cost_cents, quoted_total_cents,
       delivery_days, estimated_delivery_date, packing_solution_json, created_at, shipment_id
FROM order_shipping_selection;

DROP TABLE order_shipping_selection;
ALTER TABLE order_shipping_selection_new RENAME TO order_shipping_selection;

-- +goose StatementEnd
