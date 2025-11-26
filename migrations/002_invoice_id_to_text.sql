-- +goose Up
-- +goose StatementBegin

-- Change invoice.id from BIGINT to TEXT
-- This allows storing ServiceTitan's invoice numbers with suffixes like "136860206-1"

-- Drop the foreign key constraint temporarily
ALTER TABLE invoices DROP CONSTRAINT invoices_job_id_fkey;

-- Change the column type
ALTER TABLE invoices ALTER COLUMN id TYPE TEXT USING id::TEXT;

-- Also update job_metrics if it references invoice IDs (it doesn't currently, but for completeness)
-- Jobs table has invoice_id as optional reference
ALTER TABLE jobs ALTER COLUMN invoice_id TYPE TEXT USING invoice_id::TEXT;

-- Recreate foreign key
ALTER TABLE invoices ADD CONSTRAINT invoices_job_id_fkey 
    FOREIGN KEY (job_id) REFERENCES jobs(id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse the changes
ALTER TABLE invoices DROP CONSTRAINT invoices_job_id_fkey;

-- This will fail if there are any IDs with suffixes, which is expected
ALTER TABLE invoices ALTER COLUMN id TYPE BIGINT USING id::BIGINT;
ALTER TABLE jobs ALTER COLUMN invoice_id TYPE BIGINT USING invoice_id::BIGINT;

ALTER TABLE invoices ADD CONSTRAINT invoices_job_id_fkey 
    FOREIGN KEY (job_id) REFERENCES jobs(id);

-- +goose StatementEnd