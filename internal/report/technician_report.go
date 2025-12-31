package report

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// TechnicianReport contains all data for the technician performance report
type TechnicianReport struct {
	GeneratedAt time.Time
	FromDate    *time.Time
	ToDate      *time.Time

	// Summary stats
	TotalTechnicians   int
	TotalJobsCompleted int
	TotalSales         float64
	AvgConversionRate  float64

	// Individual technician performance
	Technicians []TechnicianPerformance

	// Monthly trends (for charts/tables)
	MonthlyTrends []MonthlyTechTrend
}

// TechnicianPerformance represents metrics for a single technician
type TechnicianPerformance struct {
	Name               string
	TotalJobs          int     // Jobs as primary (opportunities)
	SoldJobs           int     // Jobs as sold_by (conversions)
	ConversionRate     float64 // SoldJobs / TotalJobs * 100
	TotalSales         float64
	AvgSale            float64
	TotalHoursWorked   float64
	AvgHoursPerJob     float64
	TotalEstimates     int
	AvgEstimatesPerJob float64
	TotalGrossProfit   float64
	AvgGrossProfit     float64
	AvgMarginPct       float64

	// Monthly breakdown for this technician
	MonthlyData []TechMonthData
}

// TechMonthData represents a technician's performance in a specific month
type TechMonthData struct {
	Month          string // "2024-11"
	MonthLabel     string // "Nov 2024"
	Jobs           int
	Sales          float64
	ConversionRate float64
}

// MonthlyTechTrend represents aggregate performance across all techs for a month
type MonthlyTechTrend struct {
	Month             string // "2024-11"
	MonthLabel        string // "Nov 2024"
	TotalJobs         int
	TotalSales        float64
	AvgConversionRate float64
	TopPerformer      string
	TopPerformerSales float64
}

// GenerateTechnicianReport builds the complete technician performance report
func GenerateTechnicianReport(ctx context.Context, db *sql.DB, fromDate, toDate *time.Time) (*TechnicianReport, error) {
	report := &TechnicianReport{
		GeneratedAt: time.Now(),
		FromDate:    fromDate,
		ToDate:      toDate,
	}

	var err error

	// Load technician performance
	report.Technicians, err = loadTechnicianPerformance(ctx, db, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("loading technician performance: %w", err)
	}

	// Calculate summary stats
	report.TotalTechnicians = len(report.Technicians)
	totalConvRate := 0.0
	techsWithJobs := 0
	for _, t := range report.Technicians {
		report.TotalJobsCompleted += t.TotalJobs
		report.TotalSales += t.TotalSales
		if t.TotalJobs > 0 {
			totalConvRate += t.ConversionRate
			techsWithJobs++
		}
	}
	if techsWithJobs > 0 {
		report.AvgConversionRate = totalConvRate / float64(techsWithJobs)
	}

	// Load monthly trends
	report.MonthlyTrends, err = loadMonthlyTrends(ctx, db, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("loading monthly trends: %w", err)
	}

	// Load monthly data for each technician
	for i := range report.Technicians {
		report.Technicians[i].MonthlyData, err = loadTechnicianMonthlyData(ctx, db, report.Technicians[i].Name, fromDate, toDate)
		if err != nil {
			return nil, fmt.Errorf("loading monthly data for %s: %w", report.Technicians[i].Name, err)
		}
	}

	return report, nil
}

