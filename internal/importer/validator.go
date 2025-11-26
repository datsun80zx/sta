package importer

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/datsun80zx/sta.git/internal/db"
)

// ValidationResult contains validation warnings and errors
type ValidationResult struct {
	JobsWithoutInvoices []string
	Warnings            []string
}

// ValidateImport checks data quality after import
func ValidateImport(ctx context.Context, tx *sql.Tx, batchID int64) (*ValidationResult, error) {
	queries := db.New(tx) // FIX: Pass tx to db.New()

	result := &ValidationResult{
		JobsWithoutInvoices: make([]string, 0),
		Warnings:            make([]string, 0),
	}

	// Check for jobs without invoices
	jobsWithoutInvoices, err := queries.GetJobsWithoutInvoices(ctx, batchID) // FIX: Remove tx parameter - it's already in the queries object
	if err != nil {
		return nil, fmt.Errorf("failed to check jobs without invoices: %w", err)
	}

	if len(jobsWithoutInvoices) > 0 {
		for _, job := range jobsWithoutInvoices {
			result.JobsWithoutInvoices = append(result.JobsWithoutInvoices, job.ID)
		}
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Found %d jobs without invoices", len(jobsWithoutInvoices)))
	}

	return result, nil
}
