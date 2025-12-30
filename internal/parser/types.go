package parser

import (
	"time"

	"github.com/shopspring/decimal"
)

// JobRow represents a parsed row from the Jobs report
type JobRow struct {
	// Core identifiers
	JobID      string
	CustomerID int64
	LocationID *int64
	InvoiceID  *string

	// Customer info
	CustomerName  *string
	CustomerType  *string
	CustomerCity  *string
	CustomerState *string
	CustomerZip   *string

	// Location info (service address)
	LocationCity  *string
	LocationState *string
	LocationZip   *string

	// Job details
	JobType        string
	Status         string
	BusinessUnit   *string
	BusinessUnitID *int64

	// Dates
	JobCreationDate   *time.Time
	JobScheduleDate   *time.Time
	JobCompletionDate *time.Time

	// People
	AssignedTechnicians *string
	SoldBy              *string
	BookedBy            *string
	DispatchedBy        *string
	PrimaryTechnician   *string

	// Campaign/Marketing
	JobCampaignID    *int64
	CallCampaignID   *int64
	CampaignCategory *string

	// Revenue
	JobsSubtotal          *decimal.Decimal
	JobTotal              *decimal.Decimal
	EstimateSalesSubtotal *decimal.Decimal // What was sold via estimates

	// Other
	Summary          *string
	Priority         *string
	TotalHoursWorked *decimal.Decimal
	SurveyResult     *decimal.Decimal
	MemberStatus     *string
	Tags             *string
	EstimateCount    *int64

	// Boolean flags
	Opportunity   bool
	Warranty      bool
	Recall        bool
	Converted     bool
	ZeroDollarJob bool
}

// InvoiceRow represents a parsed row from the Invoices report
type InvoiceRow struct {
	// Core identifiers
	InvoiceID             string
	JobID                 string
	CustomerID            *int64
	LocationID            *int64
	ProjectNumber         *int64
	InvoiceBusinessUnitID *int64

	// Invoice details
	InvoiceDate    time.Time
	InvoiceStatus  *string
	InvoiceType    *string
	InvoiceSummary *string

	// Totals
	Total    *decimal.Decimal
	Balance  *decimal.Decimal
	Payments *decimal.Decimal

	// Payment info
	PaymentTypes *string
	PaymentTerm  *string

	// Costs (critical for profitability)
	MaterialCosts      *decimal.Decimal
	EquipmentCosts     *decimal.Decimal
	PurchaseOrderCosts *decimal.Decimal
	ReturnCosts        *decimal.Decimal
	CostsTotal         *decimal.Decimal

	// Retail/Markup
	MaterialRetail  *decimal.Decimal
	MaterialMarkup  *decimal.Decimal
	EquipmentRetail *decimal.Decimal
	EquipmentMarkup *decimal.Decimal
	Labor           *decimal.Decimal
	Income          *decimal.Decimal
	DiscountTotal   *decimal.Decimal
	PricebookPrice  *decimal.Decimal

	// Labor costs (inaccurate, but stored)
	LaborPay        *decimal.Decimal
	LaborBurden     *decimal.Decimal
	TotalLaborCosts *decimal.Decimal

	// Flags
	IsAdjustment           bool
	DispatchServiceFeeOnly bool
	PrevailingWage         bool

	// Job type (for validation)
	JobType *string
}
