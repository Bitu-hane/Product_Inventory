
-- name: CreateProduct :one
-- Inserts a new product and returns the created record.
INSERT INTO products (name,sku,description,price,stock_quantity,status)VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;


-- name: GetProductByID :one
-- Fetches a single product by its primary key ID.
SELECT * FROM products WHERE id = $1;


-- name: GetProductBySKU :one
-- Fetches a single product by its unique SKU.
SELECT * FROM products WHERE sku = $1;


-- name: ListProducts :many
-- Returns products with pagination and optional status filtering.
SELECT * FROM products WHERE ( sqlc.narg('status')::VARCHAR IS NULL OR status = 
sqlc.narg('status')) ORDER BY created_at DESC 
LIMIT $1 OFFSET $2;


-- name: CountProducts :one
-- Counts products, optionally filtering by status.
SELECT COUNT(*) FROM products WHERE (
    sqlc.narg('status')::VARCHAR IS NULL
    OR status = sqlc.narg('status')
);


-- name: UpdateProduct :one
-- Updates an existing product and returns the updated record.
UPDATE products SET name = $2, sku = $3, description = $4, price = $5, stock_quantity = $6, status = $7, updated_at = NOW()
WHERE id = $1
RETURNING *;


-- name: DeleteProduct :exec
-- Deletes a product by its primary key ID.
DELETE FROM products WHERE id = $1;