func loadTechnicianPerformance(ctx context.Context, db *sql.DB, fromDate, toDate *time.Time) ([]TechnicianPerformance, error) {
	dateClause, dateArgs := buildTechDateClause(fromDate, toDate, 0)

	query := `
		WITH tech_jobs AS (
			SELECT 
				t.name,
				jt.role,
				j.id as job_id,
				j.jobs_subtotal,
				j.estimate_sales_subtotal,
				j.total_hours_worked,
				COALESCE(j.estimate_count, 0) as estimate_count,
				jm.gross_profit
			FROM technicians t
			JOIN job_technicians jt ON t.id = jt.technician_id
			JOIN jobs j ON jt.job_id = j.id
			LEFT JOIN job_metrics jm ON j.id = jm.job_id
			WHERE j.status = 'Completed'` + dateClause + `
		),
		tech_primary AS (
			SELECT 
				name,
				COUNT(DISTINCT job_id) as total_jobs,
				SUM(CASE 
					WHEN estimate_sales_subtotal > 0 THEN estimate_sales_subtotal
					ELSE 0
				END) as estimate_sales,
				SUM(total_hours_worked) as total_hours,
				SUM(estimate_count) as total_estimates
			FROM tech_jobs
			WHERE role = 'primary'
			GROUP BY name
		),
		tech_sold AS (
			SELECT 
				name,
				COUNT(DISTINCT job_id) as sold_jobs,
				SUM(gross_profit) as total_profit
			FROM tech_jobs
			WHERE role = 'sold_by'
			GROUP BY name
		),
		tech_same_visit AS (
			-- Jobs where tech is both primary AND sold_by with no estimate sales
			SELECT 
				tj1.name,
				SUM(tj1.jobs_subtotal) as same_visit_sales
			FROM tech_jobs tj1
			WHERE tj1.role = 'primary'
			  AND tj1.estimate_sales_subtotal = 0
			  AND EXISTS (
				SELECT 1 FROM tech_jobs tj2 
				WHERE tj2.name = tj1.name 
				  AND tj2.job_id = tj1.job_id 
				  AND tj2.role = 'sold_by'
			  )
			GROUP BY tj1.name
		)
		SELECT 
			COALESCE(p.name, s.name) as name,
			COALESCE(p.total_jobs, 0) as total_jobs,
			COALESCE(s.sold_jobs, 0) as sold_jobs,
			COALESCE(p.estimate_sales, 0) + COALESCE(sv.same_visit_sales, 0) as total_sales,
			COALESCE(p.total_hours, 0) as total_hours,
			COALESCE(p.total_estimates, 0) as total_estimates,
			COALESCE(s.total_profit, 0) as total_profit
		FROM tech_primary p
		FULL OUTER JOIN tech_sold s ON p.name = s.name
		LEFT JOIN tech_same_visit sv ON COALESCE(p.name, s.name) = sv.name
		WHERE COALESCE(p.total_jobs, 0) > 0 OR COALESCE(s.sold_jobs, 0) > 0
		ORDER BY total_sales DESC
	`

	rows, err := db.QueryContext(ctx, query, dateArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TechnicianPerformance
	for rows.Next() {
		var t TechnicianPerformance
		var totalSales, totalHours, totalProfit sql.NullFloat64
		var totalEstimates sql.NullInt64

		err := rows.Scan(
			&t.Name,
			&t.TotalJobs,
			&t.SoldJobs,
			&totalSales,
			&totalHours,
			&totalEstimates,
			&totalProfit,
		)
		if err != nil {
			return nil, err
		}

		if totalSales.Valid {
			t.TotalSales = totalSales.Float64
		}
		if totalHours.Valid {
			t.TotalHoursWorked = totalHours.Float64
		}
		if totalEstimates.Valid {
			t.TotalEstimates = int(totalEstimates.Int64)
		}
		if totalProfit.Valid {
			t.TotalGrossProfit = totalProfit.Float64
		}

		// Calculate derived metrics
		if t.TotalJobs > 0 {
			t.ConversionRate = float64(t.SoldJobs) / float64(t.TotalJobs) * 100
			t.AvgHoursPerJob = t.TotalHoursWorked / float64(t.TotalJobs)
			t.AvgEstimatesPerJob = float64(t.TotalEstimates) / float64(t.TotalJobs)
		}
		if t.SoldJobs > 0 {
			t.AvgSale = t.TotalSales / float64(t.SoldJobs)
			t.AvgGrossProfit = t.TotalGrossProfit / float64(t.SoldJobs)
			if t.TotalSales > 0 {
				t.AvgMarginPct = t.TotalGrossProfit / t.TotalSales * 100
			}
		}

		results = append(results, t)
	}

	return results, rows.Err()
}

func loadMonthlyTrends(ctx context.Context, db *sql.DB, fromDate, toDate *time.Time) ([]MonthlyTechTrend, error) {
	dateClause, dateArgs := buildTechDateClause(fromDate, toDate, 0)

	query := `
		WITH monthly_data AS (
			SELECT 
				TO_CHAR(j.job_completion_date, 'YYYY-MM') as month,
				t.name,
				jt.role,
				j.id as job_id,
				j.estimate_sales_subtotal,
				j.jobs_subtotal
			FROM technicians t
			JOIN job_technicians jt ON t.id = jt.technician_id
			JOIN jobs j ON jt.job_id = j.id
			WHERE j.status = 'Completed'
			  AND j.job_completion_date IS NOT NULL` + dateClause + `
		),
		monthly_primary AS (
			SELECT 
				month,
				name,
				COUNT(DISTINCT job_id) as jobs
			FROM monthly_data
			WHERE role = 'primary'
			GROUP BY month, name
		),
		monthly_sales AS (
			SELECT 
				month,
				name,
				SUM(CASE 
					WHEN estimate_sales_subtotal > 0 THEN estimate_sales_subtotal
					ELSE jobs_subtotal
				END) as sales
			FROM monthly_data
			WHERE role = 'primary'
			  AND (estimate_sales_subtotal > 0 OR 
				   EXISTS (SELECT 1 FROM monthly_data md2 
				           WHERE md2.month = monthly_data.month 
				             AND md2.name = monthly_data.name 
				             AND md2.job_id = monthly_data.job_id 
				             AND md2.role = 'sold_by'))
			GROUP BY month, name
		),
		monthly_sold AS (
			SELECT 
				month,
				name,
				COUNT(DISTINCT job_id) as sold_jobs
			FROM monthly_data
			WHERE role = 'sold_by'
			GROUP BY month, name
		),
		monthly_agg AS (
			SELECT 
				p.month,
				SUM(p.jobs) as total_jobs,
				SUM(COALESCE(s.sales, 0)) as total_sales,
				AVG(CASE WHEN p.jobs > 0 THEN COALESCE(so.sold_jobs, 0)::float / p.jobs * 100 END) as avg_conv_rate
			FROM monthly_primary p
			LEFT JOIN monthly_sales s ON p.month = s.month AND p.name = s.name
			LEFT JOIN monthly_sold so ON p.month = so.month AND p.name = so.name
			GROUP BY p.month
		),
		top_performers AS (
			SELECT DISTINCT ON (month)
				month,
				name as top_performer,
				sales as top_sales
			FROM monthly_sales
			ORDER BY month, sales DESC
		)
		SELECT 
			ma.month,
			ma.total_jobs,
			ma.total_sales,
			COALESCE(ma.avg_conv_rate, 0),
			COALESCE(tp.top_performer, ''),
			COALESCE(tp.top_sales, 0)
		FROM monthly_agg ma
		LEFT JOIN top_performers tp ON ma.month = tp.month
		ORDER BY ma.month
	`

	rows, err := db.QueryContext(ctx, query, dateArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MonthlyTechTrend
	for rows.Next() {
		var t MonthlyTechTrend
		err := rows.Scan(
			&t.Month,
			&t.TotalJobs,
			&t.TotalSales,
			&t.AvgConversionRate,
			&t.TopPerformer,
			&t.TopPerformerSales,
		)
		if err != nil {
			return nil, err
		}

		// Parse month for label
		if parsed, err := time.Parse("2006-01", t.Month); err == nil {
			t.MonthLabel = parsed.Format("Jan 2006")
		} else {
			t.MonthLabel = t.Month
		}

		results = append(results, t)
	}

	return results, rows.Err()
}

func loadTechnicianMonthlyData(ctx context.Context, db *sql.DB, techName string, fromDate, toDate *time.Time) ([]TechMonthData, error) {
	dateClause, dateArgs := buildTechDateClause(fromDate, toDate, 1) // offset 1 for tech name param

	query := `
		WITH monthly_data AS (
			SELECT 
				TO_CHAR(j.job_completion_date, 'YYYY-MM') as month,
				jt.role,
				j.id as job_id,
				j.estimate_sales_subtotal,
				j.jobs_subtotal
			FROM technicians t
			JOIN job_technicians jt ON t.id = jt.technician_id
			JOIN jobs j ON jt.job_id = j.id
			WHERE t.name = $1
			  AND j.status = 'Completed'
			  AND j.job_completion_date IS NOT NULL` + dateClause + `
		),
		monthly_primary AS (
			SELECT 
				month,
				COUNT(DISTINCT job_id) as jobs
			FROM monthly_data
			WHERE role = 'primary'
			GROUP BY month
		),
		monthly_sold AS (
			SELECT 
				month,
				COUNT(DISTINCT job_id) as sold_jobs
			FROM monthly_data
			WHERE role = 'sold_by'
			GROUP BY month
		),
		monthly_sales AS (
			SELECT 
				month,
				SUM(CASE 
					WHEN estimate_sales_subtotal > 0 THEN estimate_sales_subtotal
					ELSE jobs_subtotal
				END) as sales
			FROM monthly_data
			WHERE role = 'primary'
			  AND (estimate_sales_subtotal > 0 OR 
				   EXISTS (SELECT 1 FROM monthly_data md2 
				           WHERE md2.month = monthly_data.month 
				             AND md2.job_id = monthly_data.job_id 
				             AND md2.role = 'sold_by'))
			GROUP BY month
		)
		SELECT 
			p.month,
			p.jobs,
			COALESCE(s.sales, 0),
			CASE WHEN p.jobs > 0 THEN COALESCE(so.sold_jobs, 0)::float / p.jobs * 100 ELSE 0 END
		FROM monthly_primary p
		LEFT JOIN monthly_sales s ON p.month = s.month
		LEFT JOIN monthly_sold so ON p.month = so.month
		ORDER BY p.month
	`

	args := []interface{}{techName}
	args = append(args, dateArgs...)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TechMonthData
	for rows.Next() {
		var t TechMonthData
		err := rows.Scan(
			&t.Month,
			&t.Jobs,
			&t.Sales,
			&t.ConversionRate,
		)
		if err != nil {
			return nil, err
		}

		// Parse month for label
		if parsed, err := time.Parse("2006-01", t.Month); err == nil {
			t.MonthLabel = parsed.Format("Jan 2006")
		} else {
			t.MonthLabel = t.Month
		}

		results = append(results, t)
	}

	return results, rows.Err()
}

func buildTechDateClause(fromDate, toDate *time.Time, argOffset int) (string, []interface{}) {
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

// // RenderTechnicianReport renders the technician report to HTML
// func (r *Renderer) RenderTechnicianReport(w io.Writer, report *TechnicianReport) error {
// 	return r.templates.ExecuteTemplate(w, "technicians.html", report)
// }
