package analytics

import (
	"context"
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// Service provides analytics functionality using raw database data
type Service struct {
	db *sql.DB
}

// NewService creates a new analytics service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// GetTechnicianKPIs calculates KPIs for all technicians (or one if specified)
func (s *Service) GetTechnicianKPIs(ctx context.Context, filter TechnicianFilter) ([]TechnicianKPIs, error) {
	// Fetch raw data
	jobs, err := s.fetchJobs(ctx, filter.DateRange, filter.BusinessUnit)
	if err != nil {
		return nil, err
	}

	jobTechs, err := s.fetchJobTechnicians(ctx)
	if err != nil {
		return nil, err
	}

	technicians, err := s.fetchTechnicians(ctx)
	if err != nil {
		return nil, err
	}

	// Create calculator and compute KPIs
	calc := NewCalculator(jobs, jobTechs, technicians)

	if filter.TechnicianID != nil {
		kpi := calc.CalculateTechnicianKPIs(*filter.TechnicianID)
		// Fill in technician name
		for _, t := range technicians {
			if t.ID == *filter.TechnicianID {
				kpi.TechnicianName = t.Name
				if t.BusinessUnit != nil {
					kpi.BusinessUnit = *t.BusinessUnit
				}
				break
			}
		}
		return []TechnicianKPIs{kpi}, nil
	}

	return calc.CalculateAllTechnicianKPIs(), nil
}

// GetTeamKPIs calculates KPIs aggregated by business unit
func (s *Service) GetTeamKPIs(ctx context.Context, filter TechnicianFilter) ([]TeamKPIs, error) {
	jobs, err := s.fetchJobs(ctx, filter.DateRange, filter.BusinessUnit)
	if err != nil {
		return nil, err
	}

	jobTechs, err := s.fetchJobTechnicians(ctx)
	if err != nil {
		return nil, err
	}

	technicians, err := s.fetchTechnicians(ctx)
	if err != nil {
		return nil, err
	}

	calc := NewCalculator(jobs, jobTechs, technicians)
	return calc.CalculateTeamKPIs(), nil
}

// GetTechnicianTrends calculates KPIs over time periods
func (s *Service) GetTechnicianTrends(ctx context.Context, dateRange DateRange, periodType PeriodType, technicianID *int64) ([]TechnicianTrend, error) {
	// Fetch all data within the range (trends will filter further)
	jobs, err := s.fetchJobs(ctx, &dateRange, nil)
	if err != nil {
		return nil, err
	}

	jobTechs, err := s.fetchJobTechnicians(ctx)
	if err != nil {
		return nil, err
	}

	technicians, err := s.fetchTechnicians(ctx)
	if err != nil {
		return nil, err
	}

	tc := NewTrendCalculator(jobs, jobTechs, technicians)
	return tc.CalculateTrends(dateRange, periodType, technicianID), nil
}

// GetYoYComparison calculates year-over-year comparison
func (s *Service) GetYoYComparison(ctx context.Context, currentPeriod DateRange, technicianID *int64) ([]YoYComparison, error) {
	// Calculate prior period
	priorPeriod := DateRange{
		From: currentPeriod.From.AddDate(-1, 0, 0),
		To:   currentPeriod.To.AddDate(-1, 0, 0),
	}

	// Fetch data for both periods
	extendedRange := DateRange{
		From: priorPeriod.From,
		To:   currentPeriod.To,
	}

	jobs, err := s.fetchJobs(ctx, &extendedRange, nil)
	if err != nil {
		return nil, err
	}

	jobTechs, err := s.fetchJobTechnicians(ctx)
	if err != nil {
		return nil, err
	}

	technicians, err := s.fetchTechnicians(ctx)
	if err != nil {
		return nil, err
	}

	tc := NewTrendCalculator(jobs, jobTechs, technicians)
	return tc.CalculateYoYComparison(currentPeriod, technicianID), nil
}

// GetDateBounds returns the earliest and latest job dates in the database
func (s *Service) GetDateBounds(ctx context.Context) (*time.Time, *time.Time, error) {
	query := `
		SELECT 
			MIN(job_completion_date) as earliest_date,
			MAX(job_completion_date) as latest_date
		FROM jobs
		WHERE job_completion_date IS NOT NULL
		  AND status = 'Completed'
	`

	var earliest, latest sql.NullTime
	err := s.db.QueryRowContext(ctx, query).Scan(&earliest, &latest)
	if err != nil {
		return nil, nil, err
	}

	var earliestPtr, latestPtr *time.Time
	if earliest.Valid {
		earliestPtr = &earliest.Time
	}
	if latest.Valid {
		latestPtr = &latest.Time
	}

	return earliestPtr, latestPtr, nil
}

// GetBusinessUnits returns all unique business units
func (s *Service) GetBusinessUnits(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT business_unit
		FROM jobs
		WHERE business_unit IS NOT NULL
		ORDER BY business_unit
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var units []string
	for rows.Next() {
		var unit string
		if err := rows.Scan(&unit); err != nil {
			return nil, err
		}
		units = append(units, unit)
	}

	return units, rows.Err()
}

// -----------------------------------------------------------------
// Private data fetching methods
// -----------------------------------------------------------------

func (s *Service) fetchJobs(ctx context.Context, dateRange *DateRange, businessUnit *string) ([]JobData, error) {
	var query string
	var args []interface{}

	if dateRange != nil && businessUnit != nil {
		query = `
			SELECT 
				id, project_id, status, job_completion_date, business_unit,
				total_hours_worked, COALESCE(estimate_count, 0), estimate_sales_subtotal,
				assigned_technician, warranty, warranty_for_job_id, recall, recall_for_job_id
			FROM jobs
			WHERE status = 'Completed'
			  AND job_completion_date >= $1
			  AND job_completion_date <= $2
			  AND business_unit = $3
			ORDER BY job_completion_date DESC
		`
		args = []interface{}{dateRange.From, dateRange.To, *businessUnit}
	} else if dateRange != nil {
		query = `
			SELECT 
				id, project_id, status, job_completion_date, business_unit,
				total_hours_worked, COALESCE(estimate_count, 0), estimate_sales_subtotal,
				assigned_technician, warranty, warranty_for_job_id, recall, recall_for_job_id
			FROM jobs
			WHERE status = 'Completed'
			  AND job_completion_date >= $1
			  AND job_completion_date <= $2
			ORDER BY job_completion_date DESC
		`
		args = []interface{}{dateRange.From, dateRange.To}
	} else if businessUnit != nil {
		query = `
			SELECT 
				id, project_id, status, job_completion_date, business_unit,
				total_hours_worked, COALESCE(estimate_count, 0), estimate_sales_subtotal,
				assigned_technician, warranty, warranty_for_job_id, recall, recall_for_job_id
			FROM jobs
			WHERE status = 'Completed'
			  AND business_unit = $1
			ORDER BY job_completion_date DESC
		`
		args = []interface{}{*businessUnit}
	} else {
		query = `
			SELECT 
				id, project_id, status, job_completion_date, business_unit,
				total_hours_worked, COALESCE(estimate_count, 0), estimate_sales_subtotal,
				assigned_technician, warranty, warranty_for_job_id, recall, recall_for_job_id
			FROM jobs
			WHERE status = 'Completed'
			ORDER BY job_completion_date DESC
		`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobData
	for rows.Next() {
		var job JobData
		var projectID, businessUnit, assignedTech, warrantyFor, recallFor sql.NullString
		var completionDate sql.NullTime
		var hoursWorked, estimateSales decimal.NullDecimal
		var estimateCount int

		err := rows.Scan(
			&job.ID,
			&projectID,
			&job.Status,
			&completionDate,
			&businessUnit,
			&hoursWorked,
			&estimateCount,
			&estimateSales,
			&assignedTech,
			&job.Warranty,
			&warrantyFor,
			&job.Recall,
			&recallFor,
		)
		if err != nil {
			return nil, err
		}

		if projectID.Valid {
			job.ProjectID = &projectID.String
		}
		if completionDate.Valid {
			job.JobCompletionDate = &completionDate.Time
		}
		if businessUnit.Valid {
			job.BusinessUnit = &businessUnit.String
		}
		if hoursWorked.Valid {
			job.TotalHoursWorked = hoursWorked.Decimal
		}
		job.EstimateCount = estimateCount
		if estimateSales.Valid {
			job.EstimateSalesSubtotal = estimateSales.Decimal
		}
		if assignedTech.Valid {
			job.AssignedTechnicians = &assignedTech.String
		}
		if warrantyFor.Valid {
			job.WarrantyForJobID = &warrantyFor.String
		}
		if recallFor.Valid {
			job.RecallForJobID = &recallFor.String
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (s *Service) fetchJobTechnicians(ctx context.Context) ([]JobTechnicianData, error) {
	query := `
		SELECT 
			jt.job_id,
			jt.technician_id,
			t.name as technician_name,
			jt.role
		FROM job_technicians jt
		JOIN technicians t ON t.id = jt.technician_id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobTechs []JobTechnicianData
	for rows.Next() {
		var jt JobTechnicianData
		err := rows.Scan(&jt.JobID, &jt.TechnicianID, &jt.TechnicianName, &jt.Role)
		if err != nil {
			return nil, err
		}
		jobTechs = append(jobTechs, jt)
	}

	return jobTechs, rows.Err()
}

func (s *Service) fetchTechnicians(ctx context.Context) ([]TechnicianData, error) {
	query := `
		SELECT 
			t.id,
			t.name,
			(
				SELECT j.business_unit 
				FROM job_technicians jt
				JOIN jobs j ON j.id = jt.job_id
				WHERE jt.technician_id = t.id 
				  AND j.business_unit IS NOT NULL
				GROUP BY j.business_unit
				ORDER BY COUNT(*) DESC
				LIMIT 1
			) as business_unit
		FROM technicians t
		ORDER BY t.name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var technicians []TechnicianData
	for rows.Next() {
		var t TechnicianData
		var businessUnit sql.NullString

		err := rows.Scan(&t.ID, &t.Name, &businessUnit)
		if err != nil {
			return nil, err
		}

		if businessUnit.Valid {
			t.BusinessUnit = &businessUnit.String
		}

		technicians = append(technicians, t)
	}

	return technicians, rows.Err()
}
