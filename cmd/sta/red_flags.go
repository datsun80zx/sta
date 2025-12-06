package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

// parseMarginThreshold extracts --margin-threshold flag from args
func parseMarginThreshold(args []string, defaultVal float64) (float64, []string) {
	threshold := defaultVal
	var remainingArgs []string

	i := 0
	for i < len(args) {
		if args[i] == "--margin-threshold" && i+1 < len(args) {
			if val, err := strconv.ParseFloat(args[i+1], 64); err == nil {
				threshold = val
			} else {
				fmt.Printf("Warning: invalid --margin-threshold '%s', using default %.1f%%\n", args[i+1], defaultVal)
			}
			i += 2
		} else {
			remainingArgs = append(remainingArgs, args[i])
			i++
		}
	}

	return threshold, remainingArgs
}

// redFlagsJobs shows individual jobs with negative margins
func redFlagsJobs(ctx context.Context, db *sql.DB, args []string) {
	fromDate, toDate, _ := parseDateFlags(args)
	dateClause, dateArgs := buildDateFilter(fromDate, toDate, 0)

	query := `
		SELECT 
			j.id as job_id,
			c.customer_name,
			j.job_type,
			m.revenue,
			m.total_costs,
			m.gross_profit,
			m.gross_margin_pct,
			j.job_completion_date
		FROM jobs j
		JOIN job_metrics m ON j.id = m.job_id
		JOIN customers c ON j.customer_id = c.id
		WHERE j.status = 'Completed'
		  AND m.gross_profit < 0` + dateClause + `
		ORDER BY m.gross_profit ASC
	`

	rows, err := db.QueryContext(ctx, query, dateArgs...)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type NegativeJob struct {
		JobID          string
		CustomerName   string
		JobType        string
		Revenue        float64
		TotalCosts     float64
		GrossProfit    float64
		GrossMarginPct sql.NullFloat64
		CompletionDate sql.NullTime
	}

	var results []NegativeJob
	for rows.Next() {
		var r NegativeJob
		err := rows.Scan(
			&r.JobID,
			&r.CustomerName,
			&r.JobType,
			&r.Revenue,
			&r.TotalCosts,
			&r.GrossProfit,
			&r.GrossMarginPct,
			&r.CompletionDate,
		)
		if err != nil {
			fmt.Printf("Error reading results: %v\n", err)
			return
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Println("✅ No jobs with negative margins found")
		printDateRange(fromDate, toDate)
		return
	}

	fmt.Println("🚩 RED FLAG: Jobs with Negative Margins")
	printDateRange(fromDate, toDate)
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-12s  %-25s  %-25s  %11s  %11s  %12s  %10s\n",
		"Job ID", "Customer", "Job Type", "Revenue", "Costs", "Loss", "Date")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────")

	totalLoss := 0.0
	for _, r := range results {
		customerName := r.CustomerName
		if len(customerName) > 25 {
			customerName = customerName[:22] + "..."
		}

		jobType := r.JobType
		if len(jobType) > 25 {
			jobType = jobType[:22] + "..."
		}

		dateStr := "N/A"
		if r.CompletionDate.Valid {
			dateStr = r.CompletionDate.Time.Format("2006-01-02")
		}

		fmt.Printf("%-12s  %-25s  %-25s  $%10.2f  $%10.2f  $%11.2f  %10s\n",
			r.JobID,
			customerName,
			jobType,
			r.Revenue,
			r.TotalCosts,
			r.GrossProfit,
			dateStr,
		)

		totalLoss += r.GrossProfit
	}

	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("⚠️  You lost money on %d jobs totaling $%.2f\n", len(results), -totalLoss)
}

