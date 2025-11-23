-- name: CreateImportBatch :one
INSERT INTO import_batches (
    job_report_filename,
    invoice_report_filename,
    job_report_hash,
    invoice_report_hash,
    row_count_jobs,
    row_count_invoices,
    status
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetImportBatchByHashes :one
SELECT * FROM import_batches
WHERE job_report_hash = $1 AND invoice_report_hash = $2
LIMIT 1;

-- name: UpdateImportBatchStatus :exec
UPDATE import_batches
SET status = $2, error_message = $3
WHERE id = $1;

-- name: ListImportBatches :many
SELECT * FROM import_batches
ORDER BY imported_at DESC
LIMIT $1;