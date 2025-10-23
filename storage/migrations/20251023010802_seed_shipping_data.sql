-- +goose Up
-- +goose StatementBegin

-- Clear existing data from box_catalog
DELETE FROM box_catalog;

-- Insert box catalog data
INSERT INTO box_catalog (id, sku, name, length_inches, width_inches, height_inches, box_weight_oz, unit_cost_usd, is_active) VALUES
('box-cxbss654', 'CXBSS654', '6x5x4', 6.0, 5.0, 4.0, 2.5, 0.40, 1),
('box-cxbss18', 'CXBSS18', '8x6x4', 8.0, 6.0, 4.0, 3.5, 0.38, 1),
('box-cxbss21', 'CXBSS21', '9x7x5', 9.0, 7.0, 5.0, 4.0, 0.51, 1),
('box-cxbss24', 'CXBSS24', '10x8x6', 10.0, 8.0, 6.0, 6.0, 0.54, 1),
('box-cxbsm1294', 'CXBSM1294', '12x9x4', 12.0, 9.0, 4.0, 6.0, 0.62, 1),
('box-cxbsm12124', 'CXBSM12124', '12x12x4', 12.0, 12.0, 4.0, 7.0, 0.72, 1);

-- Insert or update shipping configuration
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
    },
    "item_weights": {
      "small": {
        "min_grams": 43,
        "max_grams": 71,
        "avg_grams": 57,
        "avg_oz": 2.0
      },
      "medium": {
        "min_grams": 143,
        "max_grams": 286,
        "avg_grams": 150,
        "avg_oz": 5.3
      },
      "large": {
        "min_grams": 357,
        "max_grams": 786,
        "avg_grams": 571,
        "avg_oz": 20.1
      },
      "xlarge": {
        "min_grams": 929,
        "max_grams": 1071,
        "avg_grams": 1000,
        "avg_oz": 35.3
      }
    },
    "packing_materials": {
      "bubble_wrap_per_item_oz": 0.1,
      "packing_paper_per_box_oz": 0.5,
      "tape_and_labels_per_box_oz": 0.3,
      "air_pillows_per_box_oz": 0.3
    }
  },
  "boxes": [
    {
      "sku": "CXBSS654",
      "name": "6x5x4",
      "L": 6,
      "W": 5,
      "H": 4,
      "box_weight_oz": 2.5,
      "unit_cost_usd": 0.40
    },
    {
      "sku": "CXBSS18",
      "name": "8x6x4",
      "L": 8,
      "W": 6,
      "H": 4,
      "box_weight_oz": 3.5,
      "unit_cost_usd": 0.38
    },
    {
      "sku": "CXBSS21",
      "name": "9x7x5",
      "L": 9,
      "W": 7,
      "H": 5,
      "box_weight_oz": 4.0,
      "unit_cost_usd": 0.51
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
      "sku": "CXBSM12124",
      "name": "12x12x4",
      "L": 12,
      "W": 12,
      "H": 4,
      "box_weight_oz": 7.0,
      "unit_cost_usd": 0.72
    }
  ],
  "shipping": {
    "shipstation_api_version": "v2",
    "api_key_secret_storage": "env",
    "ship_from_usps": {
      "name": "Creswood Corners",
      "phone": "715-XXX-XXXX",
      "address_line1": "YOUR ADDRESS",
      "city_locality": "Cadott",
      "state_province": "WI",
      "postal_code": "54727",
      "country_code": "US",
      "address_residential_indicator": "no"
    },
    "ship_from_other": {
      "name": "Creswood Corners",
      "phone": "715-XXX-XXXX",
      "address_line1": "YOUR ADDRESS",
      "city_locality": "Eau Claire",
      "state_province": "WI",
      "postal_code": "54701",
      "country_code": "US",
      "address_residential_indicator": "no"
    },
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
}')
ON CONFLICT(id) DO UPDATE SET
    config_json = excluded.config_json,
    updated_at = CURRENT_TIMESTAMP;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove seeded box data
DELETE FROM box_catalog WHERE sku IN ('CXBSS654', 'CXBSS18', 'CXBSS21', 'CXBSS24', 'CXBSM1294', 'CXBSM12124');

-- Clear shipping config (optional - could also restore previous config if needed)
DELETE FROM shipping_config WHERE id = 1;

-- +goose StatementEnd
