-- Queries for fetching data needed by Go-side metrics calculations

-- name: GetJobsForMetrics :many
SELECT 
    id,
    status,
    jobs_subtotal
FROM jobs
WHERE import_batch_id = $1;

-- name: GetInvoicesForMetrics :many
SELECT 
    id,
    job_id,
    costs_total,
    is_adjustment
FROM invoices
WHERE import_batch_id = $1;

-- name: GetJobsForTechnicianMetrics :many
SELECT 
    id,
    status,
    jobs_subtotal,
    total_hours_worked,
    COALESCE(estimate_count, 0) as estimate_count,
    is_opportunity,
    is_converted
FROM jobs
WHERE import_batch_id = $1;

-- name: GetAllTechnicianIDs :many
SELECT id FROM technicians;

-- name: GetJobTechniciansForBatch :many
SELECT 
    jt.job_id,
    jt.technician_id,
    jt.role
FROM job_technicians jt
JOIN jobs j ON jt.job_id = j.id
WHERE j.import_batch_id = $1;

-- name: GetAllJobMetrics :many
SELECT 
    job_id,
    revenue,
    total_costs,
    gross_profit,
    gross_margin_pct,
    invoice_count,
    has_adjustment
FROM job_metrics;