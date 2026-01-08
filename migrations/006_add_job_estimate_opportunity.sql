-- +goose Up
-- +goose StatementBegin

-- Add columns needed for technician performance metrics
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS estimate_count INTEGER DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS is_opportunity BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS is_converted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS primary_technician TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS warranty BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS recall BOOLEAN NOT NULL DEFAULT false;

-- Index for conversion analysis
CREATE INDEX IF NOT EXISTS idx_jobs_opportunity ON jobs(is_opportunity) WHERE is_opportunity = true;
CREATE INDEX IF NOT EXISTS idx_jobs_converted ON jobs(is_converted) WHERE is_converted = true;
CREATE INDEX IF NOT EXISTS idx_jobs_primary_technician ON jobs(primary_technician);
CREATE INDEX IF NOT EXISTS idx_jobs_warranty ON jobs(warranty) WHERE warranty = true;
CREATE INDEX IF NOT EXISTS idx_jobs_recall ON jobs(recall) WHERE recall = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_jobs_recall;
DROP INDEX IF EXISTS idx_jobs_warranty;
DROP INDEX IF EXISTS idx_jobs_primary_technician;
DROP INDEX IF EXISTS idx_jobs_converted;
DROP INDEX IF EXISTS idx_jobs_opportunity;
ALTER TABLE jobs DROP COLUMN IF EXISTS recall;
ALTER TABLE jobs DROP COLUMN IF EXISTS warranty;
ALTER TABLE jobs DROP COLUMN IF EXISTS primary_technician;
ALTER TABLE jobs DROP COLUMN IF EXISTS is_converted;
ALTER TABLE jobs DROP COLUMN IF EXISTS is_opportunity;
ALTER TABLE jobs DROP COLUMN IF EXISTS estimate_count;

-- +goose StatementEnd