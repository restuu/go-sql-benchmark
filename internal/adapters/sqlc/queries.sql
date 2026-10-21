-- Explicit columns (no SELECT *) so every adapter reads the same columns.
-- :execrows returns the rows affected, which the runner verifies.

-- name: CreateModel :execrows
INSERT INTO models (name, title, fax, web, age, "right", counter)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateModel :execrows
UPDATE models
SET name = $1, title = $2, fax = $3, web = $4, age = $5, "right" = $6, counter = $7
WHERE id = $8;

-- name: DeleteModel :execrows
DELETE FROM models WHERE id = $1;

-- name: GetModel :one
SELECT id, name, title, fax, web, age, "right", counter
FROM models
WHERE id = $1;

-- name: JoinOrders :many
SELECT o.id AS order_id, o.customer_id, o.status, o.total, o.created_at, c.name, c.email, c.city
FROM orders o JOIN customers c ON c.id = o.customer_id
WHERE c.id = $1;

-- name: UnionByCity :many
SELECT c.name, c.email, c.city FROM customers c WHERE c.city = $1
UNION
SELECT s.name, s.email, s.city FROM suppliers s WHERE s.city = $1;
