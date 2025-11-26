-- +goose Up
-- +goose StatementBegin

-- Convert job IDs from BIGINT to TEXT
-- This allows storing ServiceTitan job numbers with letter prefixes like "D22713050"

-- Drop foreign key constraints that reference jobs.id
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_job_id_fkey;
ALTER TABLE job_metrics DROP CONSTRAINT IF EXISTS job_metrics_job_id_fkey;

-- Convert the primary key and foreign keys
ALTER TABLE jobs ALTER COLUMN id TYPE TEXT USING id::TEXT;
ALTER TABLE invoices ALTER COLUMN job_id TYPE TEXT USING job_id::TEXT;
ALTER TABLE job_metrics ALTER COLUMN job_id TYPE TEXT USING job_id::TEXT;

-- Recreate foreign key constraints
ALTER TABLE invoices ADD CONSTRAINT invoices_job_id_fkey 
    FOREIGN KEY (job_id) REFERENCES jobs(id);
    
ALTER TABLE job_metrics ADD CONSTRAINT job_metrics_job_id_fkey 
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse the changes
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_job_id_fkey;
ALTER TABLE job_metrics DROP CONSTRAINT IF EXISTS job_metrics_job_id_fkey;

-- This will fail if there are any job IDs with letter prefixes
ALTER TABLE jobs ALTER COLUMN id TYPE BIGINT USING id::BIGINT;
ALTER TABLE invoices ALTER COLUMN job_id TYPE BIGINT USING job_id::BIGINT;
ALTER TABLE job_metrics ALTER COLUMN job_id TYPE BIGINT USING job_id::BIGINT;

-- Recreate constraints
ALTER TABLE invoices ADD CONSTRAINT invoices_job_id_fkey 
    FOREIGN KEY (job_id) REFERENCES jobs(id);
    
ALTER TABLE job_metrics ADD CONSTRAINT job_metrics_job_id_fkey 
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE;

-- +goose StatementEnd