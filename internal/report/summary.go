package report

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SummaryReport contains all data for the summary report
type SummaryReport struct {
	GeneratedAt time.Time
	FromDate    *time.Time
	ToDate      *time.Time

	// Executive Summary
	TotalJobs    int
	TotalRevenue float64
	TotalCosts   float64
	TotalProfit  float64
	AvgMarginPct float64
	JobsWithLoss int
	TotalLoss    float64

	// Breakdowns
	JobTypes     []JobTypeStats
	Campaigns    []CampaignStats
	TopCustomers []CustomerStats
	RedFlagJobs  []RedFlagJob
}

// JobTypeStats represents profitability stats for a job type
type JobTypeStats struct {
	JobType      string
	JobCount     int
	AvgRevenue   float64
	AvgCosts     float64
	AvgProfit    float64
	AvgMarginPct *float64
	TotalProfit  float64
}

// CampaignStats represents profitability stats for a campaign
type CampaignStats struct {
	CampaignName     string
	CampaignCategory string
	JobCount         int
	AvgRevenue       float64
	AvgProfit        float64
	AvgMarginPct     *float64
	TotalProfit      float64
}

// CustomerStats represents profitability stats for a customer
type CustomerStats struct {
	CustomerID   int64
	CustomerName string
	CustomerType string
	JobCount     int
	AvgProfit    float64
	AvgMarginPct *float64
	TotalProfit  float64
}

// RedFlagJob represents a job with negative margin
type RedFlagJob struct {
	JobID          string
	CustomerName   string
	JobType        string
	Revenue        float64
	Costs          float64
	Loss           float64
	CompletionDate *time.Time
}

// GenerateSummary builds the complete summary report
func GenerateSummary(ctx context.Context, db *sql.DB, fromDate, toDate *time.Time) (*SummaryReport, error) {
	report := &SummaryReport{
		GeneratedAt: time.Now(),
		FromDate:    fromDate,
		ToDate:      toDate,
	}

	var err error

	// Get executive summary stats
	if err = loadExecutiveSummary(ctx, db, report, fromDate, toDate); err != nil {
		return nil, fmt.Errorf("loading executive summary: %w", err)
	}

	// Get job type breakdown
	if report.JobTypes, err = loadJobTypes(ctx, db, fromDate, toDate); err != nil {
		return nil, fmt.Errorf("loading job types: %w", err)
	}

	// Get campaign breakdown
	if report.Campaigns, err = loadCampaigns(ctx, db, fromDate, toDate); err != nil {
		return nil, fmt.Errorf("loading campaigns: %w", err)
	}

	// Get top customers
	if report.TopCustomers, err = loadTopCustomers(ctx, db, fromDate, toDate, 10); err != nil {
		return nil, fmt.Errorf("loading top customers: %w", err)
	}

	// Get red flag jobs
	if report.RedFlagJobs, err = loadRedFlagJobs(ctx, db, fromDate, toDate); err != nil {
		return nil, fmt.Errorf("loading red flag jobs: %w", err)
	}

	return report, nil
}

func buildDateClause(fromDate, toDate *time.Time, argOffset int) (string, []interface{}) {
	var clause string
	var args []interface{}

	if fromDate != nil {
		argOffset++
		clause += fmt.Sprintf(" AND j.job_completion_date >= $%d", argOffset)
		args = append(args, *fromDate)
	}

	if toDate != nil {
		argOffset++
		clause += fmt.Sprintf(" AND j.job_completion_date <= $%d", argOffset)
		args = append(args, *toDate)
	}

	return clause, args
}

func loadExecutiveSummary(ctx context.Context, db *sql.DB, report *SummaryReport, fromDate, toDate *time.Time) error {
	dateClause, dateArgs := buildDateClause(fromDate, toDate, 0)

	query := `
		SELECT 
			COUNT(*) as total_jobs,
			COALESCE(SUM(m.revenue), 0) as total_revenue,
			COALESCE(SUM(m.total_costs), 0) as total_costs,
			COALESCE(SUM(m.gross_profit), 0) as total_profit,
			COALESCE(AVG(m.gross_margin_pct) FILTER (WHERE m.gross_margin_pct IS NOT NULL), 0) as avg_margin_pct,
			COUNT(*) FILTER (WHERE m.gross_profit < 0) as jobs_with_loss,
			COALESCE(SUM(m.gross_profit) FILTER (WHERE m.gross_profit < 0), 0) as total_loss
		FROM jobs j
		JOIN job_metrics m ON j.id = m.job_id
		WHERE j.status = 'Completed'` + dateClause

	row := db.QueryRowContext(ctx, query, dateArgs...)
	return row.Scan(
		&report.TotalJobs,
		&report.TotalRevenue,
		&report.TotalCosts,
		&report.TotalProfit,
		&report.AvgMarginPct,
		&report.JobsWithLoss,
		&report.TotalLoss,
	)
}

