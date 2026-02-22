-- name: GetConfig :one
SELECT * FROM system_config WHERE key = $1;

-- name: UpdateConfig :exec
UPDATE system_config
SET value = $2, updated_at = now(), updated_by = $3
WHERE key = $1;

-- name: ListConfigs :many
SELECT * FROM system_config ORDER BY key;

-- name: UpsertConfig :exec
INSERT INTO system_config (key, value, description)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = now();
