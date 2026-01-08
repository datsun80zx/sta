package analytics

import (
	"strings"

	"github.com/shopspring/decimal"
)

// Calculator performs KPI calculations from raw data
type Calculator struct {
	jobs          []JobData
	jobTechs      []JobTechnicianData
	techsByID     map[int64]TechnicianData
	jobsByID      map[string]JobData
	jobTechsByJob map[string][]JobTechnicianData
}

// NewCalculator creates a calculator with the provided raw data
func NewCalculator(jobs []JobData, jobTechs []JobTechnicianData, technicians []TechnicianData) *Calculator {
	c := &Calculator{
		jobs:          jobs,
		jobTechs:      jobTechs,
		techsByID:     make(map[int64]TechnicianData),
		jobsByID:      make(map[string]JobData),
		jobTechsByJob: make(map[string][]JobTechnicianData),
	}

	// Build lookup maps
	for _, t := range technicians {
		c.techsByID[t.ID] = t
	}
	for _, j := range jobs {
		c.jobsByID[j.ID] = j
	}
	for _, jt := range jobTechs {
		c.jobTechsByJob[jt.JobID] = append(c.jobTechsByJob[jt.JobID], jt)
	}

	return c
}

// CalculateAllTechnicianKPIs computes KPIs for all technicians
func (c *Calculator) CalculateAllTechnicianKPIs() []TechnicianKPIs {
	var results []TechnicianKPIs

	for techID, tech := range c.techsByID {
		kpis := c.CalculateTechnicianKPIs(techID)
		kpis.TechnicianName = tech.Name
		if tech.BusinessUnit != nil {
			kpis.BusinessUnit = *tech.BusinessUnit
		}
		results = append(results, kpis)
	}

	return results
}

// CalculateTechnicianKPIs computes all KPIs for a single technician
func (c *Calculator) CalculateTechnicianKPIs(technicianID int64) TechnicianKPIs {
	kpis := TechnicianKPIs{
		TechnicianID: technicianID,
	}

	// Gather jobs by role for this technician
	primaryJobs := c.getJobsWhereRole(technicianID, "primary")
	soldByJobs := c.getJobsWhereRole(technicianID, "sold_by")

	// 1. Total Jobs Completed (Primary role, Completed status)
	kpis.TotalJobsCompleted = c.countCompletedJobs(primaryJobs)

	// 2. Callback Rate
	kpis.CallbackCount, kpis.CallbackRate = c.calculateCallbackRate(technicianID, primaryJobs)

	// 3. Time on Job (exclude multi-tech jobs)
	kpis.TotalHoursWorked, kpis.AvgTimeOnJob, kpis.JobsIncludedInTimeCalc = c.calculateTimeOnJob(primaryJobs)

	// 4. Conversion Rate (dedupe sold jobs by project)
	kpis.Opportunities, kpis.Conversions, kpis.ConversionRate = c.calculateConversionRate(primaryJobs, soldByJobs)

	// 5. Average Ticket (dedupe by project, sum estimate sales subtotal)
	kpis.TotalSales, kpis.AvgTicket = c.calculateAverageTicket(soldByJobs)

	// 6. Estimates per Job
	kpis.TotalEstimates, kpis.AvgEstimatesPerJob = c.calculateEstimatesPerJob(primaryJobs)

	return kpis
}

// getJobsWhereRole returns job IDs where the technician has the specified role
func (c *Calculator) getJobsWhereRole(technicianID int64, role string) []string {
	var jobIDs []string
	seen := make(map[string]bool)

	for _, jt := range c.jobTechs {
		if jt.TechnicianID == technicianID && jt.Role == role {
			if !seen[jt.JobID] {
				jobIDs = append(jobIDs, jt.JobID)
				seen[jt.JobID] = true
			}
		}
	}

	return jobIDs
}

// countCompletedJobs counts jobs with Status = "Completed"
func (c *Calculator) countCompletedJobs(jobIDs []string) int {
	count := 0
	for _, jobID := range jobIDs {
		if job, ok := c.jobsByID[jobID]; ok {
			if job.Status == "Completed" {
				count++
			}
		}
	}
	return count
}

// calculateCallbackRate computes callback rate for a technician
// Callback = warranty/recall job that traces back to a job this tech ran as Primary
func (c *Calculator) calculateCallbackRate(technicianID int64, primaryJobIDs []string) (int, float64) {
	// Build set of jobs this tech ran as primary
	primaryJobSet := make(map[string]bool)
	for _, jobID := range primaryJobIDs {
		primaryJobSet[jobID] = true
	}

	// Count callbacks: jobs where warranty_for or recall_for points to one of their primary jobs
	callbackCount := 0
	for _, job := range c.jobs {
		// Check if this is a warranty job pointing to one of their jobs
		if job.Warranty && job.WarrantyForJobID != nil {
			if primaryJobSet[*job.WarrantyForJobID] {
				callbackCount++
			}
		}
		// Check if this is a recall job pointing to one of their jobs
		if job.Recall && job.RecallForJobID != nil {
			if primaryJobSet[*job.RecallForJobID] {
				callbackCount++
			}
		}
	}

	// Calculate rate
	totalJobs := len(primaryJobIDs)
	if totalJobs == 0 {
		return 0, 0
	}

	rate := float64(callbackCount) / float64(totalJobs) * 100
	return callbackCount, rate
}

// calculateTimeOnJob computes average time, excluding multi-tech jobs
func (c *Calculator) calculateTimeOnJob(primaryJobIDs []string) (float64, float64, int) {
	totalHours := decimal.Zero
	jobCount := 0

	for _, jobID := range primaryJobIDs {
		job, ok := c.jobsByID[jobID]
		if !ok || job.Status != "Completed" {
			continue
		}

		// Exclude multi-tech jobs
		if c.isMultiTechJob(job) {
			continue
		}

		totalHours = totalHours.Add(job.TotalHoursWorked)
		jobCount++
	}

	if jobCount == 0 {
		return 0, 0, 0
	}

	totalFloat, _ := totalHours.Float64()
	avgHours := totalFloat / float64(jobCount)

	return totalFloat, avgHours, jobCount
}

// isMultiTechJob checks if a job has multiple assigned technicians
func (c *Calculator) isMultiTechJob(job JobData) bool {
	if job.AssignedTechnicians == nil || *job.AssignedTechnicians == "" {
		return false
	}

	// Split by comma and count non-empty names
	names := strings.Split(*job.AssignedTechnicians, ",")
	count := 0
	for _, name := range names {
		if strings.TrimSpace(name) != "" {
			count++
		}
	}

	return count > 1
}

// calculateConversionRate computes conversion rate with project deduplication
// Opportunities = jobs ran as primary
// Conversions = unique sales (deduplicated by project_id)
func (c *Calculator) calculateConversionRate(primaryJobIDs, soldByJobIDs []string) (int, int, float64) {
	// Opportunities = completed jobs where tech is primary
	opportunities := 0
	for _, jobID := range primaryJobIDs {
		if job, ok := c.jobsByID[jobID]; ok {
			if job.Status == "Completed" {
				opportunities++
			}
		}
	}

	// Conversions = unique sales, deduplicated by project
	conversions := c.countUniqueSales(soldByJobIDs)

	if opportunities == 0 {
		return 0, conversions, 0
	}

	rate := float64(conversions) / float64(opportunities) * 100
	return opportunities, conversions, rate
}

// countUniqueSales counts unique sales, deduplicating by project_id
// Jobs with null project_id each count as 1 sale
func (c *Calculator) countUniqueSales(soldByJobIDs []string) int {
	projectsSeen := make(map[string]bool)
	noProjectCount := 0

	for _, jobID := range soldByJobIDs {
		job, ok := c.jobsByID[jobID]
		if !ok || job.Status != "Completed" {
			continue
		}

		if job.ProjectID != nil && *job.ProjectID != "" {
			// Has project - deduplicate
			projectsSeen[*job.ProjectID] = true
		} else {
			// No project - counts as individual sale
			noProjectCount++
		}
	}

	return len(projectsSeen) + noProjectCount
}

