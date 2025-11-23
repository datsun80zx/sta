-- name: UpsertCustomer :one
INSERT INTO customers (
    id, customer_name, customer_type,
    customer_city, customer_state, customer_zip,
    location_city, location_state, location_zip,
    first_job_date, last_job_date
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE SET
    customer_name = EXCLUDED.customer_name,
    customer_type = EXCLUDED.customer_type,
    customer_city = EXCLUDED.customer_city,
    customer_state = EXCLUDED.customer_state,
    customer_zip = EXCLUDED.customer_zip,
    location_city = EXCLUDED.location_city,
    location_state = EXCLUDED.location_state,
    location_zip = EXCLUDED.location_zip,
    last_job_date = GREATEST(customers.last_job_date, EXCLUDED.last_job_date),
    first_job_date = LEAST(customers.first_job_date, EXCLUDED.first_job_date),
    updated_at = NOW()
RETURNING *;

-- name: GetCustomer :one
SELECT * FROM customers WHERE id = $1;