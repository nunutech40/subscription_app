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
INSERT INTO guest_logins (guest_code_id, email, login_count, last_login_at, referral_source, ip_address)
VALUES ($1, $2, 1, now(), $3, $4)
ON CONFLICT (guest_code_id, email)
DO UPDATE SET login_count = guest_logins.login_count + 1, last_login_at = now(),
  referral_source = CASE WHEN $3 != '' THEN $3 ELSE guest_logins.referral_source END,
  ip_address = COALESCE($4, guest_logins.ip_address)
RETURNING *;

-- name: ListGuestLoginsByCode :many
SELECT * FROM guest_logins WHERE guest_code_id = $1 ORDER BY last_login_at DESC;

-- name: CountGuestCodeLogins :one
SELECT COUNT(*) FROM guest_logins WHERE guest_code_id = $1;

-- name: SumGuestCodeLogins :one
SELECT COALESCE(SUM(login_count), 0)::bigint FROM guest_logins WHERE guest_code_id = $1;

-- name: ListAllAudience :many
SELECT
  gl.email,
  'guest'::text AS user_type,
  gc.code AS guest_code,
  COALESCE(gl.referral_source, '') AS referral_source,
  COALESCE(gl.ip_address::text, '') AS ip_address,
  gl.login_count::bigint AS total_logins,
  0::bigint AS amount_paid,
  gl.last_login_at AS last_active,
  gl.created_at
FROM guest_logins gl
JOIN guest_codes gc ON gc.id = gl.guest_code_id
UNION ALL
SELECT
  u.email,
  u.role::text AS user_type,
  ''::text AS guest_code,
  ''::text AS referral_source,
  COALESCE((SELECT s3.ip_at_login::text FROM sessions s3 WHERE s3.user_id = u.id AND s3.ip_at_login IS NOT NULL ORDER BY s3.created_at DESC LIMIT 1), '') AS ip_address,
  COUNT(s2.id)::bigint AS total_logins,
  COALESCE(SUM(sub.amount_paid_idr), 0)::bigint AS amount_paid,
  COALESCE(MAX(s2.created_at), u.created_at) AS last_active,
  u.created_at
FROM users u
LEFT JOIN sessions s2 ON s2.user_id = u.id
LEFT JOIN subscriptions sub ON sub.user_id = u.id AND sub.status = 'active'
GROUP BY u.id, u.email, u.role, u.created_at
ORDER BY last_active DESC
LIMIT $1 OFFSET $2;
