package main

import (
	"context"
	"database/sql"
	"fmt"
)

func reportTechnicians(ctx context.Context, db *sql.DB, args []string) {
	fromDate, toDate, remainingArgs := parseDateFlags(args)

	// Note: Date filtering for technician reports would require re-calculating metrics
	// For now, show all-time metrics but print the date range if specified
	if fromDate != nil || toDate != nil {
		fmt.Println("Note: Technician metrics are calculated across all imported data.")
		fmt.Println("Date filtering will be supported in a future version.")
		fmt.Println()
	}
	_ = fromDate
	_ = toDate

	// Check for subcommand
	subcommand := "overview"
	if len(remainingArgs) > 0 {
		subcommand = remainingArgs[0]
	}

	switch subcommand {
	case "overview", "":
		reportTechnicianOverview(ctx, db)
	case "sales":
		reportTechnicianSales(ctx, db)
	case "conversion":
		reportTechnicianConversion(ctx, db)
	case "efficiency":
		reportTechnicianEfficiency(ctx, db)
	case "help":
		printTechnicianUsage()
	default:
		fmt.Printf("Unknown technician report type: %s\n", subcommand)
		printTechnicianUsage()
	}
}

func printTechnicianUsage() {
	fmt.Println(`Technician Performance Reports

Usage:
  sta report technicians [type]

Report Types:
  overview     All KPIs for each technician (default)
  sales        Ranked by average sale amount
  conversion   Ranked by conversion rate (min 5 opportunities)
  efficiency   Ranked by average hours per job (lower is better)

Examples:
  sta report technicians
  sta report technicians sales
  sta report technicians conversion`)
}

func reportTechnicianOverview(ctx context.Context, db *sql.DB) {
	query := `
		SELECT 
			t.name,
			COALESCE(tm.jobs_sold, 0) as jobs_sold,
			tm.avg_sale,
			tm.conversion_rate,
			COALESCE(tm.jobs_serviced, 0) as jobs_serviced,
			tm.avg_hours_per_job
		FROM technicians t
		JOIN technician_metrics tm ON t.id = tm.technician_id
		WHERE (tm.jobs_sold > 0 OR tm.jobs_serviced > 0)
		ORDER BY t.name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type TechOverview struct {
		Name             string
		JobsSold         int
		AvgSale          sql.NullFloat64
		ConversionRate   sql.NullFloat64
		JobsServiced     int
		AvgHoursPerJob   sql.NullFloat64
		AvgMarginPct     sql.NullFloat64
		TotalGrossProfit sql.NullFloat64
	}

	var results []TechOverview
	for rows.Next() {
		var r TechOverview
		err := rows.Scan(
			&r.Name,
			&r.JobsSold,
			&r.AvgSale,
			&r.ConversionRate,
			&r.JobsServiced,
			&r.AvgHoursPerJob,
		)
		if err != nil {
			fmt.Printf("Error reading results: %v\n", err)
			return
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Println("No technician data found")
		fmt.Println("Run 'sta import' with data that includes technician information")
		return
	}

	fmt.Println("Technician Performance Overview")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-25s  %6s  %11s  %10s  %8s  %10s\n",
		"Technician", "Sold", "Avg Sale", "Conv %", "Serviced", "Avg Hrs")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────")

	for _, r := range results {
		name := r.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		avgSale := "N/A"
		if r.AvgSale.Valid {
			avgSale = fmt.Sprintf("$%10.2f", r.AvgSale.Float64)
		}

		convRate := "N/A"
		if r.ConversionRate.Valid {
			convRate = fmt.Sprintf("%8.1f%%", r.ConversionRate.Float64)
		}

		avgHrs := "N/A"
		if r.AvgHoursPerJob.Valid {
			avgHrs = fmt.Sprintf("%8.1f", r.AvgHoursPerJob.Float64)
		}

		// marginPct := "N/A"
		// if r.AvgMarginPct.Valid {
		// 	marginPct = fmt.Sprintf("%7.1f%%", r.AvgMarginPct.Float64)
		// }

		// totalProfit := "N/A"
		// if r.TotalGrossProfit.Valid {
		// 	totalProfit = fmt.Sprintf("$%13.2f", r.TotalGrossProfit.Float64)
		// }

		fmt.Printf("%-25s  %6d  %11s  %10s  %8d  %10s\n",
			name,
			r.JobsSold,
			avgSale,
			convRate,
			r.JobsServiced,
			avgHrs,
			// marginPct,
			// totalProfit,
		)
	}
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("Total: %d technicians\n", len(results))
}

func reportTechnicianSales(ctx context.Context, db *sql.DB) {
	query := `
		SELECT 
			t.name,
			tm.jobs_sold,
			tm.total_sales,
			tm.avg_sale,
			tm.avg_margin_pct,
			tm.total_gross_profit
		FROM technicians t
		JOIN technician_metrics tm ON t.id = tm.technician_id
		WHERE tm.jobs_sold > 0
		ORDER BY tm.avg_sale DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type TechSales struct {
		Name             string
		JobsSold         int
		TotalSales       float64
		AvgSale          sql.NullFloat64
		AvgMarginPct     sql.NullFloat64
		TotalGrossProfit sql.NullFloat64
	}

	var results []TechSales
	for rows.Next() {
		var r TechSales
		err := rows.Scan(
			&r.Name,
			&r.JobsSold,
			&r.TotalSales,
			&r.AvgSale,
			&r.AvgMarginPct,
			&r.TotalGrossProfit,
		)
		if err != nil {
			fmt.Printf("Error reading results: %v\n", err)
			return
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Println("No sales data found")
		return
	}

	fmt.Println("Technician Sales Performance (Ranked by Avg Sale)")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-25s  %6s  %14s  %12s  %9s  %14s\n",
		"Technician", "Jobs", "Total Sales", "Avg Sale", "Margin %", "Total Profit")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────")

	for i, r := range results {
		name := r.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		avgSale := "N/A"
		if r.AvgSale.Valid {
			avgSale = fmt.Sprintf("$%11.2f", r.AvgSale.Float64)
		}

		marginPct := "N/A"
		if r.AvgMarginPct.Valid {
			marginPct = fmt.Sprintf("%7.1f%%", r.AvgMarginPct.Float64)
		}

		totalProfit := "N/A"
		if r.TotalGrossProfit.Valid {
			totalProfit = fmt.Sprintf("$%13.2f", r.TotalGrossProfit.Float64)
		}

		rank := "   "
		if i < 3 {
			medals := []string{"🥇 ", "🥈 ", "🥉 "}
			rank = medals[i]
		}

		fmt.Printf("%s%-22s  %6d  $%13.2f  %12s  %9s  %14s\n",
			rank,
			name,
			r.JobsSold,
			r.TotalSales,
			avgSale,
			marginPct,
			totalProfit,
		)
	}
	fmt.Println("════════════════════════════════════════════════════════════════════════════════════════════")
}

