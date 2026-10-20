package types

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const nestFixtureRoot = "../../test/resources/typehints-tests"

const (
	pkgModel = "text-template-server/src/model"
	pkgOther = "text-template-server/src/othermodel"
)

// TestDictAsGenericArg: View[map{...}] instantiates and .Model is the dict.
func TestDictAsGenericArg(t *testing.T) {
	lt, err := LoadTypeFromHint(
		pkgModel+`.View[map{"a": `+pkgOther+`.Gadget, "b": string}]`, nestFixtureRoot,
	)
	require.NoError(t, err)
	require.NotNil(t, lt.DotType, "View[dict] is a named type, not a top-level dict")

	mt := fieldType(lt.DotType, "Model")
	require.NotNil(t, mt)
	dict, ok := mt.(*DictType)
	require.True(t, ok, "Model should resolve to the dict, got %T", mt)
	assert.Equal(t, []string{"a", "b"}, dict.DictKeys())
}

// TestDictNestedInComposite: *[]map{...} resolves to pointer->slice->dict.
func TestDictNestedInComposite(t *testing.T) {
	lt, err := LoadTypeFromHint(`*[]map{"a": `+pkgOther+`.Gadget}`, nestFixtureRoot)
	require.NoError(t, err)
	require.NotNil(t, lt.DotType)

	ptr, ok := lt.DotType.(*types.Pointer)
	require.True(t, ok, "want *T, got %T", lt.DotType)
	slice, ok := ptr.Elem().(*types.Slice)
	require.True(t, ok, "want slice, got %T", ptr.Elem())
	_, ok = slice.Elem().(*DictType)
	require.True(t, ok, "want dict element, got %T", slice.Elem())
}

// TestTopLevelDictViaResolver: a bare map{...} resolves to Tree.DictType.
func TestTopLevelDictViaResolver(t *testing.T) {
	lt, err := LoadTypeFromHint(`map{"a": string, "b": []`+pkgOther+`.Gadget}`, nestFixtureRoot)
	require.NoError(t, err)
	require.Nil(t, lt.DotType)
	require.NotNil(t, lt.DictType)
	assert.Equal(t, []string{"a", "b"}, lt.DictType.DictKeys())
}

// TestMultiParamGenericDictValue: comma inside brackets no longer breaks a dict
// value (go/parser splits it, not a naive comma split).
func TestMultiParamGenericDictValue(t *testing.T) {
	lt, err := LoadTypeFromHint(
		`map{"p": `+pkgModel+`.Pair[`+pkgOther+`.Gadget, string]}`, nestFixtureRoot,
	)
	require.NoError(t, err)
	require.NotNil(t, lt.DictType)
	pv, ok := lt.DictType.LookupDictKey("p")
	require.True(t, ok)
	named, ok := pv.(*types.Named)
	require.True(t, ok, "want Pair named type, got %T", pv)
	assert.Equal(t, "Pair", named.Obj().Name())
	require.Equal(t, 2, named.TypeArgs().Len())
}

// TestPointerToDictCollapses: text/template dereferences pointers, so a pointer
// to a dict resolves to the dict itself and keeps key lookups working.
func TestPointerToDictCollapses(t *testing.T) {
	lt, err := LoadTypeFromHint(`*map{"a": `+pkgOther+`.Gadget}`, nestFixtureRoot)
	require.NoError(t, err)
	require.NotNil(t, lt.DictType, "a root *map{...} should resolve to the dict")
	assert.Equal(t, []string{"a"}, lt.DictType.DictKeys())

	lt, err = LoadTypeFromHint(pkgModel+`.View[*map{"a": string}]`, nestFixtureRoot)
	require.NoError(t, err)
	_, ok := fieldType(lt.DotType, "Model").(*DictType)
	assert.True(t, ok, "View[*map{...}].Model should be the dict")
}

// TestPointerDictInsideComposite: go/parser folds `[]*` and `map[K]*` into the
// dict literal's type; both must parse and wrap the (collapsed) dict.
func TestPointerDictInsideComposite(t *testing.T) {
	lt, err := LoadTypeFromHint(`[]*map{"a": `+pkgOther+`.Gadget}`, nestFixtureRoot)
	require.NoError(t, err)
	slice, ok := lt.DotType.(*types.Slice)
	require.True(t, ok, "want slice, got %T", lt.DotType)
	_, ok = slice.Elem().(*DictType)
	assert.True(t, ok, "want dict element, got %T", slice.Elem())

	lt, err = LoadTypeFromHint(`map[string]*map{"a": string}`, nestFixtureRoot)
	require.NoError(t, err)
	m, ok := lt.DotType.(*types.Map)
	require.True(t, ok, "want map, got %T", lt.DotType)
	_, ok = m.Elem().(*DictType)
	assert.True(t, ok, "want dict value, got %T", m.Elem())
}

// TestHintsEqualCanonical: equivalent hints that differ only in dict key order
// or whitespace are equal; genuinely different hints are not.
func TestHintsEqualCanonical(t *testing.T) {
	cases := []struct {
		a, b  string
		equal bool
	}{
		{`map{"a": int, "b": string}`, `map{"b": string, "a": int}`, true},
		{`map{"a": int, "b": string}`, `map{"a":int,"b":string}`, true},
		{
			`x/m.View[map{"o": map{"y": int, "x": a/p.T}}]`,
			`x/m.View[map{"o": map{"x": a/p.T, "y": int}}]`,
			true,
		},
		{`[] *a/p.T`, `[]*a/p.T`, true},
		{`map{"a": int}`, `map{"a": string}`, false},
		{`map{"a": int}`, `map{"a": int, "b": int}`, false},
		{`a/pkg.T`, `b/pkg.T`, false},
	}
	for _, tc := range cases {
		ha, ok := parseHintText("gotype: " + tc.a)
		require.True(t, ok)
		hb, ok := parseHintText("gotype: " + tc.b)
		require.True(t, ok)
		assert.Equal(t, tc.equal, hintsEqual(ha, hb), "%s vs %s", tc.a, tc.b)
	}
}
