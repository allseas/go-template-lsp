package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGlobalFuncs(t *testing.T) {
	funcs, err := LoadGlobalFuncs("../../test/resources/funcmap-tests")
	require.NoError(t, err)

	names := make([]string, 0, len(funcs))
	for k := range funcs {
		names = append(names, k)
	}

	assert.Contains(t, names, "upper")
	assert.Contains(t, names, "lower")
	assert.Contains(t, names, "repeat")
	assert.Contains(t, names, "shout")
	assert.Contains(t, names, "sprintf")
	assert.NotContains(t, names, "localOnly")

	if fn := funcs["upper"]; assert.NotNil(t, fn, "upper should resolve to a *types.Func") {
		assert.Equal(t, "Upper", fn.Name())
	}
	if fn := funcs["sprintf"]; assert.NotNil(t, fn, "sprintf should resolve to fmt.Sprintf") {
		assert.Equal(t, "Sprintf", fn.Name())
	}
	// inline literal function — no identifier to resolve, stored as nil.
	_, present := funcs["shout"]
	assert.True(t, present)
}

func TestGlobalFuncsCacheRoundTrip(t *testing.T) {
	SetGlobalFuncs(nil)
	t.Cleanup(func() { SetGlobalFuncs(nil) })

	assert.Nil(t, GlobalFuncs())

	SetGlobalFuncs(map[string]*types.Func{"foo": nil})
	got := GlobalFuncs()
	require.NotNil(t, got)
	_, ok := got["foo"]
	assert.True(t, ok)

	// Returned map must be a snapshot, not the live cache.
	got["bar"] = nil
	assert.NotContains(t, GlobalFuncs(), "bar")
}

func TestCollectGlobalFuncs_OnlyGlobalHint(t *testing.T) {
	src := `package x

import "text/template"

//tmpl:func "global"
func G() template.FuncMap {
	return template.FuncMap{
		"a": nil,
		"b": nil,
	}
}

//tmpl:func "other"
func O() template.FuncMap {
	return template.FuncMap{
		"skipme": nil,
	}
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	require.NoError(t, err)

	out := map[string]*types.Func{}
	collectGlobalFuncs(file, nil, out)

	assert.Contains(t, out, "a")
	assert.Contains(t, out, "b")
	assert.NotContains(t, out, "skipme")
}

func TestIsFuncMapType(t *testing.T) {
	cases := []struct {
		name string
		expr ast.Expr
		want bool
	}{
		{"plain ident", &ast.Ident{Name: "FuncMap"}, true},
		{
			"selector ident",
			&ast.SelectorExpr{X: &ast.Ident{Name: "template"}, Sel: &ast.Ident{Name: "FuncMap"}},
			true,
		},
		{"different ident", &ast.Ident{Name: "MyMap"}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isFuncMapType(tc.expr))
		})
	}
}
