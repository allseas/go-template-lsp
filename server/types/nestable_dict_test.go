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
