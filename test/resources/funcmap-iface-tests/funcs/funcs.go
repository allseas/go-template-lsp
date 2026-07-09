// Package funcs exposes a workspace funcmap that includes a function taking
// a named interface parameter. It is used to test that interface satisfaction
// works for arguments whose concrete type is loaded via a gotype hint.
package funcs

import (
	"text/template"

	"text-template-server/funcmap-iface-tests/iface"
)

// describe returns a description for any value implementing iface.HasType.
// The parameter type is a named interface so it exercises the code path that
// unwraps *types.Named before performing an interface-satisfaction check.
func describe(h iface.HasType) string {
	if h == nil {
		return ""
	}
	return h.Kind()
}

//tmpl:func "global"
func GlobalFuncs() template.FuncMap {
	return template.FuncMap{
		"describe": describe,
	}
}
