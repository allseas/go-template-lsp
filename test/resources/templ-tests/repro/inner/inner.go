// Package inner is only reachable through package outer, never named directly
// in a gotype hint. Used to reproduce go-to-definition through a transitively
// loaded package.
package inner

// Inner is the type of outer.Thing.C.
type Inner struct {
	Deep string
}