// calculateAverageTicket computes average ticket with project deduplication
// Groups estimate_sales_subtotal by project, then averages
func (c *Calculator) calculateAverageTicket(soldByJobIDs []string) (float64, float64) {
	// Group sales by project
	salesByProject := make(map[string]decimal.Decimal)
	var noProjectSales []decimal.Decimal

	for _, jobID := range soldByJobIDs {
		job, ok := c.jobsByID[jobID]
		if !ok || job.Status != "Completed" {
			continue
		}

		if job.ProjectID != nil && *job.ProjectID != "" {
			// Has project - sum into project total
			existing := salesByProject[*job.ProjectID]
			salesByProject[*job.ProjectID] = existing.Add(job.EstimateSalesSubtotal)
		} else {
			// No project - individual sale
			noProjectSales = append(noProjectSales, job.EstimateSalesSubtotal)
		}
	}

	// Calculate total and count
	totalSales := decimal.Zero
	saleCount := 0

	for _, projectTotal := range salesByProject {
		totalSales = totalSales.Add(projectTotal)
		saleCount++
	}

	for _, sale := range noProjectSales {
		totalSales = totalSales.Add(sale)
		saleCount++
	}

	if saleCount == 0 {
		return 0, 0
	}

	totalFloat, _ := totalSales.Float64()
	avgTicket := totalFloat / float64(saleCount)

	return totalFloat, avgTicket
}

// calculateEstimatesPerJob computes average estimates per job
func (c *Calculator) calculateEstimatesPerJob(primaryJobIDs []string) (int, float64) {
	totalEstimates := 0
	jobCount := 0

	for _, jobID := range primaryJobIDs {
		job, ok := c.jobsByID[jobID]
		if !ok || job.Status != "Completed" {
			continue
		}

		totalEstimates += job.EstimateCount
		jobCount++
	}

	if jobCount == 0 {
		return 0, 0
	}

	avgEstimates := float64(totalEstimates) / float64(jobCount)
	return totalEstimates, avgEstimates
}

// CalculateTeamKPIs computes KPIs aggregated by business unit
func (c *Calculator) CalculateTeamKPIs() []TeamKPIs {
	// Group technicians by business unit
	techsByBU := make(map[string][]int64)

	for techID, tech := range c.techsByID {
		bu := "Unknown"
		if tech.BusinessUnit != nil {
			bu = *tech.BusinessUnit
		}
		techsByBU[bu] = append(techsByBU[bu], techID)
	}

	var results []TeamKPIs

	for bu, techIDs := range techsByBU {
		team := TeamKPIs{
			BusinessUnit: bu,
		}

		// Calculate KPIs for each technician in the team
		for _, techID := range techIDs {
			kpis := c.CalculateTechnicianKPIs(techID)
			if tech, ok := c.techsByID[techID]; ok {
				kpis.TechnicianName = tech.Name
				kpis.BusinessUnit = bu
			}
			team.Technicians = append(team.Technicians, kpis)
		}

		// Calculate team aggregates
		team.Aggregated = c.aggregateKPIs(team.Technicians)
		team.Aggregated.BusinessUnit = bu

		results = append(results, team)
	}

	return results
}

// aggregateKPIs computes team-wide averages from individual technician KPIs
func (c *Calculator) aggregateKPIs(techKPIs []TechnicianKPIs) TechnicianKPIs {
	if len(techKPIs) == 0 {
		return TechnicianKPIs{}
	}

	agg := TechnicianKPIs{}

	// Sum totals
	for _, kpi := range techKPIs {
		agg.TotalJobsCompleted += kpi.TotalJobsCompleted
		agg.CallbackCount += kpi.CallbackCount
		agg.TotalHoursWorked += kpi.TotalHoursWorked
		agg.JobsIncludedInTimeCalc += kpi.JobsIncludedInTimeCalc
		agg.Opportunities += kpi.Opportunities
		agg.Conversions += kpi.Conversions
		agg.TotalSales += kpi.TotalSales
		agg.TotalEstimates += kpi.TotalEstimates
	}

	// Calculate team-wide rates (not averages of individual rates)
	if agg.TotalJobsCompleted > 0 {
		agg.CallbackRate = float64(agg.CallbackCount) / float64(agg.TotalJobsCompleted) * 100
	}

	if agg.JobsIncludedInTimeCalc > 0 {
		agg.AvgTimeOnJob = agg.TotalHoursWorked / float64(agg.JobsIncludedInTimeCalc)
	}

	if agg.Opportunities > 0 {
		agg.ConversionRate = float64(agg.Conversions) / float64(agg.Opportunities) * 100
	}

	if agg.Conversions > 0 {
		agg.AvgTicket = agg.TotalSales / float64(agg.Conversions)
	}

	if agg.TotalJobsCompleted > 0 {
		agg.AvgEstimatesPerJob = float64(agg.TotalEstimates) / float64(agg.TotalJobsCompleted)
	}

	return agg
}
