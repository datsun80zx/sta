-- +goose Up
-- +goose StatementBegin

-- Track each import batch (pair of files imported together)
CREATE TABLE import_batches (
    id BIGSERIAL PRIMARY KEY,
    job_report_filename TEXT NOT NULL,
    invoice_report_filename TEXT NOT NULL,
    job_report_hash TEXT NOT NULL,
    invoice_report_hash TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    row_count_jobs INTEGER NOT NULL DEFAULT 0,
    row_count_invoices INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent duplicate imports of the same file pair
CREATE UNIQUE INDEX idx_import_batches_hashes 
    ON import_batches(job_report_hash, invoice_report_hash);

CREATE INDEX idx_import_batches_imported_at 
    ON import_batches(imported_at DESC);

-- Customers (deduplicated by ServiceTitan Customer ID)
CREATE TABLE customers (
    id BIGINT PRIMARY KEY, -- ServiceTitan Customer ID
    customer_name TEXT NOT NULL,
    customer_type TEXT CHECK (customer_type IN ('Residential', 'Commercial')),
    customer_city TEXT,
    customer_state TEXT,
    customer_zip TEXT,
    location_city TEXT, -- Service address (often different from billing)
    location_state TEXT,
    location_zip TEXT,
    first_job_date DATE,
    last_job_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customers_type ON customers(customer_type);
CREATE INDEX idx_customers_location_zip ON customers(location_zip);
CREATE INDEX idx_customers_last_job_date ON customers(last_job_date DESC);

-- Jobs (one row per ServiceTitan Job ID)
CREATE TABLE jobs (
    id BIGINT PRIMARY KEY, -- ServiceTitan Job ID
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    import_batch_id BIGINT NOT NULL REFERENCES import_batches(id),
    
    -- Job details
    job_type TEXT NOT NULL,
    business_unit TEXT,
    status TEXT NOT NULL CHECK (status IN ('Scheduled', 'In Progress', 'On Hold', 'Canceled', 'Completed')),
    
    -- Dates
    job_creation_date DATE,
    job_schedule_date DATE,
    job_completion_date DATE,
    
    -- People
    assigned_technician TEXT,
    sold_by_technician TEXT,
    booked_by TEXT,
    
    -- Campaign/Marketing
    campaign_name TEXT,
    campaign_category TEXT,
    call_campaign TEXT,
    
    -- Revenue (from Job Report)
    jobs_subtotal NUMERIC(12, 2), -- Revenue without tax
    job_total NUMERIC(12, 2), -- Revenue with tax
    
    -- Other
    invoice_id BIGINT, -- Primary invoice ID from Job Report
    total_hours_worked NUMERIC(8, 2),
    priority TEXT,
    survey_score INTEGER,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_jobs_customer_id ON jobs(customer_id);
CREATE INDEX idx_jobs_import_batch_id ON jobs(import_batch_id);
CREATE INDEX idx_jobs_job_type ON jobs(job_type);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_completion_date ON jobs(job_completion_date DESC);
CREATE INDEX idx_jobs_campaign_category ON jobs(campaign_category);

-- Invoices (one row per ServiceTitan Invoice #)
CREATE TABLE invoices (
    id BIGINT PRIMARY KEY, -- ServiceTitan Invoice #
    job_id BIGINT NOT NULL REFERENCES jobs(id),
    import_batch_id BIGINT NOT NULL REFERENCES import_batches(id),
    
    -- Invoice details
    invoice_date DATE NOT NULL,
    invoice_status TEXT,
    invoice_type TEXT,
    invoice_summary TEXT,
    
    -- Totals
    total NUMERIC(12, 2), -- Invoice total with tax
    balance NUMERIC(12, 2), -- Unpaid amount
    payments NUMERIC(12, 2), -- Total payments applied
    
    -- Costs (the data we care about for profitability)
    material_costs NUMERIC(12, 2),
    equipment_costs NUMERIC(12, 2),
    purchase_order_costs NUMERIC(12, 2),
    return_costs NUMERIC(12, 2),
    costs_total NUMERIC(12, 2), -- Sum of all costs
    
    -- Retail/Markup (for future analysis)
    material_retail NUMERIC(12, 2),
    material_markup NUMERIC(12, 2),
    equipment_retail NUMERIC(12, 2),
    equipment_markup NUMERIC(12, 2),
    labor NUMERIC(12, 2), -- Labor services (retail)
    labor_pay NUMERIC(12, 2), -- Labor cost (inaccurate per Richie)
    labor_burden NUMERIC(12, 2),
    total_labor_costs NUMERIC(12, 2),
    
    -- Other
    income NUMERIC(12, 2),
    discount_total NUMERIC(12, 2),
    
    -- Critical flag for determining which invoice to use
    is_adjustment BOOLEAN NOT NULL DEFAULT false,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_job_id ON invoices(job_id);
CREATE INDEX idx_invoices_import_batch_id ON invoices(import_batch_id);
CREATE INDEX idx_invoices_invoice_date ON invoices(invoice_date DESC);
CREATE INDEX idx_invoices_is_adjustment ON invoices(is_adjustment) WHERE is_adjustment = true;

-- Pre-calculated metrics for fast querying
CREATE TABLE job_metrics (
    job_id BIGINT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    
    -- Revenue (from jobs table)
    revenue NUMERIC(12, 2) NOT NULL,
    
    -- Costs (calculated from invoices)
    total_costs NUMERIC(12, 2) NOT NULL,
    
    -- Calculated profitability
    gross_profit NUMERIC(12, 2) NOT NULL,
    gross_margin_pct NUMERIC(8, 2),
    
    -- Metadata
    invoice_count INTEGER NOT NULL, -- How many invoices were used
    has_adjustment BOOLEAN NOT NULL DEFAULT false,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_job_metrics_margin_pct ON job_metrics(gross_margin_pct DESC);
CREATE INDEX idx_job_metrics_gross_profit ON job_metrics(gross_profit DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS job_metrics CASCADE;
DROP TABLE IF EXISTS invoices CASCADE;
DROP TABLE IF EXISTS jobs CASCADE;
DROP TABLE IF EXISTS customers CASCADE;
DROP TABLE IF EXISTS import_batches CASCADE;

-- +goose StatementEnd