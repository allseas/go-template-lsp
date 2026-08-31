// Package othermodel provides a type used as a generic type argument from a
// different package than the generic base, to exercise cross-package
// go-to-definition through gotype hints.
package othermodel

// Gadget is used as a type argument to cg/model.View, so its members live in a
// different package than the generic base.
type Gadget struct {
	Serial string
	Weight int
}

// Describe returns a human-readable label for the gadget.
func (g Gadget) Describe() string {
	return g.Serial
}
