package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

const usage = `ServiceTitan Profitability Analysis Tool

Usage:
  sta import <jobs.csv> <invoices.csv>     Import ServiceTitan reports
  sta list                                  List import history
  sta report summary [--output FILE] [--from DATE] [--to DATE]
                                            Generate HTML profitability report
  sta report job-types [--from DATE] [--to DATE]
                                            Show profitability by job type
  sta report campaigns [--from DATE] [--to DATE]
                                            Show profitability by campaign
  sta report customers [--top N] [--from DATE] [--to DATE]
                                            Show top customers by profit
  sta report red-flags <type> [options]     Identify profitability problems
                                            Types: jobs, job-types, customers, high-revenue
  sta report technicians [type]             Technician performance reports
                                            Types: overview, sales, conversion, efficiency	

Date Filtering:
  --from YYYY-MM-DD    Include jobs completed on or after this date
  --to YYYY-MM-DD      Include jobs completed on or before this date

Output Options:
  --output FILE        Write report to FILE (default: profitability-report-DATE.html)

Database Configuration:
  Set DATABASE_URL environment variable:
    export DATABASE_URL="postgres://user:pass@localhost/dbname?sslmode=disable"

Examples:
  sta import jobs_2024.csv invoices_2024.csv
  sta list
  sta report summary --output q4-report.html --from 2024-10-01 --to 2024-12-31
  sta report job-types
  sta report job-types --from 2024-01-01 --to 2024-06-30
  sta report campaigns --from 2024-07-01
  sta report customers --top 20 --from 2024-01-01
  sta report red-flags jobs
  sta report red-flags job-types --margin-threshold 15
  sta report red-flags customers --from 2024-11-01
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("Error: DATABASE_URL environment variable not set")
		fmt.Println("\nExample:")
		fmt.Println(`  export DATABASE_URL="postgres://user:pass@localhost/dbname?sslmode=disable"`)
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	command := os.Args[1]

	switch command {
	case "import":
		handleImport(ctx, db, os.Args[2:])
	case "list":
		handleList(ctx, db)
	case "report":
		handleReport(ctx, db, os.Args[2:])
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		fmt.Print(usage)
		os.Exit(1)
	}
}

func handleImport(ctx context.Context, db *sql.DB, args []string) {
	if len(args) < 2 {
		fmt.Println("Error: import requires two arguments")
		fmt.Println("Usage: sta import <jobs.csv> <invoices.csv>")
		os.Exit(1)
	}

	jobsPath := args[0]
	invoicesPath := args[1]

	// Check files exist
	if _, err := os.Stat(jobsPath); os.IsNotExist(err) {
		fmt.Printf("Error: jobs file not found: %s\n", jobsPath)
		os.Exit(1)
	}
	if _, err := os.Stat(invoicesPath); os.IsNotExist(err) {
		fmt.Printf("Error: invoices file not found: %s\n", invoicesPath)
		os.Exit(1)
	}

	runImport(ctx, db, jobsPath, invoicesPath)
}

func handleList(ctx context.Context, db *sql.DB) {
	listImports(ctx, db)
}

func handleReport(ctx context.Context, db *sql.DB, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: report requires a report type")
		fmt.Println("Available reports: summary, job-types, campaigns, customers, red-flags")
		os.Exit(1)
	}

	reportType := args[0]
	reportArgs := args[1:]

	switch reportType {
	case "summary":
		reportSummary(ctx, db, reportArgs)
	case "job-types":
		reportJobTypes(ctx, db, reportArgs)
	case "campaigns":
		reportCampaigns(ctx, db, reportArgs)
	case "customers":
		reportCustomers(ctx, db, reportArgs)
	case "red-flags":
		handleRedFlags(ctx, db, reportArgs)
	case "technicians":
		reportTechnicians(ctx, db, reportArgs)
	default:
		fmt.Printf("Unknown report type: %s\n", reportType)
		fmt.Println("Available reports: summary, job-types, campaigns, customers, red-flags")
		os.Exit(1)
	}
}
