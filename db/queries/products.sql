-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: ListProducts :many
SELECT * FROM products WHERE is_active = TRUE ORDER BY name;

-- name: CreateProduct :one
INSERT INTO products (id, name, description, is_active)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListAllProducts :many
SELECT * FROM products ORDER BY created_at DESC;

-- name: UpdateProduct :exec
UPDATE products
SET name = $2, description = $3
WHERE id = $1;

-- name: ToggleProductActive :exec
UPDATE products
SET is_active = NOT is_active
WHERE id = $1;
