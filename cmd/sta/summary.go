package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/datsun80zx/sta.git/internal/report"
)

// parseOutputFlag extracts --output flag from args
func parseOutputFlag(args []string) (string, []string) {
	var output string
	var remainingArgs []string

	i := 0
	for i < len(args) {
		if args[i] == "--output" && i+1 < len(args) {
			output = args[i+1]
			i += 2
		} else if strings.HasPrefix(args[i], "--output=") {
			output = strings.TrimPrefix(args[i], "--output=")
			i++
		} else {
			remainingArgs = append(remainingArgs, args[i])
			i++
		}
	}

	return output, remainingArgs
}

func reportSummary(ctx context.Context, db *sql.DB, args []string) {
	output, remainingArgs := parseOutputFlag(args)
	fromDate, toDate, _ := parseDateFlags(remainingArgs)

	// Default output filename if not specified
	if output == "" {
		timestamp := time.Now().Format("2006-01-02")
		output = fmt.Sprintf("profitability-report-%s.html", timestamp)
	}

	// Ensure .html extension
	if !strings.HasSuffix(strings.ToLower(output), ".html") {
		output += ".html"
	}

	fmt.Println("Generating profitability report...")
	if fromDate != nil || toDate != nil {
		fmt.Print("  Date range: ")
		if fromDate != nil {
			fmt.Print(fromDate.Format("2006-01-02"))
		} else {
			fmt.Print("(all)")
		}
		fmt.Print(" to ")
		if toDate != nil {
			fmt.Print(toDate.Format("2006-01-02"))
		} else {
			fmt.Print("(all)")
		}
		fmt.Println()
	}
	fmt.Println()

	// Generate report data
	summary, err := report.GenerateSummary(ctx, db, fromDate, toDate)
	if err != nil {
		fmt.Printf("❌ Error generating report: %v\n", err)
		return
	}

	// Create renderer
	renderer, err := report.NewRenderer()
	if err != nil {
		fmt.Printf("❌ Error initializing renderer: %v\n", err)
		return
	}

	// Create output file
	file, err := os.Create(output)
	if err != nil {
		fmt.Printf("❌ Error creating output file: %v\n", err)
		return
	}
	defer file.Close()

	// Render report
	if err := renderer.RenderSummary(file, summary); err != nil {
		fmt.Printf("❌ Error rendering report: %v\n", err)
		return
	}

	absPath, _ := filepath.Abs(output)
	fmt.Printf("✅ Report generated: %s\n", absPath)
	fmt.Println()
	fmt.Println("📊 Report Summary:")
	fmt.Printf("   • %d completed jobs analyzed\n", summary.TotalJobs)
	fmt.Printf("   • %s total revenue\n", formatCurrency(summary.TotalRevenue))
	fmt.Printf("   • %s total profit (%.1f%% margin)\n", formatCurrency(summary.TotalProfit), summary.AvgMarginPct)
	if summary.JobsWithLoss > 0 {
		fmt.Printf("   • ⚠️  %d jobs with losses totaling %s\n", summary.JobsWithLoss, formatCurrency(-summary.TotalLoss))
	}
	fmt.Println()
	fmt.Println("💡 Open the HTML file in your browser and print to PDF (Cmd+P / Ctrl+P)")
}

func formatCurrency(amount float64) string {
	if amount < 0 {
		return fmt.Sprintf("($%.2f)", -amount)
	}
	return fmt.Sprintf("$%.2f", amount)
}