func loadJobTypes(ctx context.Context, db *sql.DB, fromDate, toDate *time.Time) ([]JobTypeStats, error) {
	dateClause, dateArgs := buildDateClause(fromDate, toDate, 0)

	query := `
		SELECT 
			j.job_type,
			COUNT(*) as job_count,
			AVG(m.revenue)::numeric(12,2) as avg_revenue,
			AVG(m.total_costs)::numeric(12,2) as avg_costs,
			AVG(m.gross_profit)::numeric(12,2) as avg_gross_profit,
			AVG(m.gross_margin_pct) FILTER (WHERE m.gross_margin_pct IS NOT NULL)::numeric(8,2) as avg_margin_pct,
			SUM(m.gross_profit)::numeric(12,2) as total_profit
		FROM jobs j
		JOIN job_metrics m ON j.id = m.job_id
		WHERE j.status = 'Completed'` + dateClause + `
		GROUP BY j.job_type
		ORDER BY total_profit DESC
	`

	rows, err := db.QueryContext(ctx, query, dateArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []JobTypeStats
	for rows.Next() {
		var r JobTypeStats
		var marginPct sql.NullFloat64
		err := rows.Scan(
			&r.JobType,
			&r.JobCount,
			&r.AvgRevenue,
			&r.AvgCosts,
			&r.AvgProfit,
			&marginPct,
			&r.TotalProfit,
		)
		if err != nil {
			return nil, err
		}
		if marginPct.Valid {
			r.AvgMarginPct = &marginPct.Float64
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func loadCampaigns(ctx context.Context, db *sql.DB, fromDate, toDate *time.Time) ([]CampaignStats, error) {
	dateClause, dateArgs := buildDateClause(fromDate, toDate, 0)

	query := `
		SELECT 
			COALESCE(j.campaign_name, 'Unknown') as campaign_name,
			COALESCE(j.campaign_category, 'Uncategorized') as campaign_category,
			COUNT(*) as job_count,
			AVG(m.revenue)::numeric(12,2) as avg_revenue,
			AVG(m.gross_profit)::numeric(12,2) as avg_gross_profit,
			AVG(m.gross_margin_pct) FILTER (WHERE m.gross_margin_pct IS NOT NULL)::numeric(8,2) as avg_margin_pct,
			SUM(m.gross_profit)::numeric(12,2) as total_profit
		FROM jobs j
		JOIN job_metrics m ON j.id = m.job_id
		WHERE j.status = 'Completed'` + dateClause + `
		GROUP BY j.campaign_name, j.campaign_category
		ORDER BY total_profit DESC
	`

	rows, err := db.QueryContext(ctx, query, dateArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CampaignStats
	for rows.Next() {
		var r CampaignStats
		var marginPct sql.NullFloat64
		err := rows.Scan(
			&r.CampaignName,
			&r.CampaignCategory,
			&r.JobCount,
			&r.AvgRevenue,
			&r.AvgProfit,
			&marginPct,
			&r.TotalProfit,
		)
		if err != nil {
			return nil, err
		}
		if marginPct.Valid {
			r.AvgMarginPct = &marginPct.Float64
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func loadTopCustomers(ctx context.Context, db *sql.DB, fromDate, toDate *time.Time, limit int) ([]CustomerStats, error) {
	dateClause, dateArgs := buildDateClause(fromDate, toDate, 1) // offset by 1 for LIMIT

	query := `
		SELECT 
			c.id as customer_id,
			c.customer_name,
			COALESCE(c.customer_type, 'Unknown') as customer_type,
			COUNT(j.id) as job_count,
			AVG(m.gross_profit)::numeric(12,2) as avg_profit_per_job,
			AVG(m.gross_margin_pct) FILTER (WHERE m.gross_margin_pct IS NOT NULL)::numeric(8,2) as avg_margin_pct,
			SUM(m.gross_profit)::numeric(12,2) as total_profit
		FROM customers c
		JOIN jobs j ON c.id = j.customer_id
		JOIN job_metrics m ON j.id = m.job_id
		WHERE j.status = 'Completed'` + dateClause + `
		GROUP BY c.id, c.customer_name, c.customer_type
		ORDER BY total_profit DESC
		LIMIT $1
	`

	queryArgs := []interface{}{limit}
	queryArgs = append(queryArgs, dateArgs...)

	rows, err := db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CustomerStats
	for rows.Next() {
		var r CustomerStats
		var marginPct sql.NullFloat64
		err := rows.Scan(
			&r.CustomerID,
			&r.CustomerName,
			&r.CustomerType,
			&r.JobCount,
			&r.AvgProfit,
			&marginPct,
			&r.TotalProfit,
		)
		if err != nil {
			return nil, err
		}
		if marginPct.Valid {
			r.AvgMarginPct = &marginPct.Float64
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func loadRedFlagJobs(ctx context.Context, db *sql.DB, fromDate, toDate *time.Time) ([]RedFlagJob, error) {
	dateClause, dateArgs := buildDateClause(fromDate, toDate, 0)

	query := `
		SELECT 
			j.id as job_id,
			c.customer_name,
			j.job_type,
			m.revenue,
			m.total_costs,
			m.gross_profit,
			j.job_completion_date
		FROM jobs j
		JOIN job_metrics m ON j.id = m.job_id
		JOIN customers c ON j.customer_id = c.id
		WHERE j.status = 'Completed'
		  AND m.gross_profit < 0` + dateClause + `
		ORDER BY m.gross_profit ASC
		LIMIT 20
	`

	rows, err := db.QueryContext(ctx, query, dateArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RedFlagJob
	for rows.Next() {
		var r RedFlagJob
		var completionDate sql.NullTime
		err := rows.Scan(
			&r.JobID,
			&r.CustomerName,
			&r.JobType,
			&r.Revenue,
			&r.Costs,
			&r.Loss,
			&completionDate,
		)
		if err != nil {
			return nil, err
		}
		if completionDate.Valid {
			r.CompletionDate = &completionDate.Time
		}
		results = append(results, r)
	}

	return results, rows.Err()
}
