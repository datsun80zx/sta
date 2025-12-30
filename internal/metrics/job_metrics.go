package metrics

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"
)

// JobMetric represents calculated metrics for a single job
type JobMetric struct {
	JobID          string
	Revenue        decimal.Decimal
	TotalCosts     decimal.Decimal
	GrossProfit    decimal.Decimal
	GrossMarginPct decimal.NullDecimal
	InvoiceCount   int
	HasAdjustment  bool
}

// InvoiceData holds the invoice fields needed for calculations
type InvoiceData struct {
	ID           string
	JobID        string
	CostsTotal   decimal.Decimal
	IsAdjustment bool
}

// JobData holds the job fields needed for calculations
type JobData struct {
	ID           string
	Status       string
	JobsSubtotal decimal.Decimal
}

// CalculateJobMetrics computes profitability metrics for all jobs in a batch
func CalculateJobMetrics(jobs []JobData, invoices []InvoiceData) []JobMetric {
	// Group invoices by job ID
	invoicesByJob := make(map[string][]InvoiceData)
	for _, inv := range invoices {
		invoicesByJob[inv.JobID] = append(invoicesByJob[inv.JobID], inv)
	}

	var results []JobMetric

	for _, job := range jobs {
		// Only calculate for completed jobs with revenue
		if job.Status != "Completed" {
			continue
		}
		if job.JobsSubtotal.IsZero() {
			continue
		}

		jobInvoices := invoicesByJob[job.ID]
		if len(jobInvoices) == 0 {
			continue
		}

		metric := calculateSingleJobMetric(job, jobInvoices)
		results = append(results, metric)
	}

	return results
}

func calculateSingleJobMetric(job JobData, invoices []InvoiceData) JobMetric {
	metric := JobMetric{
		JobID:        job.ID,
		Revenue:      job.JobsSubtotal,
		InvoiceCount: len(invoices),
	}

	// Check for adjustment invoices
	var adjustmentInvoice *InvoiceData
	for i := range invoices {
		if invoices[i].IsAdjustment {
			metric.HasAdjustment = true
			adjustmentInvoice = &invoices[i]
			break // Use first adjustment found (should typically only be one)
		}
	}

	// Calculate total costs
	// If there's an adjustment invoice, use its costs (it replaces the original)
	// Otherwise, sum all non-adjustment invoice costs
	if adjustmentInvoice != nil {
		metric.TotalCosts = adjustmentInvoice.CostsTotal
	} else {
		totalCosts := decimal.Zero
		for _, inv := range invoices {
			if !inv.IsAdjustment {
				totalCosts = totalCosts.Add(inv.CostsTotal)
			}
		}
		metric.TotalCosts = totalCosts
	}

	// Calculate gross profit
	metric.GrossProfit = metric.Revenue.Sub(metric.TotalCosts)

	// Calculate gross margin percentage
	if metric.Revenue.GreaterThan(decimal.Zero) {
		marginPct := metric.GrossProfit.Div(metric.Revenue).Mul(decimal.NewFromInt(100))
		metric.GrossMarginPct = decimal.NullDecimal{
			Decimal: marginPct,
			Valid:   true,
		}
	}

	return metric
}

// SaveJobMetrics persists calculated job metrics to the database
func SaveJobMetrics(ctx context.Context, tx *sql.Tx, metrics []JobMetric) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO job_metrics (job_id, revenue, total_costs, gross_profit, gross_margin_pct, invoice_count, has_adjustment)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (job_id) DO UPDATE SET
			revenue = EXCLUDED.revenue,
			total_costs = EXCLUDED.total_costs,
			gross_profit = EXCLUDED.gross_profit,
			gross_margin_pct = EXCLUDED.gross_margin_pct,
			invoice_count = EXCLUDED.invoice_count,
			has_adjustment = EXCLUDED.has_adjustment,
			calculated_at = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range metrics {
		var marginPct interface{}
		if m.GrossMarginPct.Valid {
			marginPct = m.GrossMarginPct.Decimal
		} else {
			marginPct = nil
		}

		_, err := stmt.ExecContext(ctx,
			m.JobID,
			m.Revenue,
			m.TotalCosts,
			m.GrossProfit,
			marginPct,
			m.InvoiceCount,
			m.HasAdjustment,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
