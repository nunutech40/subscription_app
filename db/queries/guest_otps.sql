-- name: CreateGuestOTP :one
INSERT INTO guest_otps (email, guest_code_id, otp_code, expires_at, ip)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPendingOTP :one
SELECT * FROM guest_otps
WHERE email = $1
  AND guest_code_id = $2
  AND otp_code = $3
  AND verified = FALSE
  AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkOTPVerified :exec
UPDATE guest_otps SET verified = TRUE WHERE id = $1;

-- name: CleanExpiredOTPs :exec
DELETE FROM guest_otps WHERE expires_at < now();

-- name: CountRecentOTPs :one
SELECT COUNT(*) FROM guest_otps
WHERE email = $1
  AND created_at > now() - interval '10 minutes';
