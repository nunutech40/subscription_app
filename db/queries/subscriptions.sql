-- name: CreateSubscription :one
INSERT INTO subscriptions (
  user_id, product_id, plan_id, segment,
  xendit_invoice_id, amount_paid_idr, status, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions WHERE id = $1;

-- name: GetActiveSubscription :one
SELECT * FROM subscriptions
WHERE user_id = $1 AND product_id = $2 AND status = 'active' AND expires_at > now()
LIMIT 1;

-- name: ListUserSubscriptions :many
SELECT * FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions
SET status = $2, paid_at = $3, starts_at = $4
WHERE id = $1;

-- name: ActivateSubscriptionByInvoice :exec
UPDATE subscriptions
SET status = 'active', paid_at = now(), starts_at = now(), xendit_payment_id = $2
WHERE xendit_invoice_id = $1;

-- name: CountActiveSubscriptions :one
SELECT COUNT(*) FROM subscriptions WHERE status = 'active' AND expires_at > now();

-- name: ListExpiringSubscriptions :many
SELECT s.*, u.email, u.name FROM subscriptions s
JOIN users u ON u.id = s.user_id
WHERE s.status = 'active' AND s.expires_at BETWEEN now() AND now() + INTERVAL '7 days'
ORDER BY s.expires_at ASC;

-- name: SetXenditInvoiceID :exec
UPDATE subscriptions
SET xendit_invoice_id = $2
WHERE id = $1;

-- name: ListAllSubscriptions :many
SELECT s.*, u.email, u.name FROM subscriptions s
JOIN users u ON u.id = s.user_id
ORDER BY s.created_at DESC LIMIT $1 OFFSET $2;

-- name: ListSubscriptionsByStatus :many
SELECT s.*, u.email, u.name FROM subscriptions s
JOIN users u ON u.id = s.user_id
WHERE s.status = $1
ORDER BY s.created_at DESC LIMIT $2 OFFSET $3;

-- name: GetTotalRevenue :one
SELECT COALESCE(SUM(amount_paid_idr), 0)::bigint FROM subscriptions WHERE status = 'active';

-- name: CountAllSubscriptions :one
SELECT COUNT(*) FROM subscriptions;
