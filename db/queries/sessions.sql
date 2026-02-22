-- name: CreateSession :one
INSERT INTO sessions (
  user_id, guest_code_id, guest_email,
  refresh_token_hash, device_fingerprint,
  ip_at_login, user_agent, country_code, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM sessions WHERE id = $1 AND is_active = TRUE;

-- name: GetSessionByRefreshToken :one
SELECT * FROM sessions WHERE refresh_token_hash = $1 AND is_active = TRUE;

-- name: GetActiveSessionByUserID :one
SELECT * FROM sessions WHERE user_id = $1 AND is_active = TRUE LIMIT 1;

-- name: RevokeSession :exec
UPDATE sessions
SET is_active = FALSE, revoked_at = now(), revoke_reason = $2
WHERE id = $1;

-- name: RevokeAllUserSessions :exec
UPDATE sessions
SET is_active = FALSE, revoked_at = now(), revoke_reason = 'new_login'
WHERE user_id = $1 AND is_active = TRUE;

-- name: CountActiveGuestSessions :one
SELECT COUNT(*) FROM sessions
WHERE guest_code_id IS NOT NULL AND is_active = TRUE AND expires_at > now();

-- name: CleanExpiredSessions :exec
UPDATE sessions
SET is_active = FALSE, revoked_at = now(), revoke_reason = 'expired'
WHERE is_active = TRUE AND expires_at < now();

-- name: ListSessionsByUser :many
SELECT * FROM sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 20;
