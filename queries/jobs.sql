-- name: CreateJob :one
INSERT INTO jobs (
    id, customer_id, import_batch_id,
    job_type, business_unit, status,
    job_creation_date, job_schedule_date, job_completion_date,
    assigned_technician, sold_by_technician, booked_by,
    campaign_name, campaign_category, call_campaign,
    jobs_subtotal, job_total, estimate_sales_subtotal,
    invoice_id, total_hours_worked, priority, survey_score,
    estimate_count, is_opportunity, is_converted, primary_technician
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
)
RETURNING *;

-- name: GetJobsWithoutInvoices :many
SELECT j.id, j.job_type, j.customer_id
FROM jobs j
LEFT JOIN invoices i ON j.id = i.job_id
WHERE i.id IS NULL
AND j.import_batch_id = $1;

-- name: GetJobsForTechnicianProcessing :many
SELECT 
    id, 
    assigned_technician, 
    sold_by_technician, 
    primary_technician,
    job_completion_date
FROM jobs
WHERE import_batch_id = $1;