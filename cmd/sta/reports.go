package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

func reportJobTypes(ctx context.Context, db *sql.DB) {
	query := `
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
		ORDER BY total_profit DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type JobTypeStats struct {
		JobType      string
		JobCount     int
		AvgRevenue   float64
		AvgCosts     float64
		AvgProfit    float64
		AvgMarginPct sql.NullFloat64
		TotalProfit  float64
	}

	var results []JobTypeStats
	for rows.Next() {
		var r JobTypeStats
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
		fmt.Println("No completed jobs with metrics found")
		return
	}

	fmt.Println("Profitability by Job Type")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-30s  %6s  %12s  %12s  %12s  %9s  %14s\n",
		"Job Type", "Jobs", "Avg Revenue", "Avg Costs", "Avg Profit", "Margin %", "Total Profit")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────")

	for _, r := range results {
		jobType := r.JobType
		if len(jobType) > 30 {
			jobType = jobType[:27] + "..."
		}

		marginStr := "N/A"
		if r.AvgMarginPct.Valid {
			marginStr = fmt.Sprintf("%8.1f%%", r.AvgMarginPct.Float64)
		}

		fmt.Printf("%-30s  %6d  $%11.2f  $%11.2f  $%11.2f  %9s  $%13.2f\n",
			jobType,
			r.JobCount,
			r.AvgRevenue,
			r.AvgCosts,
			r.AvgProfit,
			marginStr,
			r.TotalProfit,
		)
	}
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")

	// Calculate totals
	totalJobs := 0
	totalProfit := 0.0
	for _, r := range results {
		totalJobs += r.JobCount
		totalProfit += r.TotalProfit
	}
	avgProfit := totalProfit / float64(len(results))

	fmt.Printf("Total: %d job types, %d completed jobs, $%.2f total profit, $%.2f avg profit per type\n",
		len(results), totalJobs, totalProfit, avgProfit)
}

func reportCampaigns(ctx context.Context, db *sql.DB) {
	query := `
		SELECT 
			COALESCE(j.campaign_name, 'Unknown') as campaign_name,
			COALESCE(j.campaign_category, 'Uncategorized') as campaign_category,
			COUNT(*) as job_count,
			AVG(m.revenue)::numeric(12,2) as avg_revenue,
			AVG(m.total_costs)::numeric(12,2) as avg_costs,
			AVG(m.gross_profit)::numeric(12,2) as avg_gross_profit,
			AVG(m.gross_margin_pct)::numeric(8,2) as avg_margin_pct,
			SUM(m.gross_profit)::numeric(12,2) as total_profit
		FROM jobs j
		JOIN job_metrics m ON j.id = m.job_id
		WHERE j.status = 'Completed'
		GROUP BY j.campaign_name, j.campaign_category
		ORDER BY total_profit DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type CampaignStats struct {
		CampaignName     string
		CampaignCategory string
		JobCount         int
		AvgRevenue       float64
		AvgCosts         float64
		AvgProfit        float64
		AvgMarginPct     sql.NullFloat64
		TotalProfit      float64
	}

	var results []CampaignStats
	for rows.Next() {
		var r CampaignStats
		err := rows.Scan(
			&r.CampaignName,
			&r.CampaignCategory,
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
		fmt.Println("No completed jobs with campaign data found")
		return
	}

	fmt.Println("Profitability by Campaign")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-25s  %-20s  %6s  %11s  %9s  %13s  %11s\n",
		"Campaign", "Category", "Jobs", "Avg Profit", "Margin %", "Total Profit", "Avg Revenue")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────")

	for _, r := range results {
		campaign := r.CampaignName
		if len(campaign) > 25 {
			campaign = campaign[:22] + "..."
		}
		category := r.CampaignCategory
		if len(category) > 20 {
			category = category[:17] + "..."
		}

		marginStr := "N/A"
		if r.AvgMarginPct.Valid {
			marginStr = fmt.Sprintf("%8.1f%%", r.AvgMarginPct.Float64)
		}

		fmt.Printf("%-25s  %-20s  %6d  $%10.2f  %9s  $%12.2f  $%10.2f\n",
			campaign,
			category,
			r.JobCount,
			r.AvgProfit,
			marginStr,
			r.TotalProfit,
			r.AvgRevenue,
		)
	}
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")

	totalJobs := 0
	totalProfit := 0.0
	for _, r := range results {
		totalJobs += r.JobCount
		totalProfit += r.TotalProfit
	}

	fmt.Printf("Total: %d campaigns, %d completed jobs, $%.2f total profit\n",
		len(results), totalJobs, totalProfit)
}

