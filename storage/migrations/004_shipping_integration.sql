-- +goose Up
-- +goose StatementBegin

-- Shipping configuration stored as JSON blob for admin editing
CREATE TABLE shipping_config (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- Ensure only one config row
    config_json TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Box catalog (can be synced from config or managed separately)
CREATE TABLE box_catalog (
    id TEXT PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    length_inches REAL NOT NULL,
    width_inches REAL NOT NULL,
    height_inches REAL NOT NULL,
    box_weight_oz REAL NOT NULL DEFAULT 0,
    unit_cost_usd REAL NOT NULL DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Order shipping selection - stores customer's chosen shipping option
CREATE TABLE order_shipping_selection (
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
    packing_solution_json TEXT, -- JSON blob of PackingSolution
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(order_id)
);

-- Labels created for orders
CREATE TABLE shipping_labels (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    label_id TEXT UNIQUE NOT NULL, -- ShipStation label ID
    tracking_number TEXT NOT NULL,
    carrier_id TEXT NOT NULL,
    service_code TEXT NOT NULL,
    shipping_amount_cents INTEGER NOT NULL,
    label_pdf_url TEXT,
    label_pdf_path TEXT, -- Local file path if downloaded
    status TEXT NOT NULL DEFAULT 'purchased', -- purchased, voided
    shipstation_created_at DATETIME,
    voided_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(order_id) -- One label per order for now
);

-- Product categorization for shipping (extends existing products table)
-- This helps determine which "equivalence" category each product falls into
ALTER TABLE products ADD COLUMN shipping_category TEXT DEFAULT 'small'
    CHECK (shipping_category IN ('small', 'medium', 'large', 'xlarge'));

-- Indexes for performance
CREATE INDEX idx_order_shipping_selection_order_id ON order_shipping_selection(order_id);
CREATE INDEX idx_shipping_labels_order_id ON shipping_labels(order_id);
CREATE INDEX idx_shipping_labels_tracking_number ON shipping_labels(tracking_number);
CREATE INDEX idx_shipping_labels_status ON shipping_labels(status);
CREATE INDEX idx_box_catalog_sku ON box_catalog(sku);
CREATE INDEX idx_box_catalog_is_active ON box_catalog(is_active);
CREATE INDEX idx_products_shipping_category ON products(shipping_category);

-- Insert default shipping configuration
INSERT INTO shipping_config (id, config_json) VALUES (1, '{
  "packing": {
    "unit_volume_in3": 27,
    "unit_weight_oz": 2.0,
    "equivalences": {
      "small": 1,
      "medium": 3,
      "large": 6,
      "xlarge": 18
    },
    "fill_ratio": 0.80,
    "dimension_guard_in": {
      "small": { "L": 4, "W": 4, "H": 4 },
      "medium": { "L": 8, "W": 5, "H": 5 },
      "large": { "L": 20, "W": 10, "H": 6 },
      "xlarge": { "L": 24, "W": 12, "H": 10 }
    }
  },
  "boxes": [
    {
      "sku": "CXBSS21",
      "name": "8x6x4",
      "L": 8,
      "W": 6,
      "H": 4,
      "box_weight_oz": 4.0,
      "unit_cost_usd": 0.38
    },
    {
      "sku": "CXBSS24",
      "name": "10x8x6",
      "L": 10,
      "W": 8,
      "H": 6,
      "box_weight_oz": 6.0,
      "unit_cost_usd": 0.54
    },
    {
      "sku": "CXBSM1294",
      "name": "12x9x4",
      "L": 12,
      "W": 9,
      "H": 4,
      "box_weight_oz": 6.0,
      "unit_cost_usd": 0.62
    },
    {
      "sku": "MD12126",
      "name": "12x12x6 (MD)",
      "L": 12,
      "W": 12,
      "H": 6,
      "box_weight_oz": 8.0,
      "unit_cost_usd": 0.70
    }
  ],
  "shipping": {
    "shipstation_api_version": "v2",
    "api_key_secret_storage": "env",
    "ship_from": {
      "name": "Creswood Corners",
      "phone": "715-XXX-XXXX",
      "address_line1": "YOUR ADDRESS",
      "city_locality": "Cadott",
      "state_province": "WI",
      "postal_code": "54727",
      "country_code": "US",
      "address_residential_indicator": "no"
    },
    "dim_divisors": {
      "usps": 166,
      "ups": 139,
      "fedex": 139
    },
    "rate_preferences": {
      "present_top_n": 3,
      "sort": "price_then_days"
    },
    "labels": {
      "format": "pdf"
    }
  }
}');

-- Insert default box catalog from config
INSERT INTO box_catalog (id, sku, name, length_inches, width_inches, height_inches, box_weight_oz, unit_cost_usd) VALUES
    ('box_cxbss21', 'CXBSS21', '8x6x4', 8, 6, 4, 4.0, 0.38),
    ('box_cxbss24', 'CXBSS24', '10x8x6', 10, 8, 6, 6.0, 0.54),
    ('box_cxbsm1294', 'CXBSM1294', '12x9x4', 12, 9, 4, 6.0, 0.62),
    ('box_md12126', 'MD12126', '12x12x6 (MD)', 12, 12, 6, 8.0, 0.70);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE products DROP COLUMN shipping_category;

DROP INDEX IF EXISTS idx_products_shipping_category;
DROP INDEX IF EXISTS idx_box_catalog_is_active;
DROP INDEX IF EXISTS idx_box_catalog_sku;
DROP INDEX IF EXISTS idx_shipping_labels_status;
DROP INDEX IF EXISTS idx_shipping_labels_tracking_number;
DROP INDEX IF EXISTS idx_shipping_labels_order_id;
DROP INDEX IF EXISTS idx_order_shipping_selection_order_id;

DROP TABLE IF EXISTS shipping_labels;
DROP TABLE IF EXISTS order_shipping_selection;
DROP TABLE IF EXISTS box_catalog;
DROP TABLE IF EXISTS shipping_config;

-- +goose StatementEnd