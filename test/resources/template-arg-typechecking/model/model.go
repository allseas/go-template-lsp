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
