package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/datsun80zx/sta.git/internal/analytics"
)

// TechnicianCommand handles all technician-related CLI operations
type TechnicianCommand struct {
	db  *sql.DB
	svc *analytics.Service
}

// NewTechnicianCommand creates a new technician command handler
func NewTechnicianCommand(db *sql.DB) *TechnicianCommand {
	return &TechnicianCommand{
		db:  db,
		svc: analytics.NewService(db),
	}
}

// Run executes the technician command with the given arguments
func (tc *TechnicianCommand) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		tc.printUsage()
		return nil
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "kpis":
		return tc.runKPIs(ctx, subargs)
	case "trends":
		return tc.runTrends(ctx, subargs)
	case "yoy":
		return tc.runYoY(ctx, subargs)
	case "teams":
		return tc.runTeams(ctx, subargs)
	case "list":
		return tc.runList(ctx, subargs)
	case "help":
		tc.printUsage()
		return nil
	default:
		fmt.Printf("Unknown subcommand: %s\n\n", subcommand)
		tc.printUsage()
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

func (tc *TechnicianCommand) printUsage() {
	fmt.Println(`Usage: sta technicians <subcommand> [options]

Subcommands:
  kpis     Show KPI scorecards for technicians
  trends   Show KPI trends over time (weekly/monthly/quarterly/yearly)
  yoy      Show year-over-year comparison
  teams    Show team/business unit aggregated KPIs
  list     List all technicians

Examples:
  sta technicians kpis
  sta technicians kpis --from 2024-01-01 --to 2024-12-31
  sta technicians kpis --business-unit "HVAC"
  sta technicians trends --period monthly --from 2024-01-01 --to 2024-12-31
  sta technicians yoy --from 2025-01-01 --to 2025-03-31
  sta technicians teams`)
}

// -----------------------------------------------------------------
// Subcommand: kpis
// -----------------------------------------------------------------

func (tc *TechnicianCommand) runKPIs(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("kpis", flag.ExitOnError)
	fromStr := fs.String("from", "", "Start date (YYYY-MM-DD)")
	toStr := fs.String("to", "", "End date (YYYY-MM-DD)")
	businessUnit := fs.String("business-unit", "", "Filter by business unit")
	techName := fs.String("technician", "", "Filter by technician name")
	sortBy := fs.String("sort", "total_jobs", "Sort by: callback_rate, time_on_job, conversion_rate, avg_ticket, estimates_per_job, total_jobs")
	ascending := fs.Bool("asc", false, "Sort ascending (default is descending)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Build filter
	filter := analytics.TechnicianFilter{}

	if *fromStr != "" && *toStr != "" {
		from, err := time.Parse("2006-01-02", *fromStr)
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
		to, err := time.Parse("2006-01-02", *toStr)
		if err != nil {
			return fmt.Errorf("invalid to date: %w", err)
		}
		filter.DateRange = &analytics.DateRange{From: from, To: to}
	}

	if *businessUnit != "" {
		filter.BusinessUnit = businessUnit
	}

	// Get KPIs
	kpis, err := tc.svc.GetTechnicianKPIs(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get technician KPIs: %w", err)
	}

	// Filter by technician name if specified
	if *techName != "" {
		var filtered []analytics.TechnicianKPIs
		for _, kpi := range kpis {
			if strings.Contains(strings.ToLower(kpi.TechnicianName), strings.ToLower(*techName)) {
				filtered = append(filtered, kpi)
			}
		}
		kpis = filtered
	}

	// Sort results
	sortField := parseSortField(*sortBy)
	analytics.SortTechnicianKPIs(kpis, sortField, *ascending)

	// Print results
	tc.printKPIsTable(kpis, filter)

	return nil
}

func (tc *TechnicianCommand) printKPIsTable(kpis []analytics.TechnicianKPIs, filter analytics.TechnicianFilter) {
	// Print header
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("                                    TECHNICIAN KPI SCORECARD")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")

	if filter.DateRange != nil {
		fmt.Printf("  Date Range: %s to %s\n", filter.DateRange.From.Format("2006-01-02"), filter.DateRange.To.Format("2006-01-02"))
	} else {
		fmt.Println("  Date Range: All Time")
	}
	if filter.BusinessUnit != nil {
		fmt.Printf("  Business Unit: %s\n", *filter.BusinessUnit)
	}
	fmt.Println("───────────────────────────────────────────────────────────────────────────────────────────────────────────────")
	fmt.Println()

	if len(kpis) == 0 {
		fmt.Println("  No data found for the specified filters.")
		fmt.Println()
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header row
	fmt.Fprintln(w, "  Technician\tBusiness Unit\tJobs\tCallback %\tAvg Hours\tConversion %\tAvg Ticket\tEst/Job\t")
	fmt.Fprintln(w, "  ──────────\t─────────────\t────\t──────────\t─────────\t────────────\t──────────\t───────\t")

	// Data rows
	for _, kpi := range kpis {
		fmt.Fprintf(w, "  %s\t%s\t%d\t%.1f%%\t%.1f\t%.1f%%\t$%.0f\t%.1f\t\n",
			truncateName(kpi.TechnicianName, 20),
			truncateName(kpi.BusinessUnit, 15),
			kpi.TotalJobsCompleted,
			kpi.CallbackRate,
			kpi.AvgTimeOnJob,
			kpi.ConversionRate,
			kpi.AvgTicket,
			kpi.AvgEstimatesPerJob,
		)
	}

	w.Flush()
	fmt.Println()
	fmt.Printf("  Total technicians: %d\n", len(kpis))
	fmt.Println()
}

// -----------------------------------------------------------------
// Subcommand: trends
// -----------------------------------------------------------------

func (tc *TechnicianCommand) runTrends(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("trends", flag.ExitOnError)
	fromStr := fs.String("from", "", "Start date (YYYY-MM-DD) - required")
	toStr := fs.String("to", "", "End date (YYYY-MM-DD) - required")
	periodStr := fs.String("period", "monthly", "Period type: weekly, monthly, quarterly, yearly")
	techName := fs.String("technician", "", "Filter by technician name")
	metric := fs.String("metric", "all", "Metric to show: all, callback_rate, conversion_rate, avg_ticket, time_on_job")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *fromStr == "" || *toStr == "" {
		fmt.Println("Error: --from and --to are required for trends")
		return fmt.Errorf("missing required date range")
	}

	from, err := time.Parse("2006-01-02", *fromStr)
	if err != nil {
		return fmt.Errorf("invalid from date: %w", err)
	}
	to, err := time.Parse("2006-01-02", *toStr)
	if err != nil {
		return fmt.Errorf("invalid to date: %w", err)
	}

	dateRange := analytics.DateRange{From: from, To: to}
	periodType := parsePeriodType(*periodStr)

	// Get trends
	trends, err := tc.svc.GetTechnicianTrends(ctx, dateRange, periodType, nil)
	if err != nil {
		return fmt.Errorf("failed to get trends: %w", err)
	}

	// Filter by technician name if specified
	if *techName != "" {
		var filtered []analytics.TechnicianTrend
		for _, trend := range trends {
			if strings.Contains(strings.ToLower(trend.TechnicianName), strings.ToLower(*techName)) {
				filtered = append(filtered, trend)
			}
		}
		trends = filtered
	}

	// Print results
	tc.printTrendsTable(trends, dateRange, periodType, *metric)

	return nil
}

func (tc *TechnicianCommand) printTrendsTable(trends []analytics.TechnicianTrend, dateRange analytics.DateRange, periodType analytics.PeriodType, metric string) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("                                    TECHNICIAN TRENDS (%s)\n", strings.ToUpper(string(periodType)))
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("  Date Range: %s to %s\n", dateRange.From.Format("2006-01-02"), dateRange.To.Format("2006-01-02"))
	fmt.Println("───────────────────────────────────────────────────────────────────────────────────────────────────────────────")
	fmt.Println()

	if len(trends) == 0 {
		fmt.Println("  No data found for the specified filters.")
		fmt.Println()
		return
	}

	for _, trend := range trends {
		if len(trend.Periods) == 0 {
			continue
		}

		fmt.Printf("  ▶ %s\n", trend.TechnicianName)
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		// Build header based on metric selection
		if metric == "all" {
			fmt.Fprintln(w, "    Period\tJobs\tCallback %\tConversion %\tAvg Ticket\tAvg Hours\t")
			fmt.Fprintln(w, "    ──────\t────\t──────────\t────────────\t──────────\t─────────\t")
		} else {
			fmt.Fprintf(w, "    Period\t%s\t\n", metricLabel(metric))
			fmt.Fprintln(w, "    ──────\t──────────\t")
		}

		for _, p := range trend.Periods {
			if metric == "all" {
				fmt.Fprintf(w, "    %s\t%d\t%.1f%%\t%.1f%%\t$%.0f\t%.1f\t\n",
					p.Period,
					p.KPIs.TotalJobsCompleted,
					p.KPIs.CallbackRate,
					p.KPIs.ConversionRate,
					p.KPIs.AvgTicket,
					p.KPIs.AvgTimeOnJob,
				)
			} else {
				fmt.Fprintf(w, "    %s\t%s\t\n", p.Period, formatMetricValue(p.KPIs, metric))
			}
		}

		w.Flush()
		fmt.Println()
	}
}

// -----------------------------------------------------------------
// Subcommand: yoy (Year-over-Year)
// -----------------------------------------------------------------

func (tc *TechnicianCommand) runYoY(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("yoy", flag.ExitOnError)
	fromStr := fs.String("from", "", "Current period start date (YYYY-MM-DD) - required")
	toStr := fs.String("to", "", "Current period end date (YYYY-MM-DD) - required")
	techName := fs.String("technician", "", "Filter by technician name")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *fromStr == "" || *toStr == "" {
		fmt.Println("Error: --from and --to are required for YoY comparison")
		return fmt.Errorf("missing required date range")
	}

	from, err := time.Parse("2006-01-02", *fromStr)
	if err != nil {
		return fmt.Errorf("invalid from date: %w", err)
	}
	to, err := time.Parse("2006-01-02", *toStr)
	if err != nil {
		return fmt.Errorf("invalid to date: %w", err)
	}

	currentPeriod := analytics.DateRange{From: from, To: to}

	// Get YoY comparison
	comparisons, err := tc.svc.GetYoYComparison(ctx, currentPeriod, nil)
	if err != nil {
		return fmt.Errorf("failed to get YoY comparison: %w", err)
	}

	// Filter by technician name if specified
	if *techName != "" {
		var filtered []analytics.YoYComparison
		for _, comp := range comparisons {
			if strings.Contains(strings.ToLower(comp.TechnicianName), strings.ToLower(*techName)) {
				filtered = append(filtered, comp)
			}
		}
		comparisons = filtered
	}

	// Print results
	tc.printYoYTable(comparisons, currentPeriod)

	return nil
}

func (tc *TechnicianCommand) printYoYTable(comparisons []analytics.YoYComparison, currentPeriod analytics.DateRange) {
	priorPeriod := analytics.DateRange{
		From: currentPeriod.From.AddDate(-1, 0, 0),
		To:   currentPeriod.To.AddDate(-1, 0, 0),
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("                                    YEAR-OVER-YEAR COMPARISON")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("  Current Period: %s to %s\n", currentPeriod.From.Format("2006-01-02"), currentPeriod.To.Format("2006-01-02"))
	fmt.Printf("  Prior Period:   %s to %s\n", priorPeriod.From.Format("2006-01-02"), priorPeriod.To.Format("2006-01-02"))
	fmt.Println("───────────────────────────────────────────────────────────────────────────────────────────────────────────────")
	fmt.Println()

	if len(comparisons) == 0 {
		fmt.Println("  No data found for the specified filters.")
		fmt.Println()
		return
	}

	for _, comp := range comparisons {
		// Skip technicians with no data in either period
		if comp.CurrentPeriod.KPIs.TotalJobsCompleted == 0 && comp.PriorPeriod.KPIs.TotalJobsCompleted == 0 {
			continue
		}

		fmt.Printf("  ▶ %s\n", comp.TechnicianName)
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		fmt.Fprintln(w, "    Metric\tPrior\tCurrent\tChange\t")
		fmt.Fprintln(w, "    ──────\t─────\t───────\t──────\t")

		// Total Jobs
		fmt.Fprintf(w, "    Total Jobs\t%d\t%d\t%s\t\n",
			comp.PriorPeriod.KPIs.TotalJobsCompleted,
			comp.CurrentPeriod.KPIs.TotalJobsCompleted,
			formatIntChange(comp.Changes.TotalJobsChange),
		)

		// Callback Rate (lower is better)
		fmt.Fprintf(w, "    Callback Rate\t%.1f%%\t%.1f%%\t%s\t\n",
			comp.PriorPeriod.KPIs.CallbackRate,
			comp.CurrentPeriod.KPIs.CallbackRate,
			formatPercentChangeInverse(comp.Changes.CallbackRateChange),
		)

		// Conversion Rate (higher is better)
		fmt.Fprintf(w, "    Conversion Rate\t%.1f%%\t%.1f%%\t%s\t\n",
			comp.PriorPeriod.KPIs.ConversionRate,
			comp.CurrentPeriod.KPIs.ConversionRate,
			formatPercentChange(comp.Changes.ConversionRateChange),
		)

		// Average Ticket (higher is better)
		fmt.Fprintf(w, "    Avg Ticket\t$%.0f\t$%.0f\t%s\t\n",
			comp.PriorPeriod.KPIs.AvgTicket,
			comp.CurrentPeriod.KPIs.AvgTicket,
			formatDollarChange(comp.Changes.AvgTicketChange),
		)

		// Time on Job (lower is better)
		fmt.Fprintf(w, "    Avg Time on Job\t%.1f hrs\t%.1f hrs\t%s\t\n",
			comp.PriorPeriod.KPIs.AvgTimeOnJob,
			comp.CurrentPeriod.KPIs.AvgTimeOnJob,
			formatHoursChangeInverse(comp.Changes.AvgTimeOnJobChange),
		)

		// Estimates per Job
		fmt.Fprintf(w, "    Estimates/Job\t%.1f\t%.1f\t%s\t\n",
			comp.PriorPeriod.KPIs.AvgEstimatesPerJob,
			comp.CurrentPeriod.KPIs.AvgEstimatesPerJob,
			formatFloatChange(comp.Changes.EstimatesPerJobChange),
		)

		w.Flush()
		fmt.Println()
	}
}

// -----------------------------------------------------------------
// Subcommand: teams
// -----------------------------------------------------------------

func (tc *TechnicianCommand) runTeams(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("teams", flag.ExitOnError)
	fromStr := fs.String("from", "", "Start date (YYYY-MM-DD)")
	toStr := fs.String("to", "", "End date (YYYY-MM-DD)")
	showTechs := fs.Bool("show-technicians", false, "Show individual technicians within each team")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Build filter
	filter := analytics.TechnicianFilter{}

	if *fromStr != "" && *toStr != "" {
		from, err := time.Parse("2006-01-02", *fromStr)
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
		to, err := time.Parse("2006-01-02", *toStr)
		if err != nil {
			return fmt.Errorf("invalid to date: %w", err)
		}
		filter.DateRange = &analytics.DateRange{From: from, To: to}
	}

	// Get team KPIs
	teams, err := tc.svc.GetTeamKPIs(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get team KPIs: %w", err)
	}

	// Print results
	tc.printTeamsTable(teams, filter, *showTechs)

	return nil
}

func (tc *TechnicianCommand) printTeamsTable(teams []analytics.TeamKPIs, filter analytics.TechnicianFilter, showTechs bool) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("                                    TEAM KPI SUMMARY")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")

	if filter.DateRange != nil {
		fmt.Printf("  Date Range: %s to %s\n", filter.DateRange.From.Format("2006-01-02"), filter.DateRange.To.Format("2006-01-02"))
	} else {
		fmt.Println("  Date Range: All Time")
	}
	fmt.Println("───────────────────────────────────────────────────────────────────────────────────────────────────────────────")
	fmt.Println()

	if len(teams) == 0 {
		fmt.Println("  No data found.")
		fmt.Println()
		return
	}

	for _, team := range teams {
		fmt.Printf("  ▶ %s (Team)\n", team.BusinessUnit)
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		// Team aggregate row
		fmt.Fprintln(w, "    \tJobs\tCallback %\tConversion %\tAvg Ticket\tAvg Hours\tTechs\t")
		fmt.Fprintln(w, "    \t────\t──────────\t────────────\t──────────\t─────────\t─────\t")
		fmt.Fprintf(w, "    TEAM TOTAL\t%d\t%.1f%%\t%.1f%%\t$%.0f\t%.1f\t%d\t\n",
			team.Aggregated.TotalJobsCompleted,
			team.Aggregated.CallbackRate,
			team.Aggregated.ConversionRate,
			team.Aggregated.AvgTicket,
			team.Aggregated.AvgTimeOnJob,
			len(team.Technicians),
		)

		w.Flush()

		if showTechs && len(team.Technicians) > 0 {
			fmt.Println()
			fmt.Println("    Individual Technicians:")

			w2 := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w2, "      Name\tJobs\tCallback %\tConversion %\tAvg Ticket\tAvg Hours\t")
			fmt.Fprintln(w2, "      ────\t────\t──────────\t────────────\t──────────\t─────────\t")

			for _, tech := range team.Technicians {
				fmt.Fprintf(w2, "      %s\t%d\t%.1f%%\t%.1f%%\t$%.0f\t%.1f\t\n",
					truncateName(tech.TechnicianName, 20),
					tech.TotalJobsCompleted,
					tech.CallbackRate,
					tech.ConversionRate,
					tech.AvgTicket,
					tech.AvgTimeOnJob,
				)
			}

			w2.Flush()
		}

		fmt.Println()
	}
}

// -----------------------------------------------------------------
// Subcommand: list
// -----------------------------------------------------------------

func (tc *TechnicianCommand) runList(ctx context.Context, args []string) error {
	// Get all technician KPIs (just to get the list with business units)
	kpis, err := tc.svc.GetTechnicianKPIs(ctx, analytics.TechnicianFilter{})
	if err != nil {
		return fmt.Errorf("failed to get technicians: %w", err)
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("                                    TECHNICIANS")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println()

	if len(kpis) == 0 {
		fmt.Println("  No technicians found.")
		fmt.Println()
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "  ID\tName\tBusiness Unit\tTotal Jobs\t")
	fmt.Fprintln(w, "  ──\t────\t─────────────\t──────────\t")

	for _, kpi := range kpis {
		fmt.Fprintf(w, "  %d\t%s\t%s\t%d\t\n",
			kpi.TechnicianID,
			kpi.TechnicianName,
			kpi.BusinessUnit,
			kpi.TotalJobsCompleted,
		)
	}

	w.Flush()
	fmt.Println()
	fmt.Printf("  Total: %d technicians\n", len(kpis))
	fmt.Println()

	return nil
}

// -----------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------

func parseSortField(s string) analytics.SortField {
	switch strings.ToLower(s) {
	case "callback_rate", "callback":
		return analytics.SortByCallbackRate
	case "time_on_job", "time", "hours":
		return analytics.SortByTimeOnJob
	case "conversion_rate", "conversion":
		return analytics.SortByConversionRate
	case "avg_ticket", "ticket", "revenue":
		return analytics.SortByAvgTicket
	case "estimates_per_job", "estimates":
		return analytics.SortByEstimatesPerJob
	case "total_jobs", "jobs":
		return analytics.SortByTotalJobs
	default:
		return analytics.SortByTotalJobs
	}
}

func parsePeriodType(s string) analytics.PeriodType {
	switch strings.ToLower(s) {
	case "weekly", "week", "w":
		return analytics.PeriodWeekly
	case "monthly", "month", "m":
		return analytics.PeriodMonthly
	case "quarterly", "quarter", "q":
		return analytics.PeriodQuarterly
	case "yearly", "year", "y":
		return analytics.PeriodYearly
	default:
		return analytics.PeriodMonthly
	}
}

func truncateName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-3] + "..."
}

func metricLabel(metric string) string {
	switch metric {
	case "callback_rate":
		return "Callback %"
	case "conversion_rate":
		return "Conversion %"
	case "avg_ticket":
		return "Avg Ticket"
	case "time_on_job":
		return "Avg Hours"
	default:
		return "Value"
	}
}

func formatMetricValue(kpi analytics.TechnicianKPIs, metric string) string {
	switch metric {
	case "callback_rate":
		return fmt.Sprintf("%.1f%%", kpi.CallbackRate)
	case "conversion_rate":
		return fmt.Sprintf("%.1f%%", kpi.ConversionRate)
	case "avg_ticket":
		return fmt.Sprintf("$%.0f", kpi.AvgTicket)
	case "time_on_job":
		return fmt.Sprintf("%.1f hrs", kpi.AvgTimeOnJob)
	default:
		return ""
	}
}

func formatIntChange(change int) string {
	if change > 0 {
		return fmt.Sprintf("↑ +%d", change)
	} else if change < 0 {
		return fmt.Sprintf("↓ %d", change)
	}
	return "→ 0"
}

func formatFloatChange(change float64) string {
	if change > 0.05 {
		return fmt.Sprintf("↑ +%.1f", change)
	} else if change < -0.05 {
		return fmt.Sprintf("↓ %.1f", change)
	}
	return "→ 0"
}

func formatPercentChange(change float64) string {
	if change > 0.05 {
		return fmt.Sprintf("↑ +%.1f%%", change)
	} else if change < -0.05 {
		return fmt.Sprintf("↓ %.1f%%", change)
	}
	return "→ 0%"
}

func formatPercentChangeInverse(change float64) string {
	// For metrics where lower is better (callback rate)
	if change > 0.05 {
		return fmt.Sprintf("↓ +%.1f%%", change) // worse
	} else if change < -0.05 {
		return fmt.Sprintf("↑ %.1f%%", change) // better
	}
	return "→ 0%"
}

func formatDollarChange(change float64) string {
	if change > 1 {
		return fmt.Sprintf("↑ +$%.0f", change)
	} else if change < -1 {
		return fmt.Sprintf("↓ $%.0f", change)
	}
	return "→ $0"
}

func formatHoursChangeInverse(change float64) string {
	// For metrics where lower is better (time on job)
	if change > 0.05 {
		return fmt.Sprintf("↓ +%.1f hrs", change) // worse
	} else if change < -0.05 {
		return fmt.Sprintf("↑ %.1f hrs", change) // better
	}
	return "→ 0 hrs"
}

// reportTechnicians is called from handleReport for backward compatibility
// with "sta report technicians" command. It delegates to the TechnicianCommand.
func reportTechnicians(ctx context.Context, db *sql.DB, args []string) {
	cmd := NewTechnicianCommand(db)
	if err := cmd.Run(ctx, args); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
