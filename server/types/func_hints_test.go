package types

import (
	"go/ast"
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
