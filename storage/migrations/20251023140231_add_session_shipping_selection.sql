-- +goose Up
-- +goose StatementBegin
CREATE TABLE session_shipping_selection (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,

    -- Shipping rate details
    rate_id TEXT NOT NULL,
    shipment_id TEXT NOT NULL,
    carrier_name TEXT NOT NULL,
    service_name TEXT NOT NULL,
    price_cents INTEGER NOT NULL,
    delivery_days INTEGER,
    estimated_date TEXT,

    -- Cart state snapshot (for validation)
    cart_snapshot_json TEXT NOT NULL,

    -- Address info (for pre-fill)
    shipping_address_json TEXT NOT NULL,

    -- Validation state
    is_valid BOOLEAN DEFAULT TRUE,

    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_session_shipping_selection_session_id ON session_shipping_selection(session_id);
CREATE INDEX idx_session_shipping_selection_valid ON session_shipping_selection(is_valid);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_session_shipping_selection_valid;
DROP INDEX IF EXISTS idx_session_shipping_selection_session_id;
DROP TABLE IF EXISTS session_shipping_selection;
-- +goose StatementEnd
