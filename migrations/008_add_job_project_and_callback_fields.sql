-- +goose Up
-- +goose StatementBegin

-- Add project_id for grouping related jobs (used for sales deduplication)
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS project_id TEXT;

-- Add callback tracking fields (to trace warranty/recall back to original job)
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS warranty_for_job_id TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS recall_for_job_id TEXT;

-- Index for callback lookups (find all jobs that reference a specific original job)
CREATE INDEX IF NOT EXISTS idx_jobs_warranty_for ON jobs(warranty_for_job_id) WHERE warranty_for_job_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_recall_for ON jobs(recall_for_job_id) WHERE recall_for_job_id IS NOT NULL;

-- Index for project grouping
CREATE INDEX IF NOT EXISTS idx_jobs_project_id ON jobs(project_id) WHERE project_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_jobs_project_id;
DROP INDEX IF EXISTS idx_jobs_recall_for;
DROP INDEX IF EXISTS idx_jobs_warranty_for;

ALTER TABLE jobs DROP COLUMN IF EXISTS recall_for_job_id;
ALTER TABLE jobs DROP COLUMN IF EXISTS warranty_for_job_id;
ALTER TABLE jobs DROP COLUMN IF EXISTS project_id;

-- +goose StatementEnd