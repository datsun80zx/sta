package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/datsun80zx/sta.git/internal/analytics"
)

// Handler contains the HTTP handlers for the API
type Handler struct {
	analyticsService *analytics.Service
}

// NewHandler creates a new API handler
func NewHandler(analyticsService *analytics.Service) *Handler {
	return &Handler{
		analyticsService: analyticsService,
	}
}

// GetBusinessUnits handles GET /api/business-units
func (h *Handler) GetBusinessUnits(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all business units
	units, err := h.analyticsService.GetBusinessUnits(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get business units", err)
		return
	}

	// Get KPIs for each business unit
	response := BusinessUnitsResponse{
		BusinessUnits: make([]BusinessUnitSummary, 0, len(units)),
	}

	for _, unitName := range units {
		// Get KPIs for this business unit
		filter := analytics.TechnicianFilter{
			BusinessUnit: &unitName,
		}

		teamKPIs, err := h.analyticsService.GetTeamKPIs(ctx, filter)
		if err != nil {
			// Log but continue
			fmt.Printf("Warning: failed to get KPIs for %s: %v\n", unitName, err)
			continue
		}

		// Find this unit's aggregated KPIs
		var metrics Metrics
		for _, team := range teamKPIs {
			if team.BusinessUnit == unitName {
				metrics = convertKPIsToMetrics(team.Aggregated)
				break
			}
		}

		response.BusinessUnits = append(response.BusinessUnits, BusinessUnitSummary{
			ID:      slugify(unitName),
			Name:    unitName,
			Metrics: metrics,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// GetBusinessUnit handles GET /api/business-units/{id}
func (h *Handler) GetBusinessUnit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract ID from path
	unitID := extractPathParam(r.URL.Path, "/api/business-units/")
	if unitID == "" {
		writeError(w, http.StatusBadRequest, "Missing business unit ID", nil)
		return
	}

	// Parse range query param
	dateRange := parseDateRange(r.URL.Query().Get("range"))

	// Get all business units to find the name
	units, err := h.analyticsService.GetBusinessUnits(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get business units", err)
		return
	}

	// Find the unit by ID (slug)
	var unitName string
	for _, name := range units {
		if slugify(name) == unitID {
			unitName = name
			break
		}
	}

	if unitName == "" {
		writeError(w, http.StatusNotFound, "Business unit not found", nil)
		return
	}

	// Get team KPIs
	filter := analytics.TechnicianFilter{
		BusinessUnit: &unitName,
		DateRange:    dateRange,
	}

	teamKPIs, err := h.analyticsService.GetTeamKPIs(ctx, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get team KPIs", err)
		return
	}

	// Build response
	response := BusinessUnitDetail{
		ID:          unitID,
		Name:        unitName,
		Technicians: make([]TechnicianSummary, 0),
		Trend:       make([]TrendDataPoint, 0),
	}

	// Find this unit's data
	for _, team := range teamKPIs {
		if team.BusinessUnit == unitName {
			response.Metrics = convertKPIsToMetrics(team.Aggregated)

			// Add technicians
			for _, tech := range team.Technicians {
				response.Technicians = append(response.Technicians, TechnicianSummary{
					ID:      strconv.FormatInt(tech.TechnicianID, 10),
					Name:    tech.TechnicianName,
					Metrics: convertKPIsToMetrics(tech),
				})
			}
			break
		}
	}

	// Get trend data
	if dateRange != nil {
		trends, err := h.analyticsService.GetTechnicianTrends(ctx, *dateRange, analytics.PeriodWeekly, nil)
		if err == nil {
			response.Trend = aggregateTrendsForUnit(trends, unitName)
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// GetTechnician handles GET /api/technicians/{id}
func (h *Handler) GetTechnician(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract ID from path
	techIDStr := extractPathParam(r.URL.Path, "/api/technicians/")
	if techIDStr == "" {
		writeError(w, http.StatusBadRequest, "Missing technician ID", nil)
		return
	}

	techID, err := strconv.ParseInt(techIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid technician ID", err)
		return
	}

	// Parse range query param
	dateRange := parseDateRange(r.URL.Query().Get("range"))

	// Get lifetime KPIs (no date filter)
	lifetimeFilter := analytics.TechnicianFilter{
		TechnicianID: &techID,
	}

	lifetimeKPIs, err := h.analyticsService.GetTechnicianKPIs(ctx, lifetimeFilter)
	if err != nil || len(lifetimeKPIs) == 0 {
		writeError(w, http.StatusNotFound, "Technician not found", err)
		return
	}

	techKPI := lifetimeKPIs[0]

	// Build response
	response := TechnicianDetail{
		ID:   techIDStr,
		Name: techKPI.TechnicianName,
		BusinessUnit: BusinessUnitRef{
			ID:   slugify(techKPI.BusinessUnit),
			Name: techKPI.BusinessUnit,
		},
		LifetimeMetrics: convertKPIsToMetrics(techKPI),
		Trend:           make([]TrendDataPoint, 0),
	}

	// Get trend data
	if dateRange != nil {
		trends, err := h.analyticsService.GetTechnicianTrends(ctx, *dateRange, analytics.PeriodWeekly, &techID)
		if err == nil && len(trends) > 0 {
			response.Trend = convertTrendToDataPoints(trends[0])
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// -----------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------

func convertKPIsToMetrics(kpi analytics.TechnicianKPIs) Metrics {
	return Metrics{
		CloseRate:              kpi.ConversionRate / 100, // Convert from percentage to decimal
		AverageTicket:          kpi.AvgTicket,
		TotalJobs:              kpi.TotalJobsCompleted,
		AverageTimeOnJob:       kpi.AvgTimeOnJob,
		AverageEstimatesPerJob: kpi.AvgEstimatesPerJob,
		CallbackRate:           kpi.CallbackRate / 100, // Convert from percentage to decimal
	}
}

func convertTrendToDataPoints(trend analytics.TechnicianTrend) []TrendDataPoint {
	points := make([]TrendDataPoint, 0, len(trend.Periods))

	for _, period := range trend.Periods {
		points = append(points, TrendDataPoint{
			Date:                   period.Period,
			CloseRate:              period.KPIs.ConversionRate / 100,
			AverageTicket:          period.KPIs.AvgTicket,
			TotalJobs:              period.KPIs.TotalJobsCompleted,
			AverageTimeOnJob:       period.KPIs.AvgTimeOnJob,
			AverageEstimatesPerJob: period.KPIs.AvgEstimatesPerJob,
			CallbackRate:           period.KPIs.CallbackRate / 100,
		})
	}

	return points
}

func aggregateTrendsForUnit(trends []analytics.TechnicianTrend, unitName string) []TrendDataPoint {
	// Aggregate all technicians in this unit by period
	periodTotals := make(map[string]struct {
		jobs        int
		conversions int
		callbacks   int
		hours       float64
		estimates   int
		sales       float64
		techCount   int
	})

	for _, trend := range trends {
		for _, period := range trend.Periods {
			if trend.TechnicianName == "" {
				continue
			}

			existing := periodTotals[period.Period]
			existing.jobs += period.KPIs.TotalJobsCompleted
			existing.conversions += period.KPIs.Conversions
			existing.callbacks += period.KPIs.CallbackCount
			existing.hours += period.KPIs.TotalHoursWorked
			existing.estimates += period.KPIs.TotalEstimates
			existing.sales += period.KPIs.TotalSales
			existing.techCount++
			periodTotals[period.Period] = existing
		}
	}

	// Convert to data points
	points := make([]TrendDataPoint, 0, len(periodTotals))
	for period, totals := range periodTotals {
		point := TrendDataPoint{
			Date:      period,
			TotalJobs: totals.jobs,
		}

		if totals.jobs > 0 {
			point.CloseRate = float64(totals.conversions) / float64(totals.jobs)
			point.CallbackRate = float64(totals.callbacks) / float64(totals.jobs)
			point.AverageTimeOnJob = totals.hours / float64(totals.jobs)
			point.AverageEstimatesPerJob = float64(totals.estimates) / float64(totals.jobs)
		}

		if totals.conversions > 0 {
			point.AverageTicket = totals.sales / float64(totals.conversions)
		}

		points = append(points, point)
	}

	return points
}

func parseDateRange(rangeStr string) *analytics.DateRange {
	now := time.Now()
	var from time.Time

	switch rangeStr {
	case "7d":
		from = now.AddDate(0, 0, -7)
	case "30d":
		from = now.AddDate(0, 0, -30)
	case "90d":
		from = now.AddDate(0, 0, -90)
	case "ytd":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		// Default to 30 days
		from = now.AddDate(0, 0, -30)
	}

	return &analytics.DateRange{
		From: from,
		To:   now,
	}
}

func extractPathParam(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	param := strings.TrimPrefix(path, prefix)
	// Remove trailing slash if present
	param = strings.TrimSuffix(param, "/")
	// URL decode
	decoded, err := url.PathUnescape(param)
	if err != nil {
		return param
	}
	return decoded
}

func slugify(s string) string {
	// Simple slugify - lowercase and replace spaces with dashes
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	return s
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string, err error) {
	resp := ErrorResponse{Message: message}
	if err != nil {
		resp.Message = fmt.Sprintf("%s: %v", message, err)
	}
	writeJSON(w, status, resp)
}

// HealthCheck handles GET /api/health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
