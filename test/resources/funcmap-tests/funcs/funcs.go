// Package funcs provides a sample FuncMap exposed to templates as globals.
package funcs

import (
	"fmt"
	"strings"
	"text/template"
)

// Upper returns s upper-cased.
func Upper(s string) string {
	return strings.ToUpper(s)
}

// Lower returns s lower-cased.
func Lower(s string) string {
	return strings.ToLower(s)
}

// Repeat returns s concatenated n times.
func Repeat(s string, n int) string {
	return strings.Repeat(s, n)
}

// localOnly is referenced from a non-global FuncMap and must not be exposed.
func localOnly() string { return "nope" }

// BuildBox returns a function that wraps s in a box of the given width.
// It is exposed to templates via a factory call in the FuncMap.
func BuildBox(width int) func(s string) string {
	return func(s string) string {
		return strings.Repeat("*", width) + s + strings.Repeat("*", width)
	}
}

// Sequence is a generic function that returns its arguments as a slice.
// Used to test that generic instantiations are resolvable in FuncMap entries.
func Sequence[T any](xs ...T) []T { return xs }

// Pair is a generic function with two type parameters, used to test
// IndexListExpr resolution in FuncMap entries.
func Pair[A, B any](a A, b B) [2]any { return [2]any{a, b} }

//tmpl:func "global"
func GlobalFuncs() template.FuncMap {
	return template.FuncMap{
		"wc":        func(s string) int { return len(s) },
		"upper":     Upper,
		"lower":     Lower,
		"repeat":    Repeat,
		"shout":     func(s string) string { return s + "!" },
		"sprintf":   fmt.Sprintf,
		"box":       BuildBox(4),
		"sequenceI": Sequence[int],
		"pairSI":    Pair[string, int],
	}
}

//tmpl:func "local"
func LocalFuncs() template.FuncMap {
	return template.FuncMap{
		"localOnly": localOnly,
	}
}
