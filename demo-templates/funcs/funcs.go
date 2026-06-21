package funcs

// Package funcs provides template functions exposed via a FuncMap.
// The //tmpl:func "global" annotation tells the GoTemplate LSP to pick these
// up automatically - they will appear in completions and pass diagnostics.

import (
	"fmt"
	"strings"
	"text/template"
)

// Upper returns s upper-cased.
func Upper(s string) string { return strings.ToUpper(s) }

// Lower returns s lower-cased.
func Lower(s string) string { return strings.ToLower(s) }

// Repeat returns s repeated n times.
func Repeat(s string, n int) string { return strings.Repeat(s, n) }

// Indent prefixes every line of s with n spaces.
func Indent(n int, s string) string {
	pad := strings.Repeat(" ", n)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

// FormatCurrency formats an amount with a currency symbol.
func FormatCurrency(amount float64, symbol string) string {
	return fmt.Sprintf("%s %.2f", symbol, amount)
}

// YesNo converts a bool to "Yes" / "No".
func YesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func NumberInText(x int) string {
	// ...
	return "One"
}

//tmpl:func "global"
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"upper":          Upper,
		"lower":          Lower,
		"shout":          Upper,
		"indent":         Indent,
		"formatCurrency": FormatCurrency,
		"yesNo":          YesNo,
		"numberString":   NumberInText,
		"repeat":         Repeat,
		"kebabCase":      Upper,
	}
}
