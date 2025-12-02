package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

// parseDateFlags extracts --from and --to flags from args
// Returns fromDate, toDate, and remaining args
func parseDateFlags(args []string) (*time.Time, *time.Time, []string) {
	var fromDate, toDate *time.Time
	var remainingArgs []string

	i := 0
	for i < len(args) {
		if args[i] == "--from" && i+1 < len(args) {
			if t, err := time.Parse("2006-01-02", args[i+1]); err == nil {
				fromDate = &t
			} else {
				fmt.Printf("Warning: invalid --from date '%s', expected YYYY-MM-DD\n", args[i+1])
			}
			i += 2
		} else if args[i] == "--to" && i+1 < len(args) {
			if t, err := time.Parse("2006-01-02", args[i+1]); err == nil {
				toDate = &t
			} else {
				fmt.Printf("Warning: invalid --to date '%s', expected YYYY-MM-DD\n", args[i+1])
			}
			i += 2
		} else {
			remainingArgs = append(remainingArgs, args[i])
			i++
		}
	}

	return fromDate, toDate, remainingArgs
}

// buildDateFilter returns SQL WHERE clause fragment and args for date filtering
func buildDateFilter(fromDate, toDate *time.Time, argOffset int) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if fromDate != nil {
		argOffset++
		conditions = append(conditions, fmt.Sprintf("j.job_completion_date >= $%d", argOffset))
		args = append(args, *fromDate)
	}

	if toDate != nil {
		argOffset++
		conditions = append(conditions, fmt.Sprintf("j.job_completion_date <= $%d", argOffset))
		args = append(args, *toDate)
	}

	if len(conditions) == 0 {
		return "", nil
	}

	clause := ""
	for _, cond := range conditions {
		clause += " AND " + cond
	}

	return clause, args
}

// printDateRange prints the date range being used for the report
func printDateRange(fromDate, toDate *time.Time) {
	if fromDate != nil || toDate != nil {
		fmt.Print("Date range: ")
		if fromDate != nil {
			fmt.Print(fromDate.Format("2006-01-02"))
		} else {
			fmt.Print("(all)")
		}
		fmt.Print(" to ")
		if toDate != nil {
			fmt.Print(toDate.Format("2006-01-02"))
		} else {
			fmt.Print("(all)")
		}
		fmt.Println()
		fmt.Println()
	}
}

func reportJobTypes(ctx context.Context, db *sql.DB, args []string) {
	fromDate, toDate, _ := parseDateFlags(args)
	dateClause, dateArgs := buildDateFilter(fromDate, toDate, 0)

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
	printDateRange(fromDate, toDate)
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
			marginStr = fmt.Sprintf("%7.1f%%", r.AvgMarginPct.Float64)
		}

		fmt.Printf("%-30s  %6d  $%11.2f  $%11.2f  $%11.2f  %8s  $%13.2f\n",
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

func reportCampaigns(ctx context.Context, db *sql.DB, args []string) {
	fromDate, toDate, _ := parseDateFlags(args)
	dateClause, dateArgs := buildDateFilter(fromDate, toDate, 0)

	query := `
		SELECT 
			COALESCE(j.campaign_name, 'Unknown') as campaign_name,
			COALESCE(j.campaign_category, 'Uncategorized') as campaign_category,
			COUNT(*) as job_count,
			AVG(m.revenue)::numeric(12,2) as avg_revenue,
			AVG(m.total_costs)::numeric(12,2) as avg_costs,
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
	printDateRange(fromDate, toDate)
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-25s  %-20s  %6s  %11s  %11s  %9s  %13s\n",
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
			marginStr = fmt.Sprintf("%7.1f%%", r.AvgMarginPct.Float64)
		}

		fmt.Printf("%-25s  %-20s  %6d  $%10.2f  %8s  $%12.2f  $%10.2f\n",
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
	fromDate, toDate, remainingArgs := parseDateFlags(args)
	dateClause, dateArgs := buildDateFilter(fromDate, toDate, 1) // offset by 1 for LIMIT param

	limit := 25 // default

	// Parse --top flag from remaining args
	for i, arg := range remainingArgs {
		if arg == "--top" && i+1 < len(remainingArgs) {
			if n, err := strconv.Atoi(remainingArgs[i+1]); err == nil {
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
			AVG(m.gross_margin_pct) FILTER (WHERE m.gross_margin_pct IS NOT NULL)::numeric(8,2) as avg_margin_pct,
			SUM(m.gross_profit)::numeric(12,2) as total_profit
		FROM customers c
		JOIN jobs j ON c.id = j.customer_id
		JOIN job_metrics m ON j.id = m.job_id
		WHERE j.status = 'Completed'` + dateClause + `
		GROUP BY c.id, c.customer_name, c.customer_type, c.location_zip
		ORDER BY total_profit DESC
		LIMIT $1
	`

	// Build args: LIMIT first, then date args
	queryArgs := []interface{}{limit}
	queryArgs = append(queryArgs, dateArgs...)

	rows, err := db.QueryContext(ctx, query, queryArgs...)
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
	printDateRange(fromDate, toDate)
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-35s  %6s  %11s  %11s  %9s  %13s\n",
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
			marginStr = fmt.Sprintf("%7.1f%%", r.AvgMarginPct.Float64)
		}

		fmt.Printf("%-35s  %6d  $%10.2f  %8s  $%12.2f  %s\n",
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
