// Package funcs provides a sample FuncMap exposed to templates as globals.
package funcs

import (
	"fmt"
	"strings"
	"text/template"

	"cg/model"
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

//tmpl:func "global"
func GlobalFuncs() template.FuncMap {
	return template.FuncMap{
		"siemensLayout": func() *model.Layout { return &model.Layout{} },
		"kebabCase":     func(s string) string { return strings.ReplaceAll(s, " ", "-") },
		"upper":         Upper,
		"dict":          func() map[string]string { return map[string]string{"a": "A", "b": "B"} },
		"lower":         Lower,
		"repeat":        Repeat,
		"shout":         func(s string) string { return s + "!" },
		"sprintf":       fmt.Sprintf,
	}
}

//tmpl:func "local"
func LocalFuncs() template.FuncMap {
	return template.FuncMap{
		"localOnly": localOnly,
	}
}
