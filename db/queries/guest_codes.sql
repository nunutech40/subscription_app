-- name: CreateGuestCode :one
INSERT INTO guest_codes (code, product_id, label, max_logins_per_email, expires_at, generated_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetGuestCodeByCode :one
SELECT * FROM guest_codes WHERE code = $1 AND is_active = TRUE;

-- name: GetGuestCodeByID :one
SELECT * FROM guest_codes WHERE id = $1;

-- name: ListGuestCodes :many
SELECT * FROM guest_codes ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: DeactivateGuestCode :exec
UPDATE guest_codes SET is_active = FALSE WHERE id = $1;

-- name: DeleteGuestCode :exec
DELETE FROM guest_codes WHERE id = $1;

-- name: GetGuestLogin :one
SELECT * FROM guest_logins WHERE guest_code_id = $1 AND email = $2;

-- name: UpsertGuestLogin :one
INSERT INTO guest_logins (guest_code_id, email, login_count, last_login_at, referral_source)
VALUES ($1, $2, 1, now(), $3)
ON CONFLICT (guest_code_id, email)
DO UPDATE SET login_count = guest_logins.login_count + 1, last_login_at = now(),
  referral_source = CASE WHEN $3 != '' THEN $3 ELSE guest_logins.referral_source END
RETURNING *;

-- name: ListGuestLoginsByCode :many
SELECT * FROM guest_logins WHERE guest_code_id = $1 ORDER BY last_login_at DESC;

-- name: CountGuestCodeLogins :one
SELECT COUNT(*) FROM guest_logins WHERE guest_code_id = $1;
