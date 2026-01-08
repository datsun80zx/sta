-- Analytics queries - raw data only, no aggregations
-- All calculations done in Go

-- name: GetJobsForAnalytics :many
-- Fetches raw job data for analytics calculations
SELECT 
    j.id,
    j.project_id,
    j.status,
    j.job_completion_date,
    j.business_unit,
    j.total_hours_worked,
    COALESCE(j.estimate_count, 0) as estimate_count,
    j.estimate_sales_subtotal,
    j.assigned_technician,
    j.warranty,
    j.warranty_for_job_id,
    j.recall,
    j.recall_for_job_id
FROM jobs j
WHERE j.status = 'Completed'
ORDER BY j.job_completion_date DESC;

-- name: GetJobsForAnalyticsByDateRange :many
-- Fetches raw job data within a date range
SELECT 
    j.id,
    j.project_id,
    j.status,
    j.job_completion_date,
    j.business_unit,
    j.total_hours_worked,
    COALESCE(j.estimate_count, 0) as estimate_count,
    j.estimate_sales_subtotal,
    j.assigned_technician,
    j.warranty,
    j.warranty_for_job_id,
    j.recall,
    j.recall_for_job_id
FROM jobs j
WHERE j.status = 'Completed'
  AND j.job_completion_date >= $1
  AND j.job_completion_date <= $2
ORDER BY j.job_completion_date DESC;

-- name: GetJobsForAnalyticsByBusinessUnit :many
-- Fetches raw job data filtered by business unit
SELECT 
    j.id,
    j.project_id,
    j.status,
    j.job_completion_date,
    j.business_unit,
    j.total_hours_worked,
    COALESCE(j.estimate_count, 0) as estimate_count,
    j.estimate_sales_subtotal,
    j.assigned_technician,
    j.warranty,
    j.warranty_for_job_id,
    j.recall,
    j.recall_for_job_id
FROM jobs j
WHERE j.status = 'Completed'
  AND j.business_unit = $1
ORDER BY j.job_completion_date DESC;

-- name: GetJobTechniciansForAnalytics :many
-- Fetches all job-technician relationships with technician names
SELECT 
    jt.job_id,
    jt.technician_id,
    t.name as technician_name,
    jt.role
FROM job_technicians jt
JOIN technicians t ON t.id = jt.technician_id;

-- name: GetJobTechniciansForAnalyticsByJobs :many
-- Fetches job-technician relationships for specific jobs
SELECT 
    jt.job_id,
    jt.technician_id,
    t.name as technician_name,
    jt.role
FROM job_technicians jt
JOIN technicians t ON t.id = jt.technician_id
WHERE jt.job_id = ANY($1::text[]);

-- name: GetTechniciansForAnalytics :many
-- Fetches all technicians with their most common business unit
SELECT 
    t.id,
    t.name,
    (
        SELECT j.business_unit 
        FROM job_technicians jt
        JOIN jobs j ON j.id = jt.job_id
        WHERE jt.technician_id = t.id 
          AND j.business_unit IS NOT NULL
        GROUP BY j.business_unit
        ORDER BY COUNT(*) DESC
        LIMIT 1
    ) as business_unit
FROM technicians t
ORDER BY t.name;

-- name: GetTechnicianByID :one
SELECT 
    t.id,
    t.name,
    (
        SELECT j.business_unit 
        FROM job_technicians jt
        JOIN jobs j ON j.id = jt.job_id
        WHERE jt.technician_id = t.id 
          AND j.business_unit IS NOT NULL
        GROUP BY j.business_unit
        ORDER BY COUNT(*) DESC
        LIMIT 1
    ) as business_unit
FROM technicians t
WHERE t.id = $1;

-- name: GetBusinessUnits :many
-- Lists all unique business units
SELECT DISTINCT business_unit
FROM jobs
WHERE business_unit IS NOT NULL
ORDER BY business_unit;

-- name: GetDateRangeBounds :one
-- Gets the earliest and latest job completion dates
SELECT 
    MIN(job_completion_date) as earliest_date,
    MAX(job_completion_date) as latest_date
FROM jobs
WHERE job_completion_date IS NOT NULL
  AND status = 'Completed';