package types

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
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
		assert.Equal(t, 1, fn.Type().(*types.Signature).Params().Len())
		assert.Equal(t, "string", fn.Type().(*types.Signature).Params().At(0).Type().String())
	}
	if fn := funcs["sprintf"]; assert.NotNil(t, fn, "sprintf should resolve to fmt.Sprintf") {
		assert.Equal(t, "Sprintf", fn.Name())
		assert.Equal(t, 2, fn.Type().(*types.Signature).Params().Len())
		assert.Equal(t, "string", fn.Type().(*types.Signature).Params().At(0).Type().String())
		assert.Equal(
			t,
			"[]any",
			fn.Type().(*types.Signature).Params().At(1).Type().String(),
		)
	}
	// inline literal function - resolved to a synthetic *types.Func keyed by the FuncMap name.
	if fn := funcs["shout"]; assert.NotNil(
		t,
		fn,
		"shout should resolve to a synthetic *types.Func",
	) {
		assert.Equal(t, "shout", fn.Name())
		sig := fn.Type().(*types.Signature)
		assert.Equal(t, 1, sig.Params().Len())
		assert.Equal(t, "string", sig.Params().At(0).Type().String())
		assert.Equal(t, 1, sig.Results().Len())
		assert.Equal(t, "string", sig.Results().At(0).Type().String())
	}
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

	out := map[string]GlobalFuncEntry{}
	collectGlobalFuncs(file, nil, fset, out)

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

func TestComputeGlobalFuncs(t *testing.T) {
	funcs, err := ComputeGlobalFuncs("../../test/resources/funcmap-tests")
	require.NoError(t, err)

	assert.Contains(t, funcs, "upper")
	assert.Contains(t, funcs, "lower")
	assert.Contains(t, funcs, "repeat")
	assert.Contains(t, funcs, "shout")
	assert.Contains(t, funcs, "sprintf")
	assert.NotContains(t, funcs, "localOnly")
}

func TestComputeGlobalFuncsEmptyWorkspace(t *testing.T) {
	funcs, err := ComputeGlobalFuncs("")
	require.NoError(t, err)

	assert.Contains(t, funcs, "len")
	assert.Contains(t, funcs, "and")
	assert.Contains(t, funcs, "gt")
	assert.Contains(t, funcs, "eq")
	assert.Contains(t, funcs, "print")
}

func TestFindPackageRoots(t *testing.T) {
	roots, err := findPackageRoots("../../test/resources")
	require.NoError(t, err)
	require.NotEmpty(t, roots, "should find package roots in test/resources")

	var relRoots []string
	for _, root := range roots {
		rel, err := filepath.Rel("../../test/resources", root)
		require.NoError(t, err)
		relRoots = append(relRoots, rel)
	}
	sort.Strings(relRoots)

	expectedModules := []string{
		"definition-tests-client",
		"definition-tests-server",
		"funcmap-tests",
		"templ-tests",
		"template-arg-typechecking",
		"typehints-tests",
	}
	for _, expected := range expectedModules {
		assert.True(t, contains(relRoots, expected), "should find %s module root", expected)
	}
}

func TestFindPackageRoots_SkipsHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git", "subdir")
	require.NoError(t, os.MkdirAll(gitDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "go.mod"), []byte("module test"), 0o600))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module main"), 0o600))

	roots, err := findPackageRoots(tmpDir)
	require.NoError(t, err)

	// Should only find the root go.mod, not the one inside .git
	require.Equal(t, 1, len(roots))
	assert.Equal(t, tmpDir, roots[0])
}

func TestFindPackageRoots_SkipsVendor(t *testing.T) {
	tmpDir := t.TempDir()

	vendorDir := filepath.Join(tmpDir, "vendor", "some", "package")
	require.NoError(t, os.MkdirAll(vendorDir, 0o700))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(vendorDir, "go.mod"), []byte("module test"), 0o600),
	)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module main"), 0o600))

	roots, err := findPackageRoots(tmpDir)
	require.NoError(t, err)

	// Should only find the root go.mod, not the one inside vendor
	require.Equal(t, 1, len(roots))
	assert.Equal(t, tmpDir, roots[0])
}