// redFlagsJobTypes shows job types with average margin below threshold
func redFlagsJobTypes(ctx context.Context, db *sql.DB, args []string) {
	threshold, remainingArgs := parseMarginThreshold(args, 10.0)
	fromDate, toDate, _ := parseDateFlags(remainingArgs)
	dateClause, dateArgs := buildDateFilter(fromDate, toDate, 1) // offset by 1 for threshold param

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
		HAVING AVG(m.gross_margin_pct) FILTER (WHERE m.gross_margin_pct IS NOT NULL) < $1
		   OR AVG(m.gross_margin_pct) FILTER (WHERE m.gross_margin_pct IS NOT NULL) IS NULL
		ORDER BY AVG(m.gross_margin_pct) FILTER (WHERE m.gross_margin_pct IS NOT NULL) ASC NULLS FIRST
	`

	// Build args: threshold first, then date args
	queryArgs := []interface{}{threshold}
	queryArgs = append(queryArgs, dateArgs...)

	rows, err := db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type LowMarginJobType struct {
		JobType      string
		JobCount     int
		AvgRevenue   float64
		AvgCosts     float64
		AvgProfit    float64
		AvgMarginPct sql.NullFloat64
		TotalProfit  float64
	}

	var results []LowMarginJobType
	for rows.Next() {
		var r LowMarginJobType
		err := rows.Scan(
			&r.JobType,
			&r.JobCount,
			&r.AvgRevenue,
			&r.AvgCosts,
			&r.AvgProfit,
			&r.AvgMarginPct,
			&r.TotalProfit,
		)
		if err != nil {
			fmt.Printf("Error reading results: %v\n", err)
			return
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Printf("✅ No job types with average margin below %.1f%% found\n", threshold)
		printDateRange(fromDate, toDate)
		return
	}

	fmt.Printf("🚩 RED FLAG: Job Types with Average Margin Below %.1f%%\n", threshold)
	printDateRange(fromDate, toDate)
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-35s  %6s  %11s  %11s  %9s  %13s\n",
		"Job Type", "Jobs", "Avg Revenue", "Avg Profit", "Margin %", "Total Profit")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────")

	totalJobs := 0
	totalLoss := 0.0
	for _, r := range results {
		jobType := r.JobType
		if len(jobType) > 35 {
			jobType = jobType[:32] + "..."
		}

		marginStr := "N/A"
		if r.AvgMarginPct.Valid {
			marginStr = fmt.Sprintf("%7.1f%%", r.AvgMarginPct.Float64)
		}

		fmt.Printf("%-35s  %6d  $%10.2f  $%10.2f  %9s  $%12.2f\n",
			jobType,
			r.JobCount,
			r.AvgRevenue,
			r.AvgProfit,
			marginStr,
			r.TotalProfit,
		)

		totalJobs += r.JobCount
		if r.TotalProfit < 0 {
			totalLoss += r.TotalProfit
		}
	}

	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("⚠️  %d job types below %.1f%% margin threshold, affecting %d jobs\n",
		len(results), threshold, totalJobs)
	if totalLoss < 0 {
		fmt.Printf("   Total losses from unprofitable job types: $%.2f\n", -totalLoss)
	}
	fmt.Println("\n💡 Consider reviewing pricing or discontinuing these service types")
}

// redFlagsCustomers shows customers with negative total margin
func redFlagsCustomers(ctx context.Context, db *sql.DB, args []string) {
	fromDate, toDate, _ := parseDateFlags(args)
	dateClause, dateArgs := buildDateFilter(fromDate, toDate, 0)

	query := `
		SELECT 
			c.id as customer_id,
			c.customer_name,
			c.customer_type,
			COUNT(j.id) as job_count,
			SUM(m.revenue)::numeric(12,2) as total_revenue,
			SUM(m.total_costs)::numeric(12,2) as total_costs,
			SUM(m.gross_profit)::numeric(12,2) as total_profit,
			MIN(j.job_completion_date) as first_job,
			MAX(j.job_completion_date) as last_job
		FROM customers c
		JOIN jobs j ON c.id = j.customer_id
		JOIN job_metrics m ON j.id = m.job_id
		WHERE j.status = 'Completed'` + dateClause + `
		GROUP BY c.id, c.customer_name, c.customer_type
		HAVING SUM(m.gross_profit) < 0
		ORDER BY SUM(m.gross_profit) ASC
	`

	rows, err := db.QueryContext(ctx, query, dateArgs...)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type UnprofitableCustomer struct {
		CustomerID   int64
		CustomerName string
		CustomerType sql.NullString
		JobCount     int
		TotalRevenue float64
		TotalCosts   float64
		TotalProfit  float64
		FirstJob     sql.NullTime
		LastJob      sql.NullTime
	}

	var results []UnprofitableCustomer
	for rows.Next() {
		var r UnprofitableCustomer
		err := rows.Scan(
			&r.CustomerID,
			&r.CustomerName,
			&r.CustomerType,
			&r.JobCount,
			&r.TotalRevenue,
			&r.TotalCosts,
			&r.TotalProfit,
			&r.FirstJob,
			&r.LastJob,
		)
		if err != nil {
			fmt.Printf("Error reading results: %v\n", err)
			return
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Println("✅ No customers with negative total margin found")
		printDateRange(fromDate, toDate)
		return
	}

	fmt.Println("🚩 RED FLAG: Customers with Negative Total Margin")
	printDateRange(fromDate, toDate)
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-30s  %6s  %12s  %12s  %12s  %10s  %10s\n",
		"Customer", "Jobs", "Revenue", "Costs", "Total Loss", "First Job", "Last Job")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────")

	totalLoss := 0.0
	totalJobs := 0
	for _, r := range results {
		customerName := r.CustomerName
		if len(customerName) > 30 {
			customerName = customerName[:27] + "..."
		}

		firstJob := "N/A"
		if r.FirstJob.Valid {
			firstJob = r.FirstJob.Time.Format("2006-01-02")
		}

		lastJob := "N/A"
		if r.LastJob.Valid {
			lastJob = r.LastJob.Time.Format("2006-01-02")
		}

		fmt.Printf("%-30s  %6d  $%11.2f  $%11.2f  $%11.2f  %10s  %10s\n",
			customerName,
			r.JobCount,
			r.TotalRevenue,
			r.TotalCosts,
			r.TotalProfit,
			firstJob,
			lastJob,
		)

		totalLoss += r.TotalProfit
		totalJobs += r.JobCount
	}

	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("⚠️  %d customers cost you $%.2f total across %d jobs\n",
		len(results), -totalLoss, totalJobs)
	fmt.Println("\n💡 Consider reviewing pricing for these customers or ending the relationship")
}

// redFlagsHighRevenue shows jobs with high revenue but low margin
func redFlagsHighRevenue(ctx context.Context, db *sql.DB, args []string) {
	marginThreshold, remainingArgs := parseMarginThreshold(args, 15.0)
	fromDate, toDate, _ := parseDateFlags(remainingArgs)
	dateClause, dateArgs := buildDateFilter(fromDate, toDate, 2) // offset by 2 for revenue and margin params

	revenueThreshold := 2000.0

	query := `
		SELECT 
			j.id as job_id,
			c.customer_name,
			j.job_type,
			m.revenue,
			m.total_costs,
			m.gross_profit,
			m.gross_margin_pct,
			j.job_completion_date
		FROM jobs j
		JOIN job_metrics m ON j.id = m.job_id
		JOIN customers c ON j.customer_id = c.id
		WHERE j.status = 'Completed'
		  AND m.revenue > $1
		  AND (m.gross_margin_pct < $2 OR m.gross_margin_pct IS NULL)` + dateClause + `
		ORDER BY m.revenue DESC
	`

	// Build args: revenue threshold, margin threshold, then date args
	queryArgs := []interface{}{revenueThreshold, marginThreshold}
	queryArgs = append(queryArgs, dateArgs...)

	rows, err := db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type HighRevenueJob struct {
		JobID          string
		CustomerName   string
		JobType        string
		Revenue        float64
		TotalCosts     float64
		GrossProfit    float64
		GrossMarginPct sql.NullFloat64
		CompletionDate sql.NullTime
	}

	var results []HighRevenueJob
	for rows.Next() {
		var r HighRevenueJob
		err := rows.Scan(
			&r.JobID,
			&r.CustomerName,
			&r.JobType,
			&r.Revenue,
			&r.TotalCosts,
			&r.GrossProfit,
			&r.GrossMarginPct,
			&r.CompletionDate,
		)
		if err != nil {
			fmt.Printf("Error reading results: %v\n", err)
			return
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Printf("✅ No high-revenue jobs (>$%.0f) with margin below %.1f%% found\n",
			revenueThreshold, marginThreshold)
		printDateRange(fromDate, toDate)
		return
	}

	fmt.Printf("🚩 RED FLAG: High Revenue Jobs (>$%.0f) with Margin Below %.1f%%\n",
		revenueThreshold, marginThreshold)
	printDateRange(fromDate, toDate)
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-12s  %-25s  %-20s  %11s  %11s  %9s  %10s\n",
		"Job ID", "Customer", "Job Type", "Revenue", "Profit", "Margin %", "Date")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────")

	totalRevenue := 0.0
	totalProfit := 0.0
	for _, r := range results {
		customerName := r.CustomerName
		if len(customerName) > 25 {
			customerName = customerName[:22] + "..."
		}

		jobType := r.JobType
		if len(jobType) > 20 {
			jobType = jobType[:17] + "..."
		}

		marginStr := "N/A"
		if r.GrossMarginPct.Valid {
			marginStr = fmt.Sprintf("%7.1f%%", r.GrossMarginPct.Float64)
		}

		dateStr := "N/A"
		if r.CompletionDate.Valid {
			dateStr = r.CompletionDate.Time.Format("2006-01-02")
		}

		fmt.Printf("%-12s  %-25s  %-20s  $%10.2f  $%10.2f  %9s  %10s\n",
			r.JobID,
			customerName,
			jobType,
			r.Revenue,
			r.GrossProfit,
			marginStr,
			dateStr,
		)

		totalRevenue += r.Revenue
		totalProfit += r.GrossProfit
	}

	avgMargin := 0.0
	if totalRevenue > 0 {
		avgMargin = (totalProfit / totalRevenue) * 100
	}

	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("⚠️  %d high-revenue jobs with low margins\n", len(results))
	fmt.Printf("   Total revenue: $%.2f | Total profit: $%.2f | Average margin: %.1f%%\n",
		totalRevenue, totalProfit, avgMargin)
	fmt.Println("\n💡 You're busy but not maximizing profit on these large jobs - review pricing")
}

// handleRedFlags routes to the appropriate red flag subcommand
func handleRedFlags(ctx context.Context, db *sql.DB, args []string) {
	if len(args) < 1 {
		printRedFlagsUsage()
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "jobs":
		redFlagsJobs(ctx, db, subArgs)
	case "job-types":
		redFlagsJobTypes(ctx, db, subArgs)
	case "customers":
		redFlagsCustomers(ctx, db, subArgs)
	case "high-revenue":
		redFlagsHighRevenue(ctx, db, subArgs)
	case "help", "-h", "--help":
		printRedFlagsUsage()
	default:
		fmt.Printf("Unknown red-flags subcommand: %s\n\n", subcommand)
		printRedFlagsUsage()
	}
}

func printRedFlagsUsage() {
	fmt.Println(`Red Flags Reports - Identify profitability problems

Usage:
  sta report red-flags <type> [options]

Report Types:
  jobs          Individual jobs with negative margins
  job-types     Job types averaging below margin threshold
  customers     Customers with negative total margin
  high-revenue  High revenue jobs with low margins

Options:
  --from YYYY-MM-DD        Filter jobs completed on or after date
  --to YYYY-MM-DD          Filter jobs completed on or before date
  --margin-threshold N     Set margin % threshold (default: 10 for job-types, 15 for high-revenue)

Examples:
  sta report red-flags jobs
  sta report red-flags job-types --margin-threshold 15
  sta report red-flags customers --from 2024-11-01
  sta report red-flags high-revenue --from 2024-11-01 --to 2025-03-31`)
}
