-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN disclaimer TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite 3.35.0+ supports DROP COLUMN
ALTER TABLE products DROP COLUMN disclaimer;
-- +goose StatementEnd
