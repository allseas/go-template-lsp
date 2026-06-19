package funcs

import (
	"fmt"
	"strings"
	"text/template"
)

// GtmplShout upper-cases s and appends an exclamation mark. It is referenced
// by name from the custom global FuncMap so go-to-definition can resolve it.
func GtmplShout(s string) string {
	return strings.ToUpper(s) + "!"
}

// GtmplRepeatN returns s repeated n times, space-separated. It demonstrates a
// multi-argument global function registered by name.
func GtmplRepeatN(s string, n int) string {
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

//tmpl:func "global"
func CustomGlobalFuncs() template.FuncMap {
	return template.FuncMap{
		"gtmplShout":   GtmplShout,
		"gtmplRepeatN": GtmplRepeatN,
		"gtmplGreet":   func(name string) string { return fmt.Sprintf("Hello, %s!", name) },
	}
}
