package types

import (
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDictFromHint(t *testing.T) {
	hint := TypeHint{
		Type: typeHintDict,
		Dict: map[string]string{
			"Order":   "text-template-server/src/model.Order",
			"Address": "text-template-server/src/model.Address",
		},
	}
	tree, err := LoadDictFromHint(hint, "../../test/resources/typehints-tests")
	require.NoError(t, err)
	require.NotNil(t, tree)
	require.NotNil(t, tree.DictType)
	assert.Len(t, tree.DictType.Fields, 2)

	orderT, ok := tree.DictType.LookupDictKey("Order")
	require.True(t, ok)
	assert.Contains(t, orderT.String(), "Order")

	_, ok = tree.DictType.LookupDictKey("Missing")
	assert.False(t, ok)

	assert.Equal(t, []string{"Address", "Order"}, tree.DictType.DictKeys())
}

func TestLoadDictFromHint_PropagatesEntryError(t *testing.T) {
	hint := TypeHint{
		Type: typeHintDict,
		Dict: map[string]string{
			"Bad": "nonexistent/pkg.Foo",
		},
	}
	_, err := LoadDictFromHint(hint, "../../test/resources/typehints-tests")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Bad")
}

func TestLoadDictFromHint_RejectsNonDict(t *testing.T) {
	_, err := LoadDictFromHint(TypeHint{Type: typeHintStruct, Text: "Foo"}, ".")
	require.Error(t, err)
}

func TestCachedLoadHint_DispatchesByKind(t *testing.T) {
	structHint := TypeHint{
		Type: typeHintStruct,
		Text: "text-template-server/src/model.Order",
	}
	dictHint := TypeHint{
		Type: typeHintDict,
		Dict: map[string]string{"Order": "text-template-server/src/model.Order"},
	}

	root := "../../test/resources/typehints-tests"

	s1, err := CachedLoadHint(structHint, root)
	require.NoError(t, err)
	require.NotNil(t, s1.DotType)
	assert.Nil(t, s1.DictType)

	d1, err := CachedLoadHint(dictHint, root)
	require.NoError(t, err)
	require.NotNil(t, d1.DictType)
}

func TestDictType_String_Deterministic(t *testing.T) {
	pkg := types.NewPackage("example.test/fields", "fields")
	stA := types.NewNamed(types.NewTypeName(0, pkg, "A", nil), types.NewStruct(nil, nil), nil)
	stB := types.NewNamed(types.NewTypeName(0, pkg, "B", nil), types.NewStruct(nil, nil), nil)

	d := &DictType{Fields: map[string]types.Type{"A": stA, "B": stB}}

	got1 := d.String()
	got2 := d.String()
	assert.Equal(t, got1, got2)
	assert.True(t, strings.Contains(got1, `"A"`))
	assert.True(t, strings.Contains(got1, `"B"`))
	// keys must appear in sorted order
	assert.Less(t, strings.Index(got1, `"A"`), strings.Index(got1, `"B"`))
}

func TestDictTypeFields(t *testing.T) {
	pkg := types.NewPackage("example.test/fields", "fields")
	stA := types.NewNamed(types.NewTypeName(0, pkg, "A", nil), types.NewStruct(nil, nil), nil)
	stB := types.NewNamed(types.NewTypeName(0, pkg, "B", nil), types.NewStruct(nil, nil), nil)

	d := &DictType{Fields: map[string]types.Type{"B": stB, "A": stA}}
	fields := DictTypeFields(d)

	require.Len(t, fields, 2)
	assert.Equal(t, "A", fields[0].Name)
	assert.Equal(t, "B", fields[1].Name)
	assert.Equal(t, stA, fields[0].Type)
	assert.Equal(t, stB, fields[1].Type)
}

func TestDictTypeFields_Nil(t *testing.T) {
	assert.Nil(t, DictTypeFields(nil))
}
