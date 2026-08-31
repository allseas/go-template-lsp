// Package othermodel provides types used as generic type arguments from a
// different package than the generic base, to exercise cross-package
// go-to-definition through gotype hints.
package othermodel

// Gadget is used as a type argument to a generic type declared in the model
// package, so its fields live in a different package/FileSet than the base.
type Gadget struct {
	Serial string
	Weight int
}

// Describe returns a human-readable label for the gadget.
func (g Gadget) Describe() string {
	return g.Serial
}
