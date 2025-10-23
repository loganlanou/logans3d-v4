-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders ADD COLUMN tracking_number TEXT;
ALTER TABLE orders ADD COLUMN tracking_url TEXT;
ALTER TABLE orders ADD COLUMN carrier TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders DROP COLUMN tracking_number;
ALTER TABLE orders DROP COLUMN tracking_url;
ALTER TABLE orders DROP COLUMN carrier;
-- +goose StatementEnd
