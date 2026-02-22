-- name: GetPricingPlan :one
SELECT * FROM pricing_plans WHERE id = $1;

-- name: ListPricingPlansByProduct :many
SELECT * FROM pricing_plans
WHERE product_id = $1 AND is_active = TRUE
ORDER BY price_idr ASC;

-- name: CreatePricingPlan :one
INSERT INTO pricing_plans (product_id, segment, duration, duration_days, price_idr, label, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdatePricingPlan :exec
UPDATE pricing_plans
SET price_idr = $2, label = $3, is_active = $4
WHERE id = $1;
