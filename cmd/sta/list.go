package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/datsun80zx/sta.git/internal/db"
)

func listImports(ctx context.Context, database *sql.DB) {
	queries := db.New(database)

	batches, err := queries.ListImportBatches(ctx, 20)
	if err != nil {
		fmt.Printf("Error listing imports: %v\n", err)
		return
	}

	if len(batches) == 0 {
		fmt.Println("No imports found")
		fmt.Println()
		fmt.Println("💡 Import your first batch with:")
		fmt.Println("   sta import jobs.csv invoices.csv")
		return
	}

	fmt.Println("Import History")
	fmt.Println("══════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("%-4s  %-19s  %-8s  %8s  %10s  %-30s\n",
		"ID", "Date", "Status", "Jobs", "Invoices", "Job Report File")
	fmt.Println("──────────────────────────────────────────────────────────────────────────────")

	for _, batch := range batches {
		dateStr := batch.ImportedAt.Format("2006-01-02 15:04:05")
		status := batch.Status
		statusIcon := "✅"
		if status == "failed" {
			statusIcon = "❌"
		} else if status == "pending" {
			statusIcon = "⏳"
		}

		// Truncate filename if too long
		filename := batch.JobReportFilename
		if len(filename) > 30 {
			filename = filename[:27] + "..."
		}

		fmt.Printf("%-4d  %s  %s %-6s  %8d  %10d  %-30s\n",
			batch.ID,
			dateStr,
			statusIcon,
			status,
			batch.RowCountJobs,
			batch.RowCountInvoices,
			filename,
		)
	}
	fmt.Println("══════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("Total: %d import(s)\n", len(batches))
}
