-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS carrier_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    easypost_id TEXT NOT NULL UNIQUE,
    carrier_type TEXT NOT NULL,
    origin_zip TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed with production carrier accounts
INSERT INTO carrier_accounts (easypost_id, carrier_type, origin_zip) VALUES
    ('ca_849efc741f2d4ba89dcf1c76c004c909', 'USPS', '54727'),
    ('ca_33e897533c29449482b464b1b578e74e', 'UPS', '54701'),
    ('ca_5194870cc9584984a0eaf0a0269289ab', 'FedEx', '54701');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS carrier_accounts;
-- +goose StatementEnd
