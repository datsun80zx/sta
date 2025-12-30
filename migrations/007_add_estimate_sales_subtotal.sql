-- +goose Up
-- +goose StatementBegin

-- Add estimate_sales_subtotal to jobs table
-- This represents what the technician sold via estimates (distinct from job revenue)
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS estimate_sales_subtotal NUMERIC(12, 2);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE jobs DROP COLUMN IF EXISTS estimate_sales_subtotal;

-- +goose StatementEnd