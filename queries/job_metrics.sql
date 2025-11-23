-- name: CalculateJobMetrics :exec
INSERT INTO job_metrics (job_id, revenue, total_costs, gross_profit, gross_margin_pct, invoice_count, has_adjustment)
SELECT 
    j.id as job_id,
    j.jobs_subtotal as revenue,
    CASE 
        WHEN EXISTS (SELECT 1 FROM invoices WHERE job_id = j.id AND is_adjustment = true) 
        THEN (
            SELECT costs_total 
            FROM invoices 
            WHERE job_id = j.id AND is_adjustment = true 
            ORDER BY invoice_date DESC 
            LIMIT 1
        )
        ELSE COALESCE((
            SELECT SUM(costs_total) 
            FROM invoices 
            WHERE job_id = j.id AND is_adjustment = false
        ), 0)
    END as total_costs,
    j.jobs_subtotal - CASE 
        WHEN EXISTS (SELECT 1 FROM invoices WHERE job_id = j.id AND is_adjustment = true) 
        THEN (
            SELECT costs_total 
            FROM invoices 
            WHERE job_id = j.id AND is_adjustment = true 
            ORDER BY invoice_date DESC 
            LIMIT 1
        )
        ELSE COALESCE((
            SELECT SUM(costs_total) 
            FROM invoices 
            WHERE job_id = j.id AND is_adjustment = false
        ), 0)
    END as gross_profit,
    CASE 
        WHEN j.jobs_subtotal > 0 THEN
            ((j.jobs_subtotal - CASE 
                WHEN EXISTS (SELECT 1 FROM invoices WHERE job_id = j.id AND is_adjustment = true) 
                THEN (
                    SELECT costs_total 
                    FROM invoices 
                    WHERE job_id = j.id AND is_adjustment = true 
                    ORDER BY invoice_date DESC 
                    LIMIT 1
                )
                ELSE COALESCE((
                    SELECT SUM(costs_total) 
                    FROM invoices 
                    WHERE job_id = j.id AND is_adjustment = false
                ), 0)
            END) / j.jobs_subtotal) * 100
        ELSE NULL
    END as gross_margin_pct,
    (SELECT COUNT(*) FROM invoices WHERE job_id = j.id) as invoice_count,
    EXISTS (SELECT 1 FROM invoices WHERE job_id = j.id AND is_adjustment = true) as has_adjustment
FROM jobs j
WHERE j.import_batch_id = $1
  AND j.status = 'Completed'
  AND j.jobs_subtotal IS NOT NULL
  AND EXISTS (SELECT 1 FROM invoices WHERE job_id = j.id)
ON CONFLICT (job_id) DO UPDATE SET
    revenue = EXCLUDED.revenue,
    total_costs = EXCLUDED.total_costs,
    gross_profit = EXCLUDED.gross_profit,
    gross_margin_pct = EXCLUDED.gross_margin_pct,
    invoice_count = EXCLUDED.invoice_count,
    has_adjustment = EXCLUDED.has_adjustment,
    calculated_at = NOW();

-- name: GetProfitByJobType :many
SELECT 
    j.job_type,
    COUNT(*) as job_count,
    AVG(m.revenue)::numeric(12,2) as avg_revenue,
    AVG(m.total_costs)::numeric(12,2) as avg_costs,
    AVG(m.gross_profit)::numeric(12,2) as avg_gross_profit,
    AVG(m.gross_margin_pct)::numeric(8,2) as avg_margin_pct,
    SUM(m.gross_profit)::numeric(12,2) as total_profit
FROM jobs j
JOIN job_metrics m ON j.id = m.job_id
WHERE j.status = 'Completed'
GROUP BY j.job_type
ORDER BY avg_gross_profit DESC;