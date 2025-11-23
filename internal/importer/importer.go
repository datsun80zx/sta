package importer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shopspring/decimal"

	"github.com/datsun80zx/sta.git/internal/db"
	"github.com/datsun80zx/sta.git/internal/parser"
)

// Importer handles the import of ServiceTitan data
type Importer struct {
	db      *sql.DB
	queries *db.Queries
}

// NewImporter creates a new importer instance
func NewImporter(database *sql.DB) *Importer {
	return &Importer{
		db:      database,
		queries: db.New(database), // FIX: Pass database to db.New()
	}
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	BatchID           int64
	JobsImported      int
	InvoicesImported  int
	CustomersUpserted int
	MetricsCalculated int
	ValidationResult  *ValidationResult
	Duration          time.Duration
	AlreadyImported   bool
}

// ImportFiles imports both jobs and invoices CSV files
func (i *Importer) ImportFiles(ctx context.Context, jobsPath, invoicesPath string) (*ImportResult, error) {
	startTime := time.Now()

	// Step 1: Calculate file hashes
	jobsHash, invoicesHash, err := CalculateFileHashes(jobsPath, invoicesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hashes: %w", err)
	}

	// Step 2: Check if already imported
	existingBatch, err := i.queries.GetImportBatchByHashes(ctx, db.GetImportBatchByHashesParams{ // FIX: Remove i.db parameter
		JobReportHash:     jobsHash,
		InvoiceReportHash: invoicesHash,
	})
	if err == nil {
		// Already imported
		return &ImportResult{
			BatchID:         existingBatch.ID,
			AlreadyImported: true,
			Duration:        time.Since(startTime),
		}, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check for existing import: %w", err)
	}

	// Step 3: Parse files
	jobs, invoices, err := i.parseFiles(jobsPath, invoicesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse files: %w", err)
	}

	// Step 4: Start transaction
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// FIX: Create transaction-scoped queries object
	txQueries := db.New(tx)

	// Step 5: Create import batch
	batch, err := txQueries.CreateImportBatch(ctx, db.CreateImportBatchParams{ // FIX: Remove tx parameter, use txQueries
		JobReportFilename:     filepath.Base(jobsPath),
		InvoiceReportFilename: filepath.Base(invoicesPath),
		JobReportHash:         jobsHash,
		InvoiceReportHash:     invoicesHash,
		RowCountJobs:          int32(len(jobs)),
		RowCountInvoices:      int32(len(invoices)),
		Status:                "pending",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create import batch: %w", err)
	}

	// Step 6: Import customers (upsert from job data)
	customersUpserted, err := i.importCustomers(ctx, tx, jobs)
	if err != nil {
		txQueries.UpdateImportBatchStatus(ctx, db.UpdateImportBatchStatusParams{ // FIX: Remove tx parameter, use txQueries
			ID:           batch.ID,
			Status:       "failed",
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
		return nil, fmt.Errorf("failed to import customers: %w", err)
	}

	// Step 7: Import jobs
	err = i.importJobs(ctx, tx, jobs, batch.ID)
	if err != nil {
		txQueries.UpdateImportBatchStatus(ctx, db.UpdateImportBatchStatusParams{ // FIX: Remove tx parameter, use txQueries
			ID:           batch.ID,
			Status:       "failed",
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
		return nil, fmt.Errorf("failed to import jobs: %w", err)
	}

	// Step 8: Import invoices
	err = i.importInvoices(ctx, tx, invoices, batch.ID)
	if err != nil {
		txQueries.UpdateImportBatchStatus(ctx, db.UpdateImportBatchStatusParams{ // FIX: Remove tx parameter, use txQueries
			ID:           batch.ID,
			Status:       "failed",
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
		return nil, fmt.Errorf("failed to import invoices: %w", err)
	}

	// Step 9: Validate data
	validationResult, err := ValidateImport(ctx, tx, batch.ID)
	if err != nil {
		txQueries.UpdateImportBatchStatus(ctx, db.UpdateImportBatchStatusParams{ // FIX: Remove tx parameter, use txQueries
			ID:           batch.ID,
			Status:       "failed",
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Step 10: Calculate job metrics
	err = txQueries.CalculateJobMetrics(ctx, batch.ID) // FIX: Remove tx parameter, use txQueries
	if err != nil {
		txQueries.UpdateImportBatchStatus(ctx, db.UpdateImportBatchStatusParams{ // FIX: Remove tx parameter, use txQueries
			ID:           batch.ID,
			Status:       "failed",
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
		return nil, fmt.Errorf("failed to calculate metrics: %w", err)
	}

	// Step 11: Mark batch as success
	err = txQueries.UpdateImportBatchStatus(ctx, db.UpdateImportBatchStatusParams{ // FIX: Remove tx parameter, use txQueries
		ID:           batch.ID,
		Status:       "success",
		ErrorMessage: sql.NullString{Valid: false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update batch status: %w", err)
	}

	// Step 12: Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	metricsCount := len(jobs)

	return &ImportResult{
		BatchID:           batch.ID,
		JobsImported:      len(jobs),
		InvoicesImported:  len(invoices),
		CustomersUpserted: customersUpserted,
		MetricsCalculated: metricsCount,
		ValidationResult:  validationResult,
		Duration:          time.Since(startTime),
		AlreadyImported:   false,
	}, nil
}

// parseFiles parses both CSV files
func (i *Importer) parseFiles(jobsPath, invoicesPath string) ([]parser.JobRow, []parser.InvoiceRow, error) {
	csvParser := parser.NewCSVParser()

	// Parse jobs file
	jobsFile, err := os.Open(jobsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open jobs file: %w", err)
	}
	defer jobsFile.Close()

	jobs, err := csvParser.ParseJobs(jobsFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse jobs: %w", err)
	}

	// Parse invoices file
	invoicesFile, err := os.Open(invoicesPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open invoices file: %w", err)
	}
	defer invoicesFile.Close()

	invoices, err := csvParser.ParseInvoices(invoicesFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse invoices: %w", err)
	}

	return jobs, invoices, nil
}

// importCustomers upserts customer records from job data
func (i *Importer) importCustomers(ctx context.Context, tx *sql.Tx, jobs []parser.JobRow) (int, error) {
	// FIX: Create transaction-scoped queries object
	txQueries := db.New(tx)

	// Build unique set of customers
	customerMap := make(map[int64]*parser.JobRow)
	for idx := range jobs {
		job := &jobs[idx]
		if existing, ok := customerMap[job.CustomerID]; ok {
			// Keep the most recent job's customer data
			if job.JobCompletionDate != nil && existing.JobCompletionDate != nil {
				if job.JobCompletionDate.After(*existing.JobCompletionDate) {
					customerMap[job.CustomerID] = job
				}
			}
		} else {
			customerMap[job.CustomerID] = job
		}
	}

	// Upsert each customer
	count := 0
	for customerID, job := range customerMap {
		// Determine first and last job dates for this customer
		var firstJobDate, lastJobDate *time.Time
		for _, j := range jobs {
			if j.CustomerID == customerID && j.JobCompletionDate != nil {
				if firstJobDate == nil || j.JobCompletionDate.Before(*firstJobDate) {
					firstJobDate = j.JobCompletionDate
				}
				if lastJobDate == nil || j.JobCompletionDate.After(*lastJobDate) {
					lastJobDate = j.JobCompletionDate
				}
			}
		}

		params := db.UpsertCustomerParams{
			ID:            customerID,
			CustomerName:  stringOrEmpty(job.CustomerName),
			CustomerType:  sqlNullString(job.CustomerType),
			CustomerCity:  sqlNullString(job.CustomerCity),
			CustomerState: sqlNullString(job.CustomerState),
			CustomerZip:   sqlNullString(job.CustomerZip),
			LocationCity:  sqlNullString(job.LocationCity),
			LocationState: sqlNullString(job.LocationState),
			LocationZip:   sqlNullString(job.LocationZip),
			FirstJobDate:  sqlNullTime(firstJobDate),
			LastJobDate:   sqlNullTime(lastJobDate),
		}

		_, err := txQueries.UpsertCustomer(ctx, params) // FIX: Remove tx parameter, use txQueries
		if err != nil {
			return count, fmt.Errorf("failed to upsert customer %d: %w", customerID, err)
		}
		count++
	}

	return count, nil
}

// importJobs inserts job records
func (i *Importer) importJobs(ctx context.Context, tx *sql.Tx, jobs []parser.JobRow, batchID int64) error {
	// FIX: Create transaction-scoped queries object
	txQueries := db.New(tx)

	for idx, job := range jobs {
		params := db.CreateJobParams{
			ID:                 job.JobID,
			CustomerID:         job.CustomerID,
			ImportBatchID:      batchID,
			JobType:            job.JobType,
			BusinessUnit:       sqlNullString(job.BusinessUnit),
			Status:             job.Status,
			JobCreationDate:    sqlNullTime(job.JobCreationDate),
			JobScheduleDate:    sqlNullTime(job.JobScheduleDate),
			JobCompletionDate:  sqlNullTime(job.JobCompletionDate),
			AssignedTechnician: sqlNullString(job.AssignedTechnicians),
			SoldByTechnician:   sqlNullString(job.SoldBy),
			BookedBy:           sqlNullString(job.BookedBy),
			CampaignName:       sqlNullString(stringFromInt64Ptr(job.JobCampaignID)),
			CampaignCategory:   sqlNullString(job.CampaignCategory),
			CallCampaign:       sqlNullString(stringFromInt64Ptr(job.CallCampaignID)),
			JobsSubtotal:       decimalOrZero(job.JobsSubtotal),
			JobTotal:           decimalOrZero(job.JobTotal),
			InvoiceID:          sqlNullInt64(job.InvoiceID),
			TotalHoursWorked:   decimalOrZero(job.TotalHoursWorked),
			Priority:           sqlNullString(job.Priority),
			SurveyScore:        sqlNullInt32FromDecimal(job.SurveyResult),
		}

		_, err := txQueries.CreateJob(ctx, params) // FIX: Remove tx parameter, use txQueries
		if err != nil {
			return fmt.Errorf("failed to insert job %d (row %d): %w", job.JobID, idx+2, err)
		}
	}

	return nil
}

// importInvoices inserts invoice records
func (i *Importer) importInvoices(ctx context.Context, tx *sql.Tx, invoices []parser.InvoiceRow, batchID int64) error {
	// FIX: Create transaction-scoped queries object
	txQueries := db.New(tx)

	for idx, invoice := range invoices {
		params := db.CreateInvoiceParams{
			ID:                 invoice.InvoiceID,
			JobID:              invoice.JobID,
			ImportBatchID:      batchID,
			InvoiceDate:        invoice.InvoiceDate,
			InvoiceStatus:      sqlNullString(invoice.InvoiceStatus),
			InvoiceType:        sqlNullString(invoice.InvoiceType),
			InvoiceSummary:     sqlNullString(invoice.InvoiceSummary),
			Total:              decimalOrZero(invoice.Total),
			Balance:            decimalOrZero(invoice.Balance),
			Payments:           decimalOrZero(invoice.Payments),
			MaterialCosts:      decimalOrZero(invoice.MaterialCosts),
			EquipmentCosts:     decimalOrZero(invoice.EquipmentCosts),
			PurchaseOrderCosts: decimalOrZero(invoice.PurchaseOrderCosts),
			ReturnCosts:        decimalOrZero(invoice.ReturnCosts),
			CostsTotal:         decimalOrZero(invoice.CostsTotal),
			MaterialRetail:     decimalOrZero(invoice.MaterialRetail),
			MaterialMarkup:     decimalOrZero(invoice.MaterialMarkup),
			EquipmentRetail:    decimalOrZero(invoice.EquipmentRetail),
			EquipmentMarkup:    decimalOrZero(invoice.EquipmentMarkup),
			Labor:              decimalOrZero(invoice.Labor),
			LaborPay:           decimalOrZero(invoice.LaborPay),
			LaborBurden:        decimalOrZero(invoice.LaborBurden),
			TotalLaborCosts:    decimalOrZero(invoice.TotalLaborCosts),
			Income:             decimalOrZero(invoice.Income),
			DiscountTotal:      decimalOrZero(invoice.DiscountTotal),
			IsAdjustment:       invoice.IsAdjustment,
		}

		_, err := txQueries.CreateInvoice(ctx, params) // FIX: Remove tx parameter, use txQueries
		if err != nil {
			return fmt.Errorf("failed to insert invoice %d (row %d): %w", invoice.InvoiceID, idx+2, err)
		}
	}

	return nil
}

// Helper functions for converting types

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func sqlNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func sqlNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// func sqlNullInt32(i *int32) sql.NullInt32 {
// 	if i == nil {
// 		return sql.NullInt32{Valid: false}
// 	}
// 	return sql.NullInt32{Int32: *i, Valid: true}
// }

func sqlNullInt32FromDecimal(d *decimal.Decimal) sql.NullInt32 {
	if d == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(d.IntPart()), Valid: true}
}

func decimalOrZero(d *decimal.Decimal) decimal.Decimal {
	if d == nil {
		return decimal.Zero
	}
	return *d
}

func sqlNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func stringFromInt64Ptr(i *int64) *string {
	if i == nil {
		return nil
	}
	s := fmt.Sprintf("%d", *i)
	return &s
}
