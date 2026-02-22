-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: ListProducts :many
SELECT * FROM products WHERE is_active = TRUE ORDER BY name;

-- name: CreateProduct :one
INSERT INTO products (id, name, description, is_active)
VALUES ($1, $2, $3, $4)
RETURNING *;
