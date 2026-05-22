package model

import "fmt"

// Address is used as a nested struct inside Order.
type Address struct {
	Street  string
	City    string
	Country string
	Zip     string
}

// Item is one line in an order.
type Item struct {
	SKU       string
	Name      string
	Qty       int
	UnitPrice float64
}

// Order is the top-level model.
// Import path: text-template-server/src/model
// gotype hint:  {{/*gotype: text-template-server/src/model.Order*/}}
type Order struct {
	ID           string
	CustomerName string
	Email        string
	Address      Address
	Items        []Item
	TotalAmount  float64
	Paid         bool
}

// DisplayName returns a human-readable label — 1 return value, always callable.
func (o Order) DisplayName() string {
	return o.CustomerName + " (" + o.ID + ")"
}

// Summary returns a short description or an error — (string, error) contract.
func (o Order) Summary() (string, error) {
	if o.ID == "" {
		return "", fmt.Errorf("order has no ID")
	}
	return fmt.Sprintf("Order %s: %.2f", o.ID, o.TotalAmount), nil
}

// ItemCount returns the number of line items — int return.
func (o Order) ItemCount() int {
	return len(o.Items)
}

// IsLargeOrder reports whether the total exceeds a threshold — bool return.
func (o Order) IsLargeOrder() bool {
	return o.TotalAmount > 1000
}

// Format formats the total with a given currency symbol — takes an arg, filtered by TakesArgs.
func (o Order) Format(currency string) string {
	return currency + fmt.Sprintf("%.2f", o.TotalAmount)
}

// badReturn has three return values — filtered out by the template contract check.
func (o Order) badReturn() (string, int, error) {
	return "", 0, nil
}

// wrongSecond has a non-error second return — also filtered out.
func (o Order) wrongSecond() (string, int) {
	return "", 0
}
