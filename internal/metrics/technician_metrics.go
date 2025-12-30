package metrics

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"
)

// TechnicianMetric represents calculated metrics for a single technician
type TechnicianMetric struct {
	TechnicianID int64

	// Opportunity metrics (primary role - jobs they ran)
	TotalJobs int // Jobs where tech was primary (opportunities to sell)

	// Sales metrics (sold_by role - jobs they sold)
	SoldJobs   int             // Jobs where tech was sold_by (conversions)
	TotalSales decimal.Decimal // Sum of what they sold

	// Calculated metrics
	AvgSale        decimal.NullDecimal // TotalSales / SoldJobs
	ConversionRate decimal.NullDecimal // SoldJobs / TotalJobs * 100

	// Service metrics (primary role)
	TotalHoursWorked   decimal.Decimal
	AvgHoursPerJob     decimal.NullDecimal
	TotalEstimates     int
	AvgEstimatesPerJob decimal.NullDecimal

	// Profitability (sold_by role, from job_metrics)
	TotalGrossProfit decimal.NullDecimal
	AvgGrossProfit   decimal.NullDecimal
	AvgMarginPct     decimal.NullDecimal
}

// JobTechnicianData holds job_technician relationship data
type JobTechnicianData struct {
	JobID        string
	TechnicianID int64
	Role         string // "assigned", "sold_by", "primary"
}

// JobForTechMetrics holds job fields needed for technician calculations
type JobForTechMetrics struct {
	ID                    string
	Status                string
	JobsSubtotal          decimal.Decimal
	EstimateSalesSubtotal decimal.Decimal // What was sold via estimates
	TotalHoursWorked      decimal.Decimal
	EstimateCount         int
}

// CalculateTechnicianMetrics computes performance metrics for all technicians
func CalculateTechnicianMetrics(
	technicianIDs []int64,
	jobTechnicians []JobTechnicianData,
	jobs []JobForTechMetrics,
	jobMetrics []JobMetric,
) []TechnicianMetric {

	// Build lookup maps
	jobsByID := make(map[string]JobForTechMetrics)
	for _, j := range jobs {
		jobsByID[j.ID] = j
	}

	jobMetricsByID := make(map[string]JobMetric)
	for _, m := range jobMetrics {
		jobMetricsByID[m.JobID] = m
	}

	// Build a map of job_id -> set of roles for each technician
	// This helps us know if a tech is both primary AND sold_by on the same job
	type techJobRoles struct {
		isPrimary bool
		isSoldBy  bool
	}
	techJobRolesMap := make(map[int64]map[string]*techJobRoles) // tech_id -> job_id -> roles

	for _, jt := range jobTechnicians {
		if techJobRolesMap[jt.TechnicianID] == nil {
			techJobRolesMap[jt.TechnicianID] = make(map[string]*techJobRoles)
		}
		if techJobRolesMap[jt.TechnicianID][jt.JobID] == nil {
			techJobRolesMap[jt.TechnicianID][jt.JobID] = &techJobRoles{}
		}
		switch jt.Role {
		case "primary":
			techJobRolesMap[jt.TechnicianID][jt.JobID].isPrimary = true
		case "sold_by":
			techJobRolesMap[jt.TechnicianID][jt.JobID].isSoldBy = true
		}
	}

	// Initialize metrics for each technician
	metricsMap := make(map[int64]*TechnicianMetric)
	for _, techID := range technicianIDs {
		metricsMap[techID] = &TechnicianMetric{
			TechnicianID: techID,
			TotalSales:   decimal.Zero,
		}
	}

	// Process each job-technician relationship
	for _, jt := range jobTechnicians {
		job, jobExists := jobsByID[jt.JobID]
		if !jobExists {
			continue
		}

		// Only count completed jobs
		if job.Status != "Completed" {
			continue
		}

		m := metricsMap[jt.TechnicianID]
		if m == nil {
			continue
		}

		roles := techJobRolesMap[jt.TechnicianID][jt.JobID]

		switch jt.Role {
		case "sold_by":
			// Count as a conversion (they sold this job)
			m.SoldJobs++

			// Add to profitability if we have job metrics
			if jobMetric, exists := jobMetricsByID[jt.JobID]; exists {
				if !m.TotalGrossProfit.Valid {
					m.TotalGrossProfit = decimal.NullDecimal{Decimal: decimal.Zero, Valid: true}
				}
				m.TotalGrossProfit.Decimal = m.TotalGrossProfit.Decimal.Add(jobMetric.GrossProfit)
			}

		case "primary":
			// Count as an opportunity (they ran this job, had chance to sell)
			m.TotalJobs++

			// Service metrics
			if !job.TotalHoursWorked.IsZero() {
				m.TotalHoursWorked = m.TotalHoursWorked.Add(job.TotalHoursWorked)
			}
			m.TotalEstimates += job.EstimateCount

			// Sales calculation:
			// If they have estimate sales subtotal > 0, use that (they sold an estimate)
			// Else if they are also sold_by on this same job, use jobs subtotal (same-visit sell+install)
			if job.EstimateSalesSubtotal.GreaterThan(decimal.Zero) {
				m.TotalSales = m.TotalSales.Add(job.EstimateSalesSubtotal)
			} else if roles != nil && roles.isSoldBy && job.JobsSubtotal.GreaterThan(decimal.Zero) {
				// Same-visit sell+install scenario
				m.TotalSales = m.TotalSales.Add(job.JobsSubtotal)
			}
		}
	}

	// Calculate averages and build result slice
	var results []TechnicianMetric
	for _, m := range metricsMap {
		calculateTechnicianAverages(m)
		results = append(results, *m)
	}

	return results
}

