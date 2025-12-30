-- +goose Up
-- +goose StatementBegin

-- Technicians table (deduplicated by name since ServiceTitan exports don't include tech IDs)
CREATE TABLE technicians (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    first_seen_date DATE,
    last_seen_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_technicians_name ON technicians(name);

-- Junction table for job-technician relationships
-- A technician can be associated with a job in multiple roles
CREATE TABLE job_technicians (
    id BIGSERIAL PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    technician_id BIGINT NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('assigned', 'sold_by', 'primary')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(job_id, technician_id, role)
);

CREATE INDEX idx_job_technicians_job_id ON job_technicians(job_id);
CREATE INDEX idx_job_technicians_technician_id ON job_technicians(technician_id);
CREATE INDEX idx_job_technicians_role ON job_technicians(role);

-- Pre-calculated technician performance metrics
CREATE TABLE technician_metrics (
    technician_id BIGINT PRIMARY KEY REFERENCES technicians(id) ON DELETE CASCADE,
    
    -- Sales metrics (jobs where technician is "sold_by")
    jobs_sold INTEGER NOT NULL DEFAULT 0,
    total_sales NUMERIC(12, 2) NOT NULL DEFAULT 0,
    avg_sale NUMERIC(12, 2),
    
    -- Conversion metrics (opportunities vs conversions where technician is "sold_by")
    opportunities INTEGER NOT NULL DEFAULT 0,
    conversions INTEGER NOT NULL DEFAULT 0,
    conversion_rate NUMERIC(5, 2),  -- as percentage (0-100)
    
    -- Service metrics (jobs where technician is "primary")
    jobs_serviced INTEGER NOT NULL DEFAULT 0,
    total_hours_worked NUMERIC(10, 2) NOT NULL DEFAULT 0,
    avg_hours_per_job NUMERIC(8, 2),
    
    -- Estimate metrics (jobs where technician is "sold_by" or "primary")
    total_estimates INTEGER NOT NULL DEFAULT 0,
    jobs_with_estimates INTEGER NOT NULL DEFAULT 0,
    avg_estimates_per_job NUMERIC(5, 2),
    
    -- Profitability (from job_metrics for jobs where tech is sold_by)
    total_gross_profit NUMERIC(12, 2),
    avg_gross_profit NUMERIC(12, 2),
    avg_margin_pct NUMERIC(8, 2),
    
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_technician_metrics_avg_sale ON technician_metrics(avg_sale DESC);
CREATE INDEX idx_technician_metrics_conversion_rate ON technician_metrics(conversion_rate DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS technician_metrics CASCADE;
DROP TABLE IF EXISTS job_technicians CASCADE;
DROP TABLE IF EXISTS technicians CASCADE;

-- +goose StatementEnd