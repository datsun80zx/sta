-- name: CreateJob :one
INSERT INTO jobs (
    id, customer_id, import_batch_id,
    job_type, business_unit, status,
    job_creation_date, job_schedule_date, job_completion_date,
    assigned_technician, sold_by_technician, booked_by,
    campaign_name, campaign_category, call_campaign,
    jobs_subtotal, job_total,
    invoice_id, total_hours_worked, priority, survey_score
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
RETURNING *;

-- name: GetJobsWithoutInvoices :many
SELECT j.id, j.job_type, j.customer_id
FROM jobs j
LEFT JOIN invoices i ON j.id = i.job_id
WHERE i.id IS NULL
AND j.import_batch_id = $1;