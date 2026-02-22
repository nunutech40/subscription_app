-- name: CreateAnomalyLog :one
INSERT INTO anomaly_logs (user_id, event, score_delta, detail)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListAnomalyLogsByUser :many
SELECT * FROM anomaly_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: ListRecentAnomalies :many
SELECT al.*, u.email, u.name, u.anomaly_score
FROM anomaly_logs al
JOIN users u ON u.id = al.user_id
ORDER BY al.created_at DESC LIMIT $1 OFFSET $2;

-- name: CreateAccessLog :exec
INSERT INTO access_logs (user_id, session_id, ip, endpoint, method, status_code)
VALUES ($1, $2, $3, $4, $5, $6);
