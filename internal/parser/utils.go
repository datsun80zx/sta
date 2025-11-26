package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func parseRequiredString(s string, rowNum int, columnName string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", &ValidationError{
			Row:    rowNum,
			Column: columnName,
			Value:  s,
			Err:    fmt.Errorf("required field is empty"),
		}
	}
	return s, nil
}

// parseInt64 parses a string to int64, returning ValidationError on failure
func parseInt64(s string, rowNum int, columnName string) (int64, error) {
	if s == "" {
		return 0, &ValidationError{
			Row:    rowNum,
			Column: columnName,
			Value:  s,
			Err:    fmt.Errorf("required field is empty"),
		}
	}

	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		s = parts[0]
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, &ValidationError{
			Row:    rowNum,
			Column: columnName,
			Value:  s,
			Err:    err,
		}
	}

	return val, nil
}

// parseNullableInt64 parses optional int64 fields
func parseNullableInt64(s string) *int64 {
	if s == "" {
		return nil
	}

	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}

	return &val
}

// parseDecimal parses required currency/decimal fields
func parseDecimal(s string, rowNum int, columnName string) (*decimal.Decimal, error) {
	if s == "" {
		return nil, &ValidationError{
			Row:    rowNum,
			Column: columnName,
			Value:  s,
			Err:    fmt.Errorf("required field is empty"),
		}
	}

	s = cleanCurrency(s)

	val, err := decimal.NewFromString(s)
	if err != nil {
		return nil, &ValidationError{
			Row:    rowNum,
			Column: columnName,
			Value:  s,
			Err:    err,
		}
	}

	return &val, nil
}

// parseNullableDecimal parses optional decimal fields
func parseNullableDecimal(s string) *decimal.Decimal {
	if s == "" {
		return nil
	}

	s = cleanCurrency(s)

	val, err := decimal.NewFromString(s)
	if err != nil {
		return nil
	}

	return &val
}

// cleanCurrency removes $ and commas from currency strings
// Also handles accounting notation: (123.45) → -123.45
func cleanCurrency(s string) string {
	s = strings.TrimSpace(s)

	// Handle accounting notation for negative numbers: (5517.95) means -5517.95
	isNegative := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		isNegative = true
		s = strings.TrimPrefix(s, "(")
		s = strings.TrimSuffix(s, ")")
		s = strings.TrimSpace(s)
	}

	// Remove currency symbols and formatting
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	// Add negative sign if needed
	if isNegative && s != "" && s != "0" && s != "0.00" {
		s = "-" + s
	}

	return s
}

// parseNullableString returns nil for empty strings
func parseNullableString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// parseNullableDate handles multiple date formats
func parseNullableDate(s string) *time.Time {
	if s == "" {
		return nil
	}

	s = strings.TrimSpace(s)

	// ServiceTitan exports dates as M/D/YYYY format
	formats := []string{
		"1/2/2006",   // M/D/YYYY (most common from your data)
		"01/02/2006", // MM/DD/YYYY
		"2006-01-02", // ISO 8601
		"1-2-2006",   // M-D-YYYY
		"01-02-2006", // MM-DD-YYYY
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}

	return nil
}

// parseBool handles TRUE/FALSE from ServiceTitan
func parseBool(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	return s == "TRUE" || s == "YES" || s == "1"
}