func reportTechnicianConversion(ctx context.Context, db *sql.DB) {
	query := `
		SELECT 
			t.name,
			tm.opportunities,
			tm.conversions,
			tm.conversion_rate,
			tm.avg_sale
		FROM technicians t
		JOIN technician_metrics tm ON t.id = tm.technician_id
		WHERE tm.opportunities >= 5
		ORDER BY tm.conversion_rate DESC NULLS LAST
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type TechConversion struct {
		Name           string
		Opportunities  int
		Conversions    int
		ConversionRate sql.NullFloat64
		AvgSale        sql.NullFloat64
	}

	var results []TechConversion
	for rows.Next() {
		var r TechConversion
		err := rows.Scan(
			&r.Name,
			&r.Opportunities,
			&r.Conversions,
			&r.ConversionRate,
			&r.AvgSale,
		)
		if err != nil {
			fmt.Printf("Error reading results: %v\n", err)
			return
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Println("No conversion data found (minimum 5 opportunities required)")
		return
	}

	fmt.Println("Technician Conversion Rates (Min 5 Opportunities)")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-25s  %12s  %11s  %12s  %12s\n",
		"Technician", "Opportunities", "Conversions", "Conv Rate", "Avg Sale")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")

	for i, r := range results {
		name := r.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		convRate := "N/A"
		if r.ConversionRate.Valid {
			convRate = fmt.Sprintf("%10.1f%%", r.ConversionRate.Float64)
		}

		avgSale := "N/A"
		if r.AvgSale.Valid {
			avgSale = fmt.Sprintf("$%11.2f", r.AvgSale.Float64)
		}

		rank := "   "
		if i < 3 {
			medals := []string{"🥇 ", "🥈 ", "🥉 "}
			rank = medals[i]
		}

		fmt.Printf("%s%-22s  %12d  %11d  %12s  %12s\n",
			rank,
			name,
			r.Opportunities,
			r.Conversions,
			convRate,
			avgSale,
		)
	}
	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
}

func reportTechnicianEfficiency(ctx context.Context, db *sql.DB) {
	query := `
		SELECT 
			t.name,
			tm.jobs_serviced,
			tm.total_hours_worked,
			tm.avg_hours_per_job,
			tm.avg_estimates_per_job
		FROM technicians t
		JOIN technician_metrics tm ON t.id = tm.technician_id
		WHERE tm.jobs_serviced > 0
		ORDER BY tm.avg_hours_per_job ASC NULLS LAST
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		fmt.Printf("Error running report: %v\n", err)
		return
	}
	defer rows.Close()

	type TechEfficiency struct {
		Name               string
		JobsServiced       int
		TotalHoursWorked   sql.NullFloat64
		AvgHoursPerJob     sql.NullFloat64
		AvgEstimatesPerJob sql.NullFloat64
	}

	var results []TechEfficiency
	for rows.Next() {
		var r TechEfficiency
		err := rows.Scan(
			&r.Name,
			&r.JobsServiced,
			&r.TotalHoursWorked,
			&r.AvgHoursPerJob,
			&r.AvgEstimatesPerJob,
		)
		if err != nil {
			fmt.Printf("Error reading results: %v\n", err)
			return
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Println("No efficiency data found")
		return
	}

	fmt.Println("Technician Efficiency (Ranked by Avg Hours - Lower is Better)")
	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-25s  %8s  %12s  %12s  %14s\n",
		"Technician", "Jobs", "Total Hours", "Avg Hrs/Job", "Avg Est/Job")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")

	for i, r := range results {
		name := r.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		totalHrs := "N/A"
		if r.TotalHoursWorked.Valid {
			totalHrs = fmt.Sprintf("%10.1f", r.TotalHoursWorked.Float64)
		}

		avgHrs := "N/A"
		if r.AvgHoursPerJob.Valid {
			avgHrs = fmt.Sprintf("%10.1f", r.AvgHoursPerJob.Float64)
		}

		avgEst := "N/A"
		if r.AvgEstimatesPerJob.Valid {
			avgEst = fmt.Sprintf("%12.1f", r.AvgEstimatesPerJob.Float64)
		}

		rank := "   "
		if i < 3 {
			medals := []string{"🥇 ", "🥈 ", "🥉 "}
			rank = medals[i]
		}

		fmt.Printf("%s%-22s  %8d  %12s  %12s  %14s\n",
			rank,
			name,
			r.JobsServiced,
			totalHrs,
			avgHrs,
			avgEst,
		)
	}
	fmt.Println("════════════════════════════════════════════════════════════════════════════════")
}
