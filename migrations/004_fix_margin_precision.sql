-- +goose Up
-- +goose StatementBegin

-- Fix gross_margin_pct overflow
-- NUMERIC(8,2) maxes at 999,999.99 which can overflow with edge case calculations
-- Using unconstrained NUMERIC allows any value, or we can clamp in the query instead

ALTER TABLE job_metrics 
    ALTER COLUMN gross_margin_pct TYPE NUMERIC;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE job_metrics 
    ALTER COLUMN gross_margin_pct TYPE NUMERIC;

-- +goose StatementEnd