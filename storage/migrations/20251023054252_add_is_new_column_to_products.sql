-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN is_new BOOLEAN DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products DROP COLUMN is_new;
-- +goose StatementEnd
