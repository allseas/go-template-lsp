package model

type Order struct {
	ID           string
	CustomerName string
	Address      Address
}

type Address struct {
	Street string
	City   string
}