func calculateTechnicianAverages(m *TechnicianMetric) {
	// Conversion rate = SoldJobs / TotalJobs * 100
	if m.TotalJobs > 0 {
		rate := decimal.NewFromInt(int64(m.SoldJobs)).
			Div(decimal.NewFromInt(int64(m.TotalJobs))).
			Mul(decimal.NewFromInt(100))
		m.ConversionRate = decimal.NullDecimal{
			Decimal: rate,
			Valid:   true,
		}
	}

	// Average sale = TotalSales / SoldJobs
	if m.SoldJobs > 0 && m.TotalSales.GreaterThan(decimal.Zero) {
		m.AvgSale = decimal.NullDecimal{
			Decimal: m.TotalSales.Div(decimal.NewFromInt(int64(m.SoldJobs))),
			Valid:   true,
		}
	}

	// Average hours per job
	if m.TotalJobs > 0 && !m.TotalHoursWorked.IsZero() {
		m.AvgHoursPerJob = decimal.NullDecimal{
			Decimal: m.TotalHoursWorked.Div(decimal.NewFromInt(int64(m.TotalJobs))),
			Valid:   true,
		}
	}

	// Average estimates per job
	if m.TotalJobs > 0 {
		m.AvgEstimatesPerJob = decimal.NullDecimal{
			Decimal: decimal.NewFromInt(int64(m.TotalEstimates)).Div(decimal.NewFromInt(int64(m.TotalJobs))),
			Valid:   true,
		}
	}

	// Average gross profit and margin
	if m.SoldJobs > 0 && m.TotalGrossProfit.Valid {
		m.AvgGrossProfit = decimal.NullDecimal{
			Decimal: m.TotalGrossProfit.Decimal.Div(decimal.NewFromInt(int64(m.SoldJobs))),
			Valid:   true,
		}

		if m.TotalSales.GreaterThan(decimal.Zero) {
			m.AvgMarginPct = decimal.NullDecimal{
				Decimal: m.TotalGrossProfit.Decimal.Div(m.TotalSales).Mul(decimal.NewFromInt(100)),
				Valid:   true,
			}
		}
	}
}

// SaveTechnicianMetrics persists calculated technician metrics to the database
func SaveTechnicianMetrics(ctx context.Context, tx *sql.Tx, metrics []TechnicianMetric) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO technician_metrics (
			technician_id,
			jobs_sold, total_sales, avg_sale,
			opportunities, conversions, conversion_rate,
			jobs_serviced, total_hours_worked, avg_hours_per_job,
			total_estimates, jobs_with_estimates, avg_estimates_per_job,
			total_gross_profit, avg_gross_profit, avg_margin_pct
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (technician_id) DO UPDATE SET
			jobs_sold = EXCLUDED.jobs_sold,
			total_sales = EXCLUDED.total_sales,
			avg_sale = EXCLUDED.avg_sale,
			opportunities = EXCLUDED.opportunities,
			conversions = EXCLUDED.conversions,
			conversion_rate = EXCLUDED.conversion_rate,
			jobs_serviced = EXCLUDED.jobs_serviced,
			total_hours_worked = EXCLUDED.total_hours_worked,
			avg_hours_per_job = EXCLUDED.avg_hours_per_job,
			total_estimates = EXCLUDED.total_estimates,
			jobs_with_estimates = EXCLUDED.jobs_with_estimates,
			avg_estimates_per_job = EXCLUDED.avg_estimates_per_job,
			total_gross_profit = EXCLUDED.total_gross_profit,
			avg_gross_profit = EXCLUDED.avg_gross_profit,
			avg_margin_pct = EXCLUDED.avg_margin_pct,
			calculated_at = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range metrics {
		_, err := stmt.ExecContext(ctx,
			m.TechnicianID,
			m.SoldJobs,                        // jobs_sold = conversions
			m.TotalSales,                      // total_sales
			nullableDecimal(m.AvgSale),        // avg_sale
			m.TotalJobs,                       // opportunities = total jobs as primary
			m.SoldJobs,                        // conversions = jobs as sold_by
			nullableDecimal(m.ConversionRate), // conversion_rate
			m.TotalJobs,                       // jobs_serviced = same as opportunities
			m.TotalHoursWorked,                // total_hours_worked
			nullableDecimal(m.AvgHoursPerJob), // avg_hours_per_job
			m.TotalEstimates,                  // total_estimates
			m.TotalEstimates,                  // jobs_with_estimates (simplified)
			nullableDecimal(m.AvgEstimatesPerJob),
			nullableDecimal(m.TotalGrossProfit),
			nullableDecimal(m.AvgGrossProfit),
			nullableDecimal(m.AvgMarginPct),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func nullableDecimal(d decimal.NullDecimal) interface{} {
	if d.Valid {
		return d.Decimal
	}
	return nil
}
