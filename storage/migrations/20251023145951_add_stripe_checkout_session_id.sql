-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders ADD COLUMN stripe_checkout_session_id TEXT;
CREATE INDEX idx_orders_stripe_checkout_session ON orders(stripe_checkout_session_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_orders_stripe_checkout_session;
-- For SQLite, we cannot drop columns, so we would need to recreate the table
-- Since this is a new column with no critical data, we'll leave it for now
-- In production, you'd need to recreate the entire table without this column
-- +goose StatementEnd
