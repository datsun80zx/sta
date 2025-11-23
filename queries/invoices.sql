-- name: CreateInvoice :one
INSERT INTO invoices (
    id, job_id, import_batch_id,
    invoice_date, invoice_status, invoice_type, invoice_summary,
    total, balance, payments,
    material_costs, equipment_costs, purchase_order_costs, return_costs, costs_total,
    material_retail, material_markup, equipment_retail, equipment_markup,
    labor, labor_pay, labor_burden, total_labor_costs,
    income, discount_total, is_adjustment
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
)
RETURNING *;

-- name: GetInvoicesForJob :many
SELECT * FROM invoices
WHERE job_id = $1
ORDER BY invoice_date DESC;