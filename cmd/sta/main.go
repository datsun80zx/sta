package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/datsun80zx/sta.git/internal/api"
)

const usage = `ServiceTitan Analytics (STA) - Profitability Analysis Tool

Usage:
  sta <command> [arguments]

Commands:
  import <jobs.csv> <invoices.csv>    Import data from ServiceTitan exports
  list                                 List all import batches
  report <type> [options]             Generate reports
  technicians <subcommand> [options]  Technician performance analysis
  serve [--port PORT]                 Start the API server for the dashboard

Report Types:
  summary                              Overall profitability summary
  job-types [--from DATE] [--to DATE] Profitability by job type
  campaigns [--from DATE]              Profitability by campaign
  customers [--top N] [--from DATE]    Top/bottom customers by profit
  red-flags <jobs|job-types|customers> Identify problem areas
  technicians [--html] [--output FILE] Technician performance

Technician Subcommands:
  kpis [--from DATE] [--to DATE]      Show KPI scorecards
  trends --period <weekly|monthly|quarterly> --from DATE --to DATE
  yoy --from DATE --to DATE           Year-over-year comparison
  teams [--show-technicians]          Team/business unit KPIs
  list                                 List all technicians

Environment:
  DATABASE_URL    PostgreSQL connection string (required)
                  Example: postgres://user:pass@localhost/dbname?sslmode=disable

Examples:
  sta import jobs_2024.csv invoices_2024.csv
  sta list
  sta report summary --output q4-report.html --from 2024-10-01 --to 2024-12-31
  sta report job-types
  sta report technicians --html
  sta technicians kpis --from 2024-01-01 --to 2024-12-31
  sta technicians trends --period monthly --from 2024-01-01 --to 2024-12-31
  sta technicians yoy --from 2025-01-01 --to 2025-03-31
  sta serve --port 8080
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
	case "technicians", "tech":
		handleTechnicians(ctx, db, os.Args[2:])
	case "serve", "server":
		handleServe(db, os.Args[2:])
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
		fmt.Println("Available reports: summary, job-types, campaigns, customers, red-flags, technicians")
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
		fmt.Println("Available reports: summary, job-types, campaigns, customers, red-flags, technicians")
		os.Exit(1)
	}
}

func handleTechnicians(ctx context.Context, db *sql.DB, args []string) {
	cmd := NewTechnicianCommand(db)
	if err := cmd.Run(ctx, args); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func handleServe(db *sql.DB, args []string) {
	// Parse port
	port := "8080"
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			port = args[i+1]
			i++
		}
	}

	addr := ":" + port

	// Create and start API server
	server := api.NewServer(db)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nShutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			fmt.Printf("Error shutting down: %v\n", err)
		}
	}()

	fmt.Printf("Starting API server on http://localhost%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()
	fmt.Println("Available endpoints:")
	fmt.Println("  GET /api/health              - Health check")
	fmt.Println("  GET /api/business-units      - List all business units with metrics")
	fmt.Println("  GET /api/business-units/{id} - Get business unit details")
	fmt.Println("  GET /api/technicians/{id}    - Get technician details")
	fmt.Println()

	if err := server.Start(addr); err != nil && err.Error() != "http: Server closed" {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
