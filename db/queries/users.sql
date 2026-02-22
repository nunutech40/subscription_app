-- name: CreateUser :one
INSERT INTO users (email, name, password_hash, role, is_active)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserAnomalyScore :exec
UPDATE users SET anomaly_score = $2 WHERE id = $1;

-- name: SetUserActive :exec
UPDATE users SET is_active = $1 WHERE id = $2;

-- name: CountActiveSubscribers :one
SELECT COUNT(*) FROM users WHERE role = 'subscriber' AND is_active = TRUE;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: GetUserAnomalyScore :one
SELECT COALESCE(SUM(score_delta), 0)::bigint FROM anomaly_logs WHERE user_id = $1;

-- name: ListFlaggedUsers :many
SELECT u.*, COALESCE(SUM(al.score_delta), 0)::int AS total_score
FROM users u
LEFT JOIN anomaly_logs al ON al.user_id = u.id
GROUP BY u.id
HAVING COALESCE(SUM(al.score_delta), 0) >= $1
ORDER BY total_score DESC
LIMIT $2 OFFSET $3;
