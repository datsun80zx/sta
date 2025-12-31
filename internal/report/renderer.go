package report

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"math"
	"strings"
)

//go:embed templates/*.html
var templateFS embed.FS

// Renderer handles report template rendering
type Renderer struct {
	templates *template.Template
}

// NewRenderer creates a new template renderer
func NewRenderer() (*Renderer, error) {
	funcMap := template.FuncMap{
		"formatMoney":   formatMoney,
		"formatPercent": formatPercent,
		"formatDate":    formatDate,
		"truncate":      truncate,
		"abs":           math.Abs,
		"isNegative":    func(f float64) bool { return f < 0 },
		"add":           func(a, b int) int { return a + b },
		"mul":           func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"float64": func(i int) float64 { return float64(i) },
		"lt":      lessThan,
		"gt":      greaterThan,
		"eq":      equals,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	return &Renderer{templates: tmpl}, nil
}

// RenderSummary renders the summary report to HTML
func (r *Renderer) RenderSummary(w io.Writer, report *SummaryReport) error {
	return r.templates.ExecuteTemplate(w, "summary.html", report)
}

// RenderTechnicianReport renders the technician report to HTML
func (r *Renderer) RenderTechnicianReport(w io.Writer, report *TechnicianReport) error {
	return r.templates.ExecuteTemplate(w, "technicians.html", report)
}

// lessThan compares two values, handling both int and float64
func lessThan(a, b interface{}) bool {
	af := toFloat64(a)
	bf := toFloat64(b)
	return af < bf
}

// greaterThan compares two values, handling both int and float64
func greaterThan(a, b interface{}) bool {
	af := toFloat64(a)
	bf := toFloat64(b)
	return af > bf
}

// equals compares two values for equality
func equals(a, b interface{}) bool {
	af := toFloat64(a)
	bf := toFloat64(b)
	return af == bf
}

// toFloat64 converts int or float64 to float64
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}

// formatMoney formats a float as currency
func formatMoney(amount float64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}

	// Format with commas
	intPart := int64(amount)
	decPart := int64(math.Round((amount - float64(intPart)) * 100))

	var result string
	if intPart == 0 {
		result = "0"
	} else {
		var parts []string
		for intPart > 0 {
			parts = append([]string{fmt.Sprintf("%03d", intPart%1000)}, parts...)
			intPart /= 1000
		}
		result = strings.TrimLeft(strings.Join(parts, ","), "0,")
	}

	formatted := fmt.Sprintf("$%s.%02d", result, decPart)

	if negative {
		return "(" + formatted + ")"
	}
	return formatted
}

// formatPercent formats a float as percentage
func formatPercent(pct *float64) string {
	if pct == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%%", *pct)
}

// formatDate formats a time pointer as YYYY-MM-DD
func formatDate(t interface{}) string {
	switch v := t.(type) {
	case *interface{}:
		if v == nil {
			return "N/A"
		}
		return formatDate(*v)
	default:
		return "N/A"
	}
}

// truncate shortens a string with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
