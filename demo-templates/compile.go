package main

import (
	"os"
	"text/template"

	"demo/funcs"
	"demo/models"
)

func main() {
	funcMap := funcs.TemplateFuncs()

	tmpl, err := template.New("demo.txt.tmpl").Funcs(funcMap).ParseFiles("demo.txt.tmpl")
	if err != nil {
		panic(err)
	}

	order := models.Order{
		ID:           "ORD-001",
		CustomerName: "Jane Doe",
		Email:        "jane@example.com",
		Address: models.Address{
			Street:  "Herengracht 182",
			City:    "Amsterdam",
			Zip:     "1016 BR",
			Country: "NL",
		},
		Items: []models.Item{
			{SKU: "A1", Name: "Widget", Qty: 3, UnitPrice: 29.99},
			{SKU: "B2", Name: "Gadget Pro", Qty: 1, UnitPrice: 149.00},
		},
		TotalAmount: 239.97,
		Paid:        true,
	}

	if err := tmpl.Execute(os.Stdout, order); err != nil {
		panic(err)
	}
}