func reportCustomers(ctx context.Context, db *sql.DB, args []string) {
	limit := 25 // default

	// Parse --top flag
	for i, arg := range args {
		if arg == "--top" && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil {
				limit = n
			}
		}
	}

	query := `
		SELECT 
			c.id as customer_id,
			c.customer_name,
			c.customer_type,
			c.location_zip,
			COUNT(j.id) as job_count,
			AVG(m.revenue)::numeric(12,2) as avg_revenue,
			AVG(m.gross_profit)::numeric(12,2) as avg_profit_per_job,
			AVG(m.gross_margin_pct)::numeric(8,2) as avg_margin_pct,
			SUM(m.gross_profit)::numeric(12,2) as total_profit
		FROM customers c
		JOIN jobs j ON c.id = j.customer_id
		JOIN job_metrics m ON j.id = m.job_id
		WHERE j.status = 'Completed'
		GROUP BY c.id, c.customer_name, c.customer_type, c.location_zip
		ORDER BY total_profit DESC
		LIMIT $1
	`

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type CustomerStats struct {
		CustomerID      int64
		CustomerName    string
		CustomerType    sql.NullString
		LocationZip     sql.NullString
		JobCount        int
		AvgRevenue      float64
		AvgProfitPerJob float64
		AvgMarginPct    sql.NullFloat64
		TotalProfit     float64
	}

	var results []CustomerStats
	for rows.Next() {
		var r CustomerStats
		err := rows.Scan(
			&r.CustomerID,
			&r.CustomerName,
			&r.CustomerType,
			&r.LocationZip,
			&r.JobCount,
			&r.AvgRevenue,
			&r.AvgProfitPerJob,
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
		fmt.Println("No customers with completed jobs found")
		return
	}

	fmt.Printf("Top %d Customers by Profit\n", limit)
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-35s  %6s  %11s  %9s  %13s  %s\n",
		"Customer", "Jobs", "Avg/Job", "Margin %", "Total Profit", "Type")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────")

	for i, r := range results {
		name := r.CustomerName
		if len(name) > 35 {
			name = name[:32] + "..."
		}

		custType := "Unknown"
		if r.CustomerType.Valid {
			custType = r.CustomerType.String
		}

		marginStr := "N/A"
		if r.AvgMarginPct.Valid {
			marginStr = fmt.Sprintf("%8.1f%%", r.AvgMarginPct.Float64)
		}

		fmt.Printf("%-35s  %6d  $%10.2f  %9s  $%12.2f  %s\n",
			name,
			r.JobCount,
			r.AvgProfitPerJob,
			marginStr,
			r.TotalProfit,
			custType,
		)

		// Add separator every 10 rows for readability
		if (i+1)%10 == 0 && i+1 < len(results) {
			fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")
		}
	}
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")

	totalJobs := 0
	totalProfit := 0.0
	for _, r := range results {
		totalJobs += r.JobCount
		totalProfit += r.TotalProfit
	}

	fmt.Printf("Showing top %d customers, %d total jobs, $%.2f total profit\n",
		len(results), totalJobs, totalProfit)

	// Calculate LTV if we have data
	if len(results) > 0 {
		avgLifetimeValue := totalProfit / float64(len(results))
		fmt.Printf("Average customer lifetime value: $%.2f\n", avgLifetimeValue)
	}
}
