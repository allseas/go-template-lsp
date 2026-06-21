package model

// Person is used as the input type for the "person-card" template block.
type Person struct {
	Name  string
	Age   int
	Email string
}

// Address is a distinct type used to demonstrate type mismatches.
type Address struct {
	Street string
	City   string
	Zip    string
}

// Company is another distinct type used in mismatch tests.
type Company struct {
	Name    string
	Country string
}

// Item is used as the input type for the "item" template block in the
// range/template-call fixtures. It contains a non basic type, as that is handled by ConvertibleTo already
type Item struct {
	Name string
	Tag  Person
}

// Order is a top-level dot type with a slice of Item, used to exercise the
// interaction between {{range}} (which rebinds dot to the element type) and
// {{template}} (which type-checks its argument against the target's gotype).
type Order struct {
	Address string
	Items   []Item
}
