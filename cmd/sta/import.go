package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/datsun80zx/sta.git/internal/importer"
)

func runImport(ctx context.Context, db *sql.DB, jobsPath, invoicesPath string) {
	fmt.Println("Starting import...")
	fmt.Printf("  Jobs file:     %s\n", jobsPath)
	fmt.Printf("  Invoices file: %s\n", invoicesPath)
	fmt.Println()

	imp := importer.NewImporter(db)

	result, err := imp.ImportFiles(ctx, jobsPath, invoicesPath)
	if err != nil {
		fmt.Printf("❌ Import failed: %v\n", err)
		return
	}

	if result.AlreadyImported {
		fmt.Println("ℹ️  These files have already been imported")
		fmt.Printf("   Batch ID: %d\n", result.BatchID)
		return
	}

	fmt.Println("✅ Import successful!")
	fmt.Println()
	fmt.Printf("Batch ID:           %d\n", result.BatchID)
	fmt.Printf("Jobs imported:      %d\n", result.JobsImported)
	fmt.Printf("Invoices imported:  %d\n", result.InvoicesImported)
	if result.InvoicesSkipped > 0 {
		fmt.Printf("Invoices skipped:   %d (no matching job)\n", result.InvoicesSkipped)
	}
	fmt.Printf("Customers upserted: %d\n", result.CustomersUpserted)
	fmt.Printf("Metrics calculated: %d\n", result.MetricsCalculated)
	fmt.Printf("Duration:           %v\n", result.Duration.Round(time.Millisecond))

	if result.ValidationResult != nil && len(result.ValidationResult.Warnings) > 0 {
		fmt.Println()
		fmt.Println("⚠️  Warnings:")
		for _, warning := range result.ValidationResult.Warnings {
			fmt.Printf("   - %s\n", warning)
		}
	}

	fmt.Println()
	fmt.Println("💡 Next steps:")
	fmt.Println("   sta report job-types     # View profitability by job type")
	fmt.Println("   sta report campaigns     # View profitability by campaign")
	fmt.Println("   sta report customers     # View top customers by profit")
}
