package api

// BusinessUnitsResponse is the response for GET /api/business-units
type BusinessUnitsResponse struct {
	BusinessUnits []BusinessUnitSummary `json:"business_units"`
}

// BusinessUnitSummary is a summary of a business unit with aggregate metrics
type BusinessUnitSummary struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Metrics Metrics `json:"metrics"`
}

// BusinessUnitDetail is the detailed view of a business unit
type BusinessUnitDetail struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Metrics     Metrics             `json:"metrics"`
	Technicians []TechnicianSummary `json:"technicians"`
	Trend       []TrendDataPoint    `json:"trend"`
}

// TechnicianSummary is a summary of a technician within a business unit
type TechnicianSummary struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Metrics Metrics `json:"metrics"`
}

// TechnicianDetail is the detailed view of a single technician
type TechnicianDetail struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	BusinessUnit    BusinessUnitRef  `json:"business_unit"`
	LifetimeMetrics Metrics          `json:"lifetime_metrics"`
	Trend           []TrendDataPoint `json:"trend"`
}

// BusinessUnitRef is a reference to a business unit
type BusinessUnitRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Metrics contains all the KPIs we track
type Metrics struct {
	CloseRate              float64 `json:"close_rate"`                // Conversion rate as decimal (0.45 = 45%)
	AverageTicket          float64 `json:"average_ticket"`            // Average sale amount
	TotalJobs              int     `json:"total_jobs"`                // Total jobs completed
	AverageTimeOnJob       float64 `json:"average_time_on_job"`       // Average hours per job
	AverageEstimatesPerJob float64 `json:"average_estimates_per_job"` // Estimates per job
	CallbackRate           float64 `json:"callback_rate"`             // Callback rate as decimal
}

// TrendDataPoint represents a single point in a trend chart
type TrendDataPoint struct {
	Date                   string  `json:"date"` // ISO date string
	CloseRate              float64 `json:"close_rate"`
	AverageTicket          float64 `json:"average_ticket"`
	TotalJobs              int     `json:"total_jobs"`
	AverageTimeOnJob       float64 `json:"average_time_on_job"`
	AverageEstimatesPerJob float64 `json:"average_estimates_per_job"`
	CallbackRate           float64 `json:"callback_rate"`
}

// ErrorResponse is returned when an error occurs
type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
