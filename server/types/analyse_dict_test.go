package types

import (
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// analyseWithDict parses text and analyses the first (root) tree with the
// given dict as dot type. Returns the analysed tree.
func analyseWithDict(t *testing.T, text string, dict *DictType) *Tree {
	t.Helper()
	treeSet := parseTreeSet(t, text)
	require.NotEmpty(t, treeSet)
	// The root tree is "t" per parseTreeSet.
	rt := treeSet["t"]
	require.NotNil(t, rt)
	tree := NewTree(*rt, GlobalFuncs(), dict, nil, nil)
	return &tree
}

// dictOfLoaded builds a *DictType from a small helper: load each ref via
// LoadTypeFromHint and stitch them together. Only used in tests.
func dictOfLoaded(t *testing.T, refs map[string]string) *DictType {
	t.Helper()
	hint := TypeHint{Type: typeHintDict, Dict: refs}
	tr, err := LoadDictFromHint(hint, "../../test/resources/typehints-tests")
	require.NoError(t, err)
	require.NotNil(t, tr.DictType)
	return tr.DictType
}

func typeErrorMessages(tree *Tree) []string {
	msgs := make([]string, 0, len(tree.TypeErrors))
	for _, e := range tree.TypeErrors {
		msgs = append(msgs, e.Err)
	}
	return msgs
}

func TestAnalyseDict_FieldChainThroughDict(t *testing.T) {
	dict := dictOfLoaded(t, map[string]string{
		"Order":   "text-template-server/src/model.Order",
		"Address": "text-template-server/src/model.Address",
	})
	tree := analyseWithDict(t, `{{ .Order.CustomerName }}`, dict)
	if len(tree.TypeErrors) != 0 {
		t.Fatalf("expected no errors, got: %v", typeErrorMessages(tree))
	}
}

func TestAnalyseDict_MissingKeyIsFlagged(t *testing.T) {
	dict := dictOfLoaded(t, map[string]string{
		"Order": "text-template-server/src/model.Order",
	})
	tree := analyseWithDict(t, `{{ .Unknown }}`, dict)
	require.Len(t, tree.TypeErrors, 1)
	msg := tree.TypeErrors[0].Err
	if !strings.Contains(msg, `"Unknown"`) || !strings.Contains(msg, "known keys") {
		t.Fatalf("unexpected error message: %s", msg)
	}
	if !strings.Contains(msg, "Order") {
		t.Fatalf("expected error to list known keys, got: %s", msg)
	}
}

func TestAnalyseDict_VariablePreservesDictShape(t *testing.T) {
	dict := dictOfLoaded(t, map[string]string{
		"Order": "text-template-server/src/model.Order",
	})
	// Binding . to $d then walking $d.Order.CustomerName must still resolve.
	tree := analyseWithDict(
		t,
		`{{ $d := . }}{{ $d.Order.CustomerName }}`,
		dict,
	)
	if len(tree.TypeErrors) != 0 {
		t.Fatalf("expected no errors, got: %v", typeErrorMessages(tree))
	}
}

func TestAnalyseDict_WithPreservesDictShape(t *testing.T) {
	dict := dictOfLoaded(t, map[string]string{
		"Order": "text-template-server/src/model.Order",
	})
	tree := analyseWithDict(
		t,
		`{{ with . }}{{ .Order.CustomerName }}{{ end }}`,
		dict,
	)
	if len(tree.TypeErrors) != 0 {
		t.Fatalf("expected no errors, got: %v", typeErrorMessages(tree))
	}
}

// analyseWithDictAndTemplateInputs parses text and analyses the root tree with
// the given dict as dot type and a templateInputTypes map for {{template}} arg
// checking.
func analyseWithDictAndTemplateInputs(
	t *testing.T,
	text string,
	dict *DictType,
	inputs map[string]types.Type,
) *Tree {
	t.Helper()
	treeSet := parseTreeSet(t, text)
	require.NotEmpty(t, treeSet)
	rt := treeSet["t"]
	require.NotNil(t, rt)
	tree := NewTree(*rt, GlobalFuncs(), dict, nil, inputs)
	return &tree
}

func TestTypesCompatible_DictProjection(t *testing.T) {
	dict := dictOfLoaded(t, map[string]string{
		"Order": "text-template-server/src/model.Order",
	})
	mapStringAny := mapStringAnyType()
	mapStringString := types.NewMap(types.Typ[types.String], types.Typ[types.String])

	assert.True(t, typesCompatible(mapStringAny, dict),
		"dict should be assignable to map[string]any parameter")
	assert.True(t, typesCompatible(dict, mapStringAny),
		"map[string]any should be assignable to a dict parameter (dict projects to map[string]any)")
	assert.True(t, typesCompatible(dict, dict),
		"identical dicts must be compatible")
	assert.False(t, typesCompatible(mapStringString, dict),
		"dict projects to map[string]any, which is not assignable to map[string]string")
}

func TestAnalyseDict_TemplateArg_DictAcceptedByMapStringAny(t *testing.T) {
	dict := dictOfLoaded(t, map[string]string{
		"Order": "text-template-server/src/model.Order",
	})
	tree := analyseWithDictAndTemplateInputs(
		t,
		`{{ template "child" . }}`,
		dict,
		map[string]types.Type{"child": mapStringAnyType()},
	)
	if len(tree.TypeErrors) != 0 {
		t.Fatalf("expected no errors, got: %v", typeErrorMessages(tree))
	}
}

func TestAnalyseDict_TemplateArg_DictToDict_SameShapePasses(t *testing.T) {
	dict := dictOfLoaded(t, map[string]string{
		"Order": "text-template-server/src/model.Order",
	})
	tree := analyseWithDictAndTemplateInputs(
		t,
		`{{ template "child" . }}`,
		dict,
		map[string]types.Type{"child": dict},
	)
	if len(tree.TypeErrors) != 0 {
		t.Fatalf("expected no errors, got: %v", typeErrorMessages(tree))
	}
}

func TestAnalyseDict_TemplateArg_DictToDict_DifferentShapeFails(t *testing.T) {
	argDict := dictOfLoaded(t, map[string]string{
		"Order": "text-template-server/src/model.Order",
	})
	expectedDict := dictOfLoaded(t, map[string]string{
		"Address": "text-template-server/src/model.Address",
	})
	tree := analyseWithDictAndTemplateInputs(
		t,
		`{{ template "child" . }}`,
		argDict,
		map[string]types.Type{"child": expectedDict},
	)
	require.Len(t, tree.TypeErrors, 1)
	if tree.TypeErrors[0].typ != ErrorTypeMissingTemplateArgField {
		t.Fatalf("expected ErrorTypeMissingTemplateArgField (%d), got %d: %s",
			ErrorTypeMissingTemplateArgField, tree.TypeErrors[0].typ, tree.TypeErrors[0].Err)
	}
	if !strings.Contains(tree.TypeErrors[0].Err, "Address") ||
		!strings.Contains(tree.TypeErrors[0].Err, "Order") {
		t.Fatalf("expected message to mention both shapes, got: %s", tree.TypeErrors[0].Err)
	}
}
