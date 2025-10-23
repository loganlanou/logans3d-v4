-- +goose Up
-- +goose StatementBegin
UPDATE shipping_config
SET config_json = json_set(
    config_json,
    '$.packing.packing_materials.handling_fee_per_box_usd',
    1.50
)
WHERE id = 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE shipping_config
SET config_json = json_remove(
    config_json,
    '$.packing.packing_materials.handling_fee_per_box_usd'
)
WHERE id = 1;
-- +goose StatementEnd
