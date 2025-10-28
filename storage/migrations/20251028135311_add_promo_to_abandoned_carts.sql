-- +goose Up
-- +goose StatementBegin
ALTER TABLE abandoned_carts ADD COLUMN promotion_code_id TEXT;
CREATE INDEX idx_abandoned_carts_promotion_code_id ON abandoned_carts(promotion_code_id);
-- Add foreign key constraint
-- Note: SQLite doesn't support adding foreign keys to existing tables, so this is just for documentation
-- FOREIGN KEY (promotion_code_id) REFERENCES promotion_codes(id) ON DELETE SET NULL
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN directly, need to recreate table
DROP INDEX IF EXISTS idx_abandoned_carts_promotion_code_id;

CREATE TABLE abandoned_carts_backup (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    user_id TEXT REFERENCES users(id),
    customer_email TEXT,
    customer_name TEXT,
    cart_value_cents INTEGER NOT NULL DEFAULT 0,
    item_count INTEGER NOT NULL DEFAULT 0,
    abandoned_at DATETIME NOT NULL,
    recovered_at DATETIME,
    recovery_method TEXT,
    status TEXT DEFAULT 'active',
    last_contacted_at DATETIME,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK ((session_id IS NULL) != (user_id IS NULL) OR (session_id IS NOT NULL AND user_id IS NOT NULL))
);

INSERT INTO abandoned_carts_backup
SELECT id, session_id, user_id, customer_email, customer_name, cart_value_cents,
       item_count, abandoned_at, recovered_at, recovery_method, status,
       last_contacted_at, notes, created_at, updated_at
FROM abandoned_carts;

DROP TABLE abandoned_carts;

ALTER TABLE abandoned_carts_backup RENAME TO abandoned_carts;

CREATE INDEX idx_abandoned_carts_session_id ON abandoned_carts(session_id);
CREATE INDEX idx_abandoned_carts_user_id ON abandoned_carts(user_id);
CREATE INDEX idx_abandoned_carts_status ON abandoned_carts(status);
CREATE INDEX idx_abandoned_carts_abandoned_at ON abandoned_carts(abandoned_at);
CREATE INDEX idx_abandoned_carts_customer_email ON abandoned_carts(customer_email);
-- +goose StatementEnd
