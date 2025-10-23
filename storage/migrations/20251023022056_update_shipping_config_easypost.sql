-- +goose Up
-- +goose StatementBegin

-- Update shipping configuration to use EasyPost instead of ShipStation
-- This migration preserves all existing settings and only updates:
-- 1. Removes "shipstation_api_version" field
-- 2. Adds "provider": "easypost" field
-- 3. Updates phone numbers to correct number
UPDATE shipping_config
SET config_json = json_set(
    json_remove(config_json, '$.shipping.shipstation_api_version'),
    '$.shipping.provider', 'easypost',
    '$.shipping.ship_from.phone', '715-703-3768',
    '$.shipping.ship_from_usps.phone', '715-703-3768',
    '$.shipping.ship_from_other.phone', '715-703-3768'
),
updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Revert to ShipStation configuration
UPDATE shipping_config
SET config_json = json_set(
    json_remove(config_json, '$.shipping.provider'),
    '$.shipping.shipstation_api_version', 'v2',
    '$.shipping.ship_from.phone', '715-703-3768',
    '$.shipping.ship_from_usps.phone', '715-703-3768',
    '$.shipping.ship_from_other.phone', '715-703-3768'
),
updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- +goose StatementEnd
