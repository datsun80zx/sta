-- name: UpsertTechnician :one
INSERT INTO technicians (name, first_seen_date, last_seen_date)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET
    first_seen_date = LEAST(technicians.first_seen_date, EXCLUDED.first_seen_date),
    last_seen_date = GREATEST(technicians.last_seen_date, EXCLUDED.last_seen_date),
    updated_at = NOW()
RETURNING *;

-- name: GetTechnicianByName :one
SELECT * FROM technicians WHERE name = $1;

-- name: CreateJobTechnician :exec
INSERT INTO job_technicians (job_id, technician_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (job_id, technician_id, role) DO NOTHING;

-- name: GetTechnicianPerformance :many
SELECT 
    t.id,
    t.name,
    tm.jobs_sold,
    tm.avg_sale,
    tm.conversion_rate,
    tm.jobs_serviced,
    tm.avg_hours_per_job,
    tm.avg_estimates_per_job,
    tm.avg_gross_profit,
    tm.avg_margin_pct
FROM technicians t
JOIN technician_metrics tm ON t.id = tm.technician_id
WHERE tm.jobs_sold > 0 OR tm.jobs_serviced > 0
ORDER BY tm.total_sales DESC;

-- name: GetTopTechniciansBySales :many
SELECT 
    t.name,
    tm.jobs_sold,
    tm.total_sales,
    tm.avg_sale,
    tm.conversion_rate,
    tm.avg_margin_pct
FROM technicians t
JOIN technician_metrics tm ON t.id = tm.technician_id
WHERE tm.jobs_sold > 0
ORDER BY tm.avg_sale DESC
LIMIT $1;

-- name: GetTopTechniciansByConversion :many
SELECT 
    t.name,
    tm.opportunities,
    tm.conversions,
    tm.conversion_rate,
    tm.avg_sale
FROM technicians t
JOIN technician_metrics tm ON t.id = tm.technician_id
WHERE tm.opportunities >= 5  -- minimum sample size
ORDER BY tm.conversion_rate DESC
LIMIT $1;

-- name: GetTechniciansByEfficiency :many
SELECT 
    t.name,
    tm.jobs_serviced,
    tm.total_hours_worked,
    tm.avg_hours_per_job,
    tm.avg_estimates_per_job
FROM technicians t
JOIN technician_metrics tm ON t.id = tm.technician_id
WHERE tm.jobs_serviced > 0
ORDER BY tm.avg_hours_per_job ASC
LIMIT $1;