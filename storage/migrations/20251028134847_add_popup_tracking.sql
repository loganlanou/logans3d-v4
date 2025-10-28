-- +goose Up
-- +goose StatementBegin
ALTER TABLE marketing_contacts ADD COLUMN popup_shown_at DATETIME;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN directly, need to recreate table
CREATE TABLE marketing_contacts_backup (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    first_name TEXT,
    last_name TEXT,
    source TEXT NOT NULL,
    opted_in INTEGER NOT NULL DEFAULT 1,
    promotion_code_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (promotion_code_id) REFERENCES promotion_codes(id) ON DELETE SET NULL
);

INSERT INTO marketing_contacts_backup
SELECT id, email, first_name, last_name, source, opted_in, promotion_code_id, created_at, updated_at
FROM marketing_contacts;

DROP TABLE marketing_contacts;

ALTER TABLE marketing_contacts_backup RENAME TO marketing_contacts;

CREATE INDEX idx_marketing_contacts_email ON marketing_contacts(email);
CREATE INDEX idx_marketing_contacts_promotion_code_id ON marketing_contacts(promotion_code_id);
-- +goose StatementEnd
