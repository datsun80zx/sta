package analytics

import (
	"time"

	"github.com/shopspring/decimal"
)

// DateRange represents a time period for filtering
type DateRange struct {
	From time.Time
	To   time.Time
}

// Period represents a named time period for trend analysis
type Period struct {
	Label string // e.g., "2024-Q1", "2024-01", "2024-W03"
	Range DateRange
}

// TechnicianFilter controls which data to include in calculations
type TechnicianFilter struct {
	DateRange    *DateRange // nil means all time
	BusinessUnit *string    // nil means all business units
	TechnicianID *int64     // nil means all technicians
}

// TechnicianKPIs holds calculated metrics for one technician
type TechnicianKPIs struct {
	TechnicianID   int64
	TechnicianName string
	BusinessUnit   string

	// Volume
	TotalJobsCompleted int

	// Quality (Callback Rate)
	CallbackRate  float64 // percentage: (callbacks / total jobs) * 100
	CallbackCount int     // number of callbacks attributed to this tech

	// Efficiency (Time on Job)
	AvgTimeOnJob           float64 // average hours per job
	TotalHoursWorked       float64 // sum of hours
	JobsIncludedInTimeCalc int     // jobs with single tech (excluded multi-tech)

	// Sales (Conversion Rate)
	ConversionRate float64 // percentage: (conversions / opportunities) * 100
	Opportunities  int     // jobs ran as primary
	Conversions    int     // unique sales (deduplicated by project)

	// Revenue (Average Ticket)
	AvgTicket  float64 // average sale amount (deduplicated by project)
	TotalSales float64 // sum of estimate sales subtotal (grouped by project)

	// Activity (Estimates per Job)
	AvgEstimatesPerJob float64
	TotalEstimates     int
}

// TechnicianTrend holds KPIs across multiple time periods for one technician
type TechnicianTrend struct {
	TechnicianID   int64
	TechnicianName string
	Periods        []PeriodKPIs
}

// PeriodKPIs pairs a time period label with its calculated KPIs
type PeriodKPIs struct {
	Period string // "2024-Q1", "2024-01", "2024-W03", "2024"
	KPIs   TechnicianKPIs
}

// YoYComparison holds year-over-year comparison data
type YoYComparison struct {
	TechnicianID   int64
	TechnicianName string
	CurrentPeriod  PeriodKPIs
	PriorPeriod    PeriodKPIs
	Changes        KPIChanges
}

// KPIChanges represents the delta between two periods
type KPIChanges struct {
	CallbackRateChange    float64 // negative is good (fewer callbacks)
	AvgTimeOnJobChange    float64 // negative is good (faster)
	ConversionRateChange  float64 // positive is good (more sales)
	AvgTicketChange       float64 // positive is good (higher sales)
	EstimatesPerJobChange float64
	TotalJobsChange       int
}

// TeamKPIs holds aggregated KPIs for a business unit/team
type TeamKPIs struct {
	BusinessUnit string
	Technicians  []TechnicianKPIs
	Aggregated   TechnicianKPIs // team-wide averages/totals
}

// -----------------------------------------------------------------
// Raw data types (fetched from database, used for calculations)
// -----------------------------------------------------------------

// JobData represents raw job data needed for technician calculations
type JobData struct {
	ID                    string
	ProjectID             *string
	Status                string
	JobCompletionDate     *time.Time
	BusinessUnit          *string
	TotalHoursWorked      decimal.Decimal
	EstimateCount         int
	EstimateSalesSubtotal decimal.Decimal
	AssignedTechnicians   *string // comma-separated list
	Warranty              bool
	WarrantyForJobID      *string
	Recall                bool
	RecallForJobID        *string
}

// JobTechnicianData represents a job-technician relationship
type JobTechnicianData struct {
	JobID          string
	TechnicianID   int64
	TechnicianName string
	Role           string // "primary", "sold_by", "assigned"
}

// TechnicianData represents basic technician info
type TechnicianData struct {
	ID           int64
	Name         string
	BusinessUnit *string // derived from most common BU in their jobs
}
