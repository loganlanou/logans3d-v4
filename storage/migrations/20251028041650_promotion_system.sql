-- +goose Up
-- +goose StatementBegin
CREATE TABLE promotion_campaigns (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    discount_type TEXT NOT NULL,
    discount_value INTEGER NOT NULL,
    stripe_promotion_id TEXT,
    start_date DATETIME NOT NULL,
    end_date DATETIME,
    max_uses INTEGER,
    current_uses INTEGER DEFAULT 0,
    active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_promotion_campaigns_active ON promotion_campaigns(active);
CREATE INDEX idx_promotion_campaigns_dates ON promotion_campaigns(start_date, end_date);

CREATE TABLE promotion_codes (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    stripe_promotion_code_id TEXT UNIQUE,
    email TEXT,
    user_id TEXT,
    max_uses INTEGER DEFAULT 1,
    current_uses INTEGER DEFAULT 0,
    expires_at DATETIME,
    first_used_at DATETIME,
    last_used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (campaign_id) REFERENCES promotion_campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_promotion_codes_email ON promotion_codes(email);
CREATE INDEX idx_promotion_codes_campaign ON promotion_codes(campaign_id);
CREATE INDEX idx_promotion_codes_code ON promotion_codes(code);

CREATE TABLE marketing_contacts (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    first_name TEXT,
    last_name TEXT,
    source TEXT NOT NULL,
    opted_in INTEGER DEFAULT 1,
    promotion_code_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (promotion_code_id) REFERENCES promotion_codes(id) ON DELETE SET NULL
);

CREATE INDEX idx_marketing_contacts_email ON marketing_contacts(email);
CREATE INDEX idx_marketing_contacts_opted_in ON marketing_contacts(opted_in);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_marketing_contacts_opted_in;
DROP INDEX IF EXISTS idx_marketing_contacts_email;
DROP TABLE IF EXISTS marketing_contacts;

DROP INDEX IF EXISTS idx_promotion_codes_code;
DROP INDEX IF EXISTS idx_promotion_codes_campaign;
DROP INDEX IF EXISTS idx_promotion_codes_email;
DROP TABLE IF EXISTS promotion_codes;

DROP INDEX IF EXISTS idx_promotion_campaigns_dates;
DROP INDEX IF EXISTS idx_promotion_campaigns_active;
DROP TABLE IF EXISTS promotion_campaigns;
-- +goose StatementEnd
