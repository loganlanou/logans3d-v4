-- +goose Up

CREATE TABLE gift_certificates (
    id TEXT PRIMARY KEY,
    amount REAL NOT NULL,
    reference TEXT,
    issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    redeemed_at TIMESTAMP,
    redeemed_by_user_id TEXT,
    redeemer_name TEXT,
    redemption_notes TEXT,
    created_by_user_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    image_png_path TEXT,
    image_pdf_path TEXT,
    voided_at TIMESTAMP,
    voided_by_user_id TEXT,
    void_reason TEXT,
    FOREIGN KEY (redeemed_by_user_id) REFERENCES users(id),
    FOREIGN KEY (created_by_user_id) REFERENCES users(id),
    FOREIGN KEY (voided_by_user_id) REFERENCES users(id)
);

CREATE INDEX idx_gift_certificates_redeemed ON gift_certificates(redeemed_at);
CREATE INDEX idx_gift_certificates_issued ON gift_certificates(issued_at);
CREATE INDEX idx_gift_certificates_voided ON gift_certificates(voided_at);

-- +goose Down
DROP INDEX IF EXISTS idx_gift_certificates_voided;
DROP INDEX IF EXISTS idx_gift_certificates_issued;
DROP INDEX IF EXISTS idx_gift_certificates_redeemed;
DROP TABLE IF EXISTS gift_certificates;