func TestFindPackageRoots_MultipleNestedModules(t *testing.T) {
	tmpDir := t.TempDir()

	mod1Dir := filepath.Join(tmpDir, "module1")
	mod2Dir := filepath.Join(tmpDir, "module2", "nested")

	require.NoError(t, os.MkdirAll(mod1Dir, 0o700))
	require.NoError(t, os.MkdirAll(mod2Dir, 0o700))

	require.NoError(t, os.WriteFile(filepath.Join(mod1Dir, "go.mod"), []byte("module mod1"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(mod2Dir, "go.mod"), []byte("module mod2"), 0o600))

	roots, err := findPackageRoots(tmpDir)
	require.NoError(t, err)

	// Should find both module roots
	require.Equal(t, 2, len(roots))
	rootDirs := append([]string{}, roots...)
	sort.Strings(rootDirs)

	assert.Equal(t, mod1Dir, rootDirs[0])
	assert.Equal(t, mod2Dir, rootDirs[1])
}

func TestLoadGlobalFuncs_WithNestedModules(t *testing.T) {
	funcs, err := LoadGlobalFuncs("../../test/resources")
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
}

// Helper function to check if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestLoadGlobalFuncs_BuilderFactory verifies that FuncMap entries whose value
// is a call expression (e.g. `"box": BuildBox(4)`) resolve to a *types.Func
// whose signature is the *returned* function and whose position points to the
// factory function for go-to-definition.
func TestLoadGlobalFuncs_BuilderFactory(t *testing.T) {
	t.Cleanup(func() { SetGlobalFuncEntries(nil) })

	funcs, err := LoadGlobalFuncs("../../test/resources/funcmap-tests")
	require.NoError(t, err)

	fn := funcs["box"]
	require.NotNil(t, fn, "box (builder factory) should resolve to a *types.Func")

	// Name is taken from the FuncMap key, not the factory.
	assert.Equal(t, "box", fn.Name())

	// Signature must be the factory's *result* type: func(string) string.
	sig, ok := fn.Type().(*types.Signature)
	require.True(t, ok, "box should have a *types.Signature")
	require.Equal(t, 1, sig.Params().Len(), "box should take one parameter")
	assert.Equal(t, "string", sig.Params().At(0).Type().String())
	require.Equal(t, 1, sig.Results().Len(), "box should return one value")
	assert.Equal(t, "string", sig.Results().At(0).Type().String())

	// Package must be the factory's package, not nil.
	require.NotNil(t, fn.Pkg(), "box should carry the factory's package")
	assert.Equal(t, "funcs", fn.Pkg().Name())

	// Position must point at the factory (BuildBox) so go-to-definition lands
	// on real source rather than token.NoPos.
	entry, ok := GetGlobalFuncEntry("box")
	require.True(t, ok, "box entry should be cached")
	require.NotNil(t, entry.Fset)
	require.True(t, entry.Func.Pos().IsValid(), "factory position must be valid")
	pos := entry.Fset.Position(entry.Func.Pos())
	assert.Contains(
		t,
		filepath.ToSlash(pos.Filename),
		"funcmap-tests/funcs/funcs.go",
		"factory position should point at funcs.go",
	)
}

// TestLoadGlobalFuncs_GenericInstantiation verifies that FuncMap entries whose
// value is a generic instantiation (IndexExpr / IndexListExpr) resolve to the
// underlying generic *types.Func.
func TestLoadGlobalFuncs_GenericInstantiation(t *testing.T) {
	funcs, err := LoadGlobalFuncs("../../test/resources/funcmap-tests")
	require.NoError(t, err)

	// Single type parameter -> *ast.IndexExpr in the FuncMap value.
	if fn := funcs["sequenceI"]; assert.NotNil(
		t,
		fn,
		"sequenceI (generic, single type param) should resolve",
	) {
		assert.Equal(t, "Sequence", fn.Name(),
			"generic instantiation should resolve to the underlying generic function")
		require.NotNil(t, fn.Pkg())
		assert.Equal(t, "funcs", fn.Pkg().Name())
	}

	// Multiple type parameters -> *ast.IndexListExpr in the FuncMap value.
	if fn := funcs["pairSI"]; assert.NotNil(
		t,
		fn,
		"pairSI (generic, multiple type params) should resolve",
	) {
		assert.Equal(t, "Pair", fn.Name(),
			"generic instantiation should resolve to the underlying generic function")
		require.NotNil(t, fn.Pkg())
		assert.Equal(t, "funcs", fn.Pkg().Name())
	}
}

// TestResolveCalleeFunc_GenericInstantiation exercises resolveCalleeFunc
// directly on parsed-and-typechecked source so the IndexExpr / IndexListExpr
// branches are covered without depending on the workspace fixture.
func TestResolveCalleeFunc_GenericInstantiation(t *testing.T) {
	src := `package x

import "text/template"

func Sequence[T any](xs ...T) []T { return xs }
func Pair[A, B any](a A, b B) [2]any { return [2]any{a, b} }

//tmpl:func "global"
func G() template.FuncMap {
	return template.FuncMap{
		"sequenceI": Sequence[int],
		"pairSI":    Pair[string, int],
	}
}
`
	file, info, fset := typecheckSingleFile(t, src)

	out := map[string]GlobalFuncEntry{}
	collectGlobalFuncs(file, info, fset, out)

	if e, ok := out["sequenceI"]; assert.True(t, ok, "sequenceI must be collected") {
		require.NotNil(t, e.Func, "IndexExpr must resolve to a *types.Func")
		assert.Equal(t, "Sequence", e.Func.Name())
	}
	if e, ok := out["pairSI"]; assert.True(t, ok, "pairSI must be collected") {
		require.NotNil(t, e.Func, "IndexListExpr must resolve to a *types.Func")
		assert.Equal(t, "Pair", e.Func.Name())
	}
}

// TestResolveFuncObj_CallExprFallback covers the case where a CallExpr's
// result type is not a signature (e.g. the factory returns something else).
// In that fallback path resolveFuncObj must return the callee directly so the
// entry still has a valid definition position.
func TestResolveFuncObj_CallExprFallback(t *testing.T) {
	src := `package x

import "text/template"

// MakeName returns a plain string, not a function. The FuncMap entry below is
// semantically broken, but resolveFuncObj must not panic and should fall back
// to the callee so go-to-definition still works.
func MakeName() string { return "nope" }

//tmpl:func "global"
func G() template.FuncMap {
	return template.FuncMap{
		"broken": MakeName(),
	}
}
`
	file, info, fset := typecheckSingleFile(t, src)

	out := map[string]GlobalFuncEntry{}
	collectGlobalFuncs(file, info, fset, out)

	e, ok := out["broken"]
	require.True(t, ok, "broken must still be collected")
	require.NotNil(t, e.Func, "fallback must return the callee *types.Func")
	assert.Equal(t, "MakeName", e.Func.Name())
}

// typecheckSingleFile parses and type-checks src as a single-file package and
// returns the pieces needed to drive the FuncMap collectors.
func typecheckSingleFile(
	t *testing.T,
	src string,
) (*ast.File, *types.Info, *token.FileSet) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	require.NoError(t, err)

	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Implicits:  map[ast.Node]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
	}
	conf := types.Config{Importer: importer.Default()}
	_, err = conf.Check("x", fset, []*ast.File{file}, info)
	require.NoError(t, err)

	return file, info, fset
}
