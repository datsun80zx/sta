package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type CSVParser struct {
	TrimWhitespace bool
	SkipEmptyRows  bool
}

func NewCSVParser() *CSVParser {
	return &CSVParser{
		TrimWhitespace: true,
		SkipEmptyRows:  true,
	}
}

// ParseJobs reads a Jobs CSV and returns parsed rows
func (p *CSVParser) ParseJobs(r io.Reader) ([]JobRow, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	headers := records[0]
	colMap := buildColumnMap(headers)

	jobs := make([]JobRow, 0, len(records)-1)
	for i, record := range records[1:] {
		rowNum := i + 2

		job, err := p.parseJobRow(record, colMap, rowNum)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// ParseInvoices reads an Invoices CSV and returns parsed rows
func (p *CSVParser) ParseInvoices(r io.Reader) ([]InvoiceRow, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	headers := records[0]
	colMap := buildColumnMap(headers)

	invoices := make([]InvoiceRow, 0, len(records)-1)
	for i, record := range records[1:] {
		rowNum := i + 2

		invoice, err := p.parseInvoiceRow(record, colMap, rowNum)
		if err != nil {
			return nil, err
		}

		invoices = append(invoices, invoice)
	}

	return invoices, nil
}

// buildColumnMap creates a case-insensitive map of column name → index
func buildColumnMap(headers []string) map[string]int {
	m := make(map[string]int)
	for i, header := range headers {
		normalized := strings.ToLower(strings.TrimSpace(header))
		m[normalized] = i
	}
	return m
}

// parseJobRow converts a CSV row into a JobRow struct
func (p *CSVParser) parseJobRow(record []string, colMap map[string]int, rowNum int) (JobRow, error) {
	var job JobRow
	var err error

	// Required fields
	job.JobID, err = parseInt64(getField(record, colMap, "job id"), rowNum, "Job ID")
	if err != nil {
		return job, err
	}

	job.CustomerID, err = parseInt64(getField(record, colMap, "customer id"), rowNum, "Customer ID")
	if err != nil {
		return job, err
	}

	job.JobType = getField(record, colMap, "job type")
	if job.JobType == "" {
		return job, &ValidationError{Row: rowNum, Column: "Job Type", Err: fmt.Errorf("required field is empty")}
	}

	job.Status = getField(record, colMap, "status")
	if job.Status == "" {
		return job, &ValidationError{Row: rowNum, Column: "Status", Err: fmt.Errorf("required field is empty")}
	}

	// Revenue fields - Jobs Subtotal is the main revenue field
	subtotalStr := getField(record, colMap, "jobs subtotal")
	if subtotalStr != "" {
		job.JobsSubtotal, err = parseDecimal(subtotalStr, rowNum, "Jobs Subtotal")
		if err != nil {
			return job, err
		}
	} else {
		// Jobs Subtotal can be empty for some jobs (like zero dollar jobs)
		job.JobsSubtotal = nil
	}

	totalStr := getField(record, colMap, "jobs total")
	if totalStr != "" {
		job.JobTotal, err = parseDecimal(totalStr, rowNum, "Jobs Total")
		if err != nil {
			return job, err
		}
	} else {
		job.JobTotal = nil
	}

	// Customer info
	job.CustomerName = parseNullableString(getField(record, colMap, "customer name"))
	job.CustomerType = parseNullableString(getField(record, colMap, "customer type"))
	job.CustomerCity = parseNullableString(getField(record, colMap, "customer city"))
	job.CustomerState = parseNullableString(getField(record, colMap, "customer state"))
	job.CustomerZip = parseNullableString(getField(record, colMap, "customer zip"))

	// Location info
	job.LocationID = parseNullableInt64(getField(record, colMap, "location id"))
	job.LocationCity = parseNullableString(getField(record, colMap, "location city"))
	job.LocationState = parseNullableString(getField(record, colMap, "location state"))
	job.LocationZip = parseNullableString(getField(record, colMap, "location zip"))

	// Business unit
	job.BusinessUnitID = parseNullableInt64(getField(record, colMap, "business unit id"))
	job.BusinessUnit = parseNullableString(getField(record, colMap, "business unit"))

	// Dates
	job.JobCreationDate = parseNullableDate(getField(record, colMap, "created date"))
	job.JobScheduleDate = parseNullableDate(getField(record, colMap, "scheduled date"))
	job.JobCompletionDate = parseNullableDate(getField(record, colMap, "completion date"))

	// People
	job.AssignedTechnicians = parseNullableString(getField(record, colMap, "assigned technicians"))
	job.SoldBy = parseNullableString(getField(record, colMap, "sold by"))
	job.BookedBy = parseNullableString(getField(record, colMap, "booked by"))
	job.DispatchedBy = parseNullableString(getField(record, colMap, "dispatched by"))
	job.PrimaryTechnician = parseNullableString(getField(record, colMap, "primary technician"))

	// Campaign info
	job.JobCampaignID = parseNullableInt64(getField(record, colMap, "job campaign id"))
	job.CallCampaignID = parseNullableInt64(getField(record, colMap, "call campaign id"))
	job.CampaignCategory = parseNullableString(getField(record, colMap, "campaign category"))

	// Invoice reference
	job.InvoiceID = parseNullableInt64(getField(record, colMap, "invoice id"))

	// Other fields
	job.Summary = parseNullableString(getField(record, colMap, "summary"))
	job.Priority = parseNullableString(getField(record, colMap, "priority"))
	job.TotalHoursWorked = parseNullableDecimal(getField(record, colMap, "total hours worked"))
	job.SurveyResult = parseNullableDecimal(getField(record, colMap, "survey result"))
	job.MemberStatus = parseNullableString(getField(record, colMap, "member status"))
	job.Tags = parseNullableString(getField(record, colMap, "tags"))

	// Boolean fields
	job.Opportunity = parseBool(getField(record, colMap, "opportunity"))
	job.Warranty = parseBool(getField(record, colMap, "warranty"))
	job.Recall = parseBool(getField(record, colMap, "recall"))
	job.Converted = parseBool(getField(record, colMap, "converted"))
	job.ZeroDollarJob = parseBool(getField(record, colMap, "zero dollar job"))

	return job, nil
}

// parseInvoiceRow converts a CSV row into an InvoiceRow struct
func (p *CSVParser) parseInvoiceRow(record []string, colMap map[string]int, rowNum int) (InvoiceRow, error) {
	var invoice InvoiceRow
	var err error

	// Required fields
	invoice.InvoiceID, err = parseInt64(getField(record, colMap, "invoice #"), rowNum, "Invoice #")
	if err != nil {
		return invoice, err
	}

	invoice.JobID, err = parseInt64(getField(record, colMap, "job #"), rowNum, "Job #")
	if err != nil {
		return invoice, err
	}

	invoiceDateStr := getField(record, colMap, "invoice date")
	if invoiceDateStr == "" {
		return invoice, &ValidationError{Row: rowNum, Column: "Invoice Date", Err: fmt.Errorf("required field is empty")}
	}
	invoiceDate := parseNullableDate(invoiceDateStr)
	if invoiceDate == nil {
		return invoice, &ValidationError{Row: rowNum, Column: "Invoice Date", Value: invoiceDateStr, Err: fmt.Errorf("invalid date format")}
	}
	invoice.InvoiceDate = *invoiceDate

	// Total is required
	totalStr := getField(record, colMap, "total")
	if totalStr != "" {
		invoice.Total, err = parseDecimal(totalStr, rowNum, "Total")
		if err != nil {
			return invoice, err
		}
	} else {
		invoice.Total = nil
	}

	// Optional fields
	invoice.ProjectNumber = parseNullableInt64(getField(record, colMap, "project number"))
	invoice.InvoiceStatus = parseNullableString(getField(record, colMap, "invoice status"))
	invoice.InvoiceBusinessUnitID = parseNullableInt64(getField(record, colMap, "invoice business unit id"))
	invoice.InvoiceType = parseNullableString(getField(record, colMap, "invoice type"))
	invoice.InvoiceSummary = parseNullableString(getField(record, colMap, "invoice summary"))

	invoice.Balance = parseNullableDecimal(getField(record, colMap, "balance"))
	invoice.Payments = parseNullableDecimal(getField(record, colMap, "payments"))
	invoice.PaymentTypes = parseNullableString(getField(record, colMap, "payment types"))
	invoice.PaymentTerm = parseNullableString(getField(record, colMap, "payment term"))

	// Cost fields - these are critical for profitability
	invoice.MaterialCosts = parseNullableDecimal(getField(record, colMap, "material costs"))
	invoice.EquipmentCosts = parseNullableDecimal(getField(record, colMap, "equipment costs"))
	invoice.PurchaseOrderCosts = parseNullableDecimal(getField(record, colMap, "purchase order costs"))
	invoice.ReturnCosts = parseNullableDecimal(getField(record, colMap, "return costs"))
	invoice.CostsTotal = parseNullableDecimal(getField(record, colMap, "costs total"))

	// Retail/Markup
	invoice.MaterialRetail = parseNullableDecimal(getField(record, colMap, "material retail"))
	invoice.MaterialMarkup = parseNullableDecimal(getField(record, colMap, "material markup"))
	invoice.EquipmentRetail = parseNullableDecimal(getField(record, colMap, "equipment retail"))
	invoice.EquipmentMarkup = parseNullableDecimal(getField(record, colMap, "equipment markup"))
	invoice.Labor = parseNullableDecimal(getField(record, colMap, "labor"))
	invoice.Income = parseNullableDecimal(getField(record, colMap, "income"))
	invoice.DiscountTotal = parseNullableDecimal(getField(record, colMap, "discount total"))
	invoice.PricebookPrice = parseNullableDecimal(getField(record, colMap, "pricebook price"))

	// Labor costs (inaccurate per your note, but store anyway)
	invoice.LaborPay = parseNullableDecimal(getField(record, colMap, "labor pay"))
	invoice.LaborBurden = parseNullableDecimal(getField(record, colMap, "labor burden"))
	invoice.TotalLaborCosts = parseNullableDecimal(getField(record, colMap, "total labor costs"))

	// Customer/Location info (for validation)
	invoice.CustomerID = parseNullableInt64(getField(record, colMap, "customer id"))
	invoice.LocationID = parseNullableInt64(getField(record, colMap, "location id"))

	// Critical: Is this an adjustment invoice?
	invoice.IsAdjustment = parseBool(getField(record, colMap, "is adjustment"))

	// Other flags
	invoice.DispatchServiceFeeOnly = parseBool(getField(record, colMap, "dispatch/service fee only"))
	invoice.PrevailingWage = parseBool(getField(record, colMap, "prevailing wage"))

	// Job Type from invoice report (for validation)
	invoice.JobType = parseNullableString(getField(record, colMap, "job type"))

	return invoice, nil
}

// getField safely retrieves a field from a CSV row by column name
func getField(record []string, colMap map[string]int, columnName string) string {
	idx, ok := colMap[strings.ToLower(columnName)]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}
