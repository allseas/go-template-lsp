package models

import "fmt"

// Address holds a shipping or billing address.
type Address struct {
	Street  string
	City    string
	Country string
	Zip     string
}

// Line returns a single-line formatted address.
func (a Address) Line() string {
	return fmt.Sprintf("%s, %s %s", a.Street, a.City, a.Zip)
}

// IsLocal reports whether the address is domestic (NL).
func (a Address) IsLocal() bool {
	return a.Country == "NL"
}

// Item is a single line on an order.
type Item struct {
	SKU       string
	Name      string
	Qty       int
	UnitPrice float64
}

// Total returns the line total (Qty × UnitPrice).
func (i Item) Total() float64 {
	return float64(i.Qty) * i.UnitPrice
}

// Label returns a short display label.
func (i Item) Label() string {
	return fmt.Sprintf("%s x%d", i.Name, i.Qty)
}

// IsExpensive reports whether the unit price exceeds 100.
func (i Item) IsExpensive() bool {
	return i.UnitPrice > 100
}

// Order is the top-level order model passed to templates.
// gotype hint: {{/*gotype: demo/models.Order*/}}
type Order struct {
	ID           string
	CustomerName string
	Email        string
	Address      Address
	Items        []Item
	TotalAmount  float64
	Paid         bool
}

func (o Order) Format(fmt string) string {
	return ""
} 

// DisplayName returns a human-readable label for the order.
func (o Order) DisplayName() string {
	return fmt.Sprintf("%s (%s)", o.CustomerName, o.ID)
}

// ItemCount returns the number of line items.
func (o Order) ItemCount() int {
	return len(o.Items)
}

// IsLargeOrder reports whether the total exceeds 1 000.
func (o Order) IsLargeOrder() bool {
	return o.TotalAmount > 1000
}

// ReportData is passed to the monthly report template.
// gotype hint: {{/*gotype: demo/models.ReportData*/}}
type ReportData struct {
	Month    string
	Orders   []Order
	Revenue  float64
	Currency string
}

// OrderCount returns the number of orders in the report.
func (r ReportData) OrderCount() int {
	return len(r.Orders)
}
