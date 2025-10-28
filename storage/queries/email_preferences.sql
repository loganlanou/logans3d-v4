-- name: CreateEmailPreferences :one
INSERT INTO email_preferences (
    id,
    user_id,
    email,
    transactional,
    abandoned_cart,
    promotional,
    newsletter,
    product_updates,
    unsubscribe_token
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetEmailPreferencesByID :one
SELECT * FROM email_preferences
WHERE id = ?;

-- name: GetEmailPreferencesByUserID :one
SELECT * FROM email_preferences
WHERE user_id = ?
LIMIT 1;

-- name: GetEmailPreferencesByEmail :one
SELECT * FROM email_preferences
WHERE email = ?
LIMIT 1;

-- name: GetEmailPreferencesByUnsubscribeToken :one
SELECT * FROM email_preferences
WHERE unsubscribe_token = ?;

-- name: UpdateEmailPreferences :exec
UPDATE email_preferences
SET
    transactional = ?,
    abandoned_cart = ?,
    promotional = ?,
    newsletter = ?,
    product_updates = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UnsubscribeFromAllMarketing :exec
UPDATE email_preferences
SET
    abandoned_cart = 0,
    promotional = 0,
    newsletter = 0,
    product_updates = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE unsubscribe_token = ?;

-- name: UnsubscribeFromEmailType :exec
UPDATE email_preferences
SET
    abandoned_cart = CASE WHEN ? = 'abandoned_cart' THEN 0 ELSE abandoned_cart END,
    promotional = CASE WHEN ? = 'promotional' THEN 0 ELSE promotional END,
    newsletter = CASE WHEN ? = 'newsletter' THEN 0 ELSE newsletter END,
    product_updates = CASE WHEN ? = 'product_updates' THEN 0 ELSE product_updates END,
    updated_at = CURRENT_TIMESTAMP
WHERE unsubscribe_token = ?;

-- name: CheckEmailPreference :one
SELECT
    CASE
        WHEN ? = 'transactional' THEN transactional
        WHEN ? = 'abandoned_cart' THEN abandoned_cart
        WHEN ? = 'promotional' THEN promotional
        WHEN ? = 'newsletter' THEN newsletter
        WHEN ? = 'product_updates' THEN product_updates
        ELSE 0
    END as preference_value
FROM email_preferences
WHERE email = ?;

-- name: GetAllOptedInEmails :many
SELECT email FROM email_preferences
WHERE
    (? = 'abandoned_cart' AND abandoned_cart = 1) OR
    (? = 'promotional' AND promotional = 1) OR
    (? = 'newsletter' AND newsletter = 1) OR
    (? = 'product_updates' AND product_updates = 1);

-- name: CountOptedInByType :one
SELECT COUNT(*) FROM email_preferences
WHERE
    (? = 'abandoned_cart' AND abandoned_cart = 1) OR
    (? = 'promotional' AND promotional = 1) OR
    (? = 'newsletter' AND newsletter = 1) OR
    (? = 'product_updates' AND product_updates = 1);

-- name: DeleteEmailPreferences :exec
DELETE FROM email_preferences
WHERE id = ?;

-- name: GetOrCreateEmailPreferences :one
INSERT INTO email_preferences (
    id,
    user_id,
    email,
    unsubscribe_token
) VALUES (?, ?, ?, ?)
ON CONFLICT(user_id, email) DO UPDATE SET
    updated_at = CURRENT_TIMESTAMP
RETURNING *;
