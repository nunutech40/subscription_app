-- name: InsertAuditLog :exec
INSERT INTO admin_audit_logs (admin_email, action, resource, resource_id, detail, ip)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAuditLogs :many
SELECT * FROM admin_audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT count(*) FROM admin_audit_logs;

-- name: ListAuditLogsByAction :many
SELECT * FROM admin_audit_logs
WHERE action = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
