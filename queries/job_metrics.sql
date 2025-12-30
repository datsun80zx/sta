-- Simplified job_metrics.sql - calculations done in Go

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