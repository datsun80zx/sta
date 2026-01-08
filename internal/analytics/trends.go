package analytics

import (
	"fmt"
	"sort"
	"time"
)

// PeriodType defines the granularity of trend analysis
type PeriodType string

const (
	PeriodWeekly    PeriodType = "weekly"
	PeriodMonthly   PeriodType = "monthly"
	PeriodQuarterly PeriodType = "quarterly"
	PeriodYearly    PeriodType = "yearly"
)

// TrendCalculator builds trend data from raw data
type TrendCalculator struct {
	jobs     []JobData
	jobTechs []JobTechnicianData
	techs    []TechnicianData
}

// NewTrendCalculator creates a new trend calculator
func NewTrendCalculator(jobs []JobData, jobTechs []JobTechnicianData, technicians []TechnicianData) *TrendCalculator {
	return &TrendCalculator{
		jobs:     jobs,
		jobTechs: jobTechs,
		techs:    technicians,
	}
}

// CalculateTrends computes KPIs for each period within the date range
func (tc *TrendCalculator) CalculateTrends(dateRange DateRange, periodType PeriodType, technicianID *int64) []TechnicianTrend {
	periods := tc.generatePeriods(dateRange, periodType)

	// If specific technician requested, only calculate for them
	var techIDs []int64
	if technicianID != nil {
		techIDs = []int64{*technicianID}
	} else {
		for _, t := range tc.techs {
			techIDs = append(techIDs, t.ID)
		}
	}

	var results []TechnicianTrend

	for _, techID := range techIDs {
		trend := TechnicianTrend{
			TechnicianID: techID,
		}

		// Find technician name
		for _, t := range tc.techs {
			if t.ID == techID {
				trend.TechnicianName = t.Name
				break
			}
		}

		// Calculate KPIs for each period
		for _, period := range periods {
			// Filter data to this period
			periodJobs := tc.filterJobsByDateRange(period.Range)
			periodJobTechs := tc.filterJobTechsByJobs(periodJobs)

			// Create calculator for this period's data
			calc := NewCalculator(periodJobs, periodJobTechs, tc.techs)
			kpis := calc.CalculateTechnicianKPIs(techID)
			kpis.TechnicianName = trend.TechnicianName

			trend.Periods = append(trend.Periods, PeriodKPIs{
				Period: period.Label,
				KPIs:   kpis,
			})
		}

		results = append(results, trend)
	}

	return results
}

// CalculateYoYComparison computes year-over-year comparison for a given period
func (tc *TrendCalculator) CalculateYoYComparison(currentPeriod DateRange, technicianID *int64) []YoYComparison {
	// Calculate prior period (same period, one year earlier)
	priorPeriod := DateRange{
		From: currentPeriod.From.AddDate(-1, 0, 0),
		To:   currentPeriod.To.AddDate(-1, 0, 0),
	}

	// Filter data for each period
	currentJobs := tc.filterJobsByDateRange(currentPeriod)
	currentJobTechs := tc.filterJobTechsByJobs(currentJobs)

	priorJobs := tc.filterJobsByDateRange(priorPeriod)
	priorJobTechs := tc.filterJobTechsByJobs(priorJobs)

	// Create calculators
	currentCalc := NewCalculator(currentJobs, currentJobTechs, tc.techs)
	priorCalc := NewCalculator(priorJobs, priorJobTechs, tc.techs)

	// Determine which technicians to calculate
	var techIDs []int64
	if technicianID != nil {
		techIDs = []int64{*technicianID}
	} else {
		for _, t := range tc.techs {
			techIDs = append(techIDs, t.ID)
		}
	}

	var results []YoYComparison

	for _, techID := range techIDs {
		currentKPIs := currentCalc.CalculateTechnicianKPIs(techID)
		priorKPIs := priorCalc.CalculateTechnicianKPIs(techID)

		// Find tech name
		techName := ""
		for _, t := range tc.techs {
			if t.ID == techID {
				techName = t.Name
				break
			}
		}
		currentKPIs.TechnicianName = techName
		priorKPIs.TechnicianName = techName

		comparison := YoYComparison{
			TechnicianID:   techID,
			TechnicianName: techName,
			CurrentPeriod: PeriodKPIs{
				Period: formatDateRange(currentPeriod),
				KPIs:   currentKPIs,
			},
			PriorPeriod: PeriodKPIs{
				Period: formatDateRange(priorPeriod),
				KPIs:   priorKPIs,
			},
			Changes: calculateChanges(currentKPIs, priorKPIs),
		}

		results = append(results, comparison)
	}

	return results
}

// CalculateTeamTrends computes team-level trends over time
func (tc *TrendCalculator) CalculateTeamTrends(dateRange DateRange, periodType PeriodType, businessUnit *string) []TeamKPIs {
	periods := tc.generatePeriods(dateRange, periodType)

	// For each period, calculate team KPIs
	// This returns the team KPIs for each period
	// We'll return the last period's team KPIs with trend data embedded
	// (This could be extended to return a more complex structure)

	var lastPeriodTeamKPIs []TeamKPIs

	for _, period := range periods {
		periodJobs := tc.filterJobsByDateRange(period.Range)

		// Filter by business unit if specified
		if businessUnit != nil {
			filteredJobs := make([]JobData, 0)
			for _, job := range periodJobs {
				if job.BusinessUnit != nil && *job.BusinessUnit == *businessUnit {
					filteredJobs = append(filteredJobs, job)
				}
			}
			periodJobs = filteredJobs
		}

		periodJobTechs := tc.filterJobTechsByJobs(periodJobs)

		calc := NewCalculator(periodJobs, periodJobTechs, tc.techs)
		lastPeriodTeamKPIs = calc.CalculateTeamKPIs()
	}

	return lastPeriodTeamKPIs
}

// generatePeriods creates period boundaries based on the period type
func (tc *TrendCalculator) generatePeriods(dateRange DateRange, periodType PeriodType) []Period {
	var periods []Period

	current := periodStart(dateRange.From, periodType)
	end := dateRange.To

	for current.Before(end) || current.Equal(end) {
		periodEnd := nextPeriodStart(current, periodType).Add(-time.Nanosecond)
		if periodEnd.After(end) {
			periodEnd = end
		}

		periods = append(periods, Period{
			Label: formatPeriodLabel(current, periodType),
			Range: DateRange{
				From: current,
				To:   periodEnd,
			},
		})

		current = nextPeriodStart(current, periodType)
	}

	return periods
}

// periodStart returns the start of the period containing the given time
func periodStart(t time.Time, periodType PeriodType) time.Time {
	switch periodType {
	case PeriodWeekly:
		// Start of week (Monday)
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return time.Date(t.Year(), t.Month(), t.Day()-(weekday-1), 0, 0, 0, 0, t.Location())
	case PeriodMonthly:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case PeriodQuarterly:
		quarter := (int(t.Month()) - 1) / 3
		return time.Date(t.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, t.Location())
	case PeriodYearly:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

// nextPeriodStart returns the start of the next period
func nextPeriodStart(t time.Time, periodType PeriodType) time.Time {
	switch periodType {
	case PeriodWeekly:
		return t.AddDate(0, 0, 7)
	case PeriodMonthly:
		return t.AddDate(0, 1, 0)
	case PeriodQuarterly:
		return t.AddDate(0, 3, 0)
	case PeriodYearly:
		return t.AddDate(1, 0, 0)
	default:
		return t.AddDate(0, 0, 1)
	}
}

// formatPeriodLabel creates a human-readable label for a period
func formatPeriodLabel(t time.Time, periodType PeriodType) string {
	switch periodType {
	case PeriodWeekly:
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case PeriodMonthly:
		return t.Format("2006-01")
	case PeriodQuarterly:
		quarter := (int(t.Month())-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", t.Year(), quarter)
	case PeriodYearly:
		return fmt.Sprintf("%d", t.Year())
	default:
		return t.Format("2006-01-02")
	}
}

// formatDateRange creates a string representation of a date range
func formatDateRange(dr DateRange) string {
	return fmt.Sprintf("%s to %s", dr.From.Format("2006-01-02"), dr.To.Format("2006-01-02"))
}

// filterJobsByDateRange returns jobs within the given date range
func (tc *TrendCalculator) filterJobsByDateRange(dateRange DateRange) []JobData {
	var filtered []JobData

	for _, job := range tc.jobs {
		if job.JobCompletionDate == nil {
			continue
		}

		completionDate := *job.JobCompletionDate
		if (completionDate.Equal(dateRange.From) || completionDate.After(dateRange.From)) &&
			(completionDate.Equal(dateRange.To) || completionDate.Before(dateRange.To)) {
			filtered = append(filtered, job)
		}
	}

	return filtered
}

// filterJobTechsByJobs returns job_technician records for the given jobs
func (tc *TrendCalculator) filterJobTechsByJobs(jobs []JobData) []JobTechnicianData {
	jobIDs := make(map[string]bool)
	for _, job := range jobs {
		jobIDs[job.ID] = true
	}

	var filtered []JobTechnicianData
	for _, jt := range tc.jobTechs {
		if jobIDs[jt.JobID] {
			filtered = append(filtered, jt)
		}
	}

	return filtered
}

// calculateChanges computes the delta between current and prior period KPIs
func calculateChanges(current, prior TechnicianKPIs) KPIChanges {
	return KPIChanges{
		CallbackRateChange:    current.CallbackRate - prior.CallbackRate,
		AvgTimeOnJobChange:    current.AvgTimeOnJob - prior.AvgTimeOnJob,
		ConversionRateChange:  current.ConversionRate - prior.ConversionRate,
		AvgTicketChange:       current.AvgTicket - prior.AvgTicket,
		EstimatesPerJobChange: current.AvgEstimatesPerJob - prior.AvgEstimatesPerJob,
		TotalJobsChange:       current.TotalJobsCompleted - prior.TotalJobsCompleted,
	}
}

// SortTechniciansByKPI sorts technicians by a specific KPI
type SortField string

const (
	SortByCallbackRate    SortField = "callback_rate"
	SortByTimeOnJob       SortField = "time_on_job"
	SortByConversionRate  SortField = "conversion_rate"
	SortByAvgTicket       SortField = "avg_ticket"
	SortByEstimatesPerJob SortField = "estimates_per_job"
	SortByTotalJobs       SortField = "total_jobs"
)

// SortTechnicianKPIs sorts technician KPIs by the specified field
func SortTechnicianKPIs(kpis []TechnicianKPIs, field SortField, ascending bool) {
	sort.Slice(kpis, func(i, j int) bool {
		var less bool
		switch field {
		case SortByCallbackRate:
			less = kpis[i].CallbackRate < kpis[j].CallbackRate
		case SortByTimeOnJob:
			less = kpis[i].AvgTimeOnJob < kpis[j].AvgTimeOnJob
		case SortByConversionRate:
			less = kpis[i].ConversionRate < kpis[j].ConversionRate
		case SortByAvgTicket:
			less = kpis[i].AvgTicket < kpis[j].AvgTicket
		case SortByEstimatesPerJob:
			less = kpis[i].AvgEstimatesPerJob < kpis[j].AvgEstimatesPerJob
		case SortByTotalJobs:
			less = kpis[i].TotalJobsCompleted < kpis[j].TotalJobsCompleted
		default:
			less = kpis[i].TechnicianName < kpis[j].TechnicianName
		}

		if ascending {
			return less
		}
		return !less
	})
}
