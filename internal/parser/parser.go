package parser

import (
	"fmt"
	"io"
)

// Parser defines the interface for parsing ServiceTitan export files
type Parser interface {
	ParseJobs(r io.Reader) ([]JobRow, error)
	ParseInvoices(r io.Reader) ([]InvoiceRow, error)
}

// ParseResult contains the parsed data and any warnings
type ParseResult struct {
	Jobs     []JobRow
	Invoices []InvoiceRow
	Warnings []string
}

// ValidationError represents a parsing error with context
type ValidationError struct {
	Row    int
	Column string
	Value  string
	Err    error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("row %d, column %s: failed to parse '%s': %v",
		e.Row, e.Column, e.Value, e.Err)
}
