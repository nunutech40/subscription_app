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
