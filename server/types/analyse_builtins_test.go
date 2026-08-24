package types

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const typeHintsRoot = "../../test/resources/typehints-tests"

// analyseWithDot parses text and analyses the root tree against the given dot
// type, package and function map.
func analyseWithDot(
	t *testing.T,
	text string,
	dot types.Type,
	pkg *types.Package,
	funcs map[string]*types.Func,
) *Tree {
	t.Helper()
	treeSet := parseTreeSet(t, text)
	rt := treeSet["t"]
	require.NotNil(t, rt)
	tree := NewTree(*rt, funcs, dot, pkg, nil)
	return &tree
}

// loadOrder loads the model.Order struct type used across these tests.
func loadOrder(t *testing.T) (types.Type, *types.Package) {
	t.Helper()
	lt, err := LoadTypeFromHint("text-template-server/src/model.Order", typeHintsRoot)
	require.NoError(t, err)
	return lt.DotType, lt.Pkg
}

// funcsWithDict returns the builtin funcs plus a registered `dict` function.
// `dict` is not a text/template builtin (it is a common funcmap helper), so it
// must be registered to avoid the undefined-function check; its result type is
// computed by analyseCommand's dict special case, not by this placeholder
// signature.
func funcsWithDict() map[string]*types.Func {
	funcs := BuiltinFuncs()
	anyT := types.NewInterfaceType(nil, nil).Complete()
	sig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "args", types.NewSlice(anyT))),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", anyT)), true)
	funcs["dict"] = types.NewFunc(token.NoPos, nil, "dict", sig)
	return funcs
}

func allErrs(tree *Tree) string {
	return strings.Join(typeErrorMessages(tree), "\n")
}

// --- index builtin -------------------------------------------------------

func TestAnalyseIndex_SliceElementFieldResolves(t *testing.T) {
	dot, pkg := loadOrder(t)
	// index .Items 0 -> Item; .Name is a valid Item field.
	tree := analyseWithDot(t, `{{ (index .Items 0).Name }}`, dot, pkg, BuiltinFuncs())
	require.Empty(t, tree.TypeErrors, allErrs(tree))
}

func TestAnalyseIndex_SliceElementBadFieldFlagged(t *testing.T) {
	dot, pkg := loadOrder(t)
	// Proves index typed the result as a concrete Item (not any): a bad field errors.
	tree := analyseWithDot(t, `{{ (index .Items 0).Nope }}`, dot, pkg, BuiltinFuncs())
	require.Len(t, tree.TypeErrors, 1, allErrs(tree))
}

func TestAnalyseIndex_NotIndexable(t *testing.T) {
	dot, pkg := loadOrder(t)
	tree := analyseWithDot(t, `{{ index .TotalAmount 0 }}`, dot, pkg, BuiltinFuncs())
	require.NotEmpty(t, tree.TypeErrors)
	require.Contains(t, allErrs(tree), "cannot index")
}

func TestAnalyseIndex_WrongIndexType(t *testing.T) {
	dot, pkg := loadOrder(t)
	// .Items is []Item, so a string index is invalid.
	tree := analyseWithDot(t, `{{ index .Items "x" }}`, dot, pkg, BuiltinFuncs())
	require.NotEmpty(t, tree.TypeErrors)
	require.Contains(t, allErrs(tree), "cannot use")
}

// --- dict function -------------------------------------------------------

func TestAnalyseDict_ValueFieldResolvesOnChain(t *testing.T) {
	dot, pkg := loadOrder(t)
	// dict "addr" .Address -> map{"addr": Address}; field access resolves
	// directly on the chain base (dict ...).addr.City.
	tree := analyseWithDot(t, `{{ (dict "addr" .Address).addr.City }}`, dot, pkg, funcsWithDict())
	require.Empty(t, tree.TypeErrors, allErrs(tree))
}

func TestAnalyseDict_ValueBadFieldFlaggedOnChain(t *testing.T) {
	dot, pkg := loadOrder(t)
	// Proves the dict value was typed as Address (not any): a bad field errors.
	tree := analyseWithDot(t, `{{ (dict "addr" .Address).addr.Nope }}`, dot, pkg, funcsWithDict())
	require.Len(t, tree.TypeErrors, 1, allErrs(tree))
}

func TestAnalyseDict_UnknownKeyFlaggedOnChain(t *testing.T) {
	dot, pkg := loadOrder(t)
	tree := analyseWithDot(t, `{{ (dict "addr" .Address).nope }}`, dot, pkg, funcsWithDict())
	require.Len(t, tree.TypeErrors, 1, allErrs(tree))
	require.Equal(t, ErrorType(ErrorTypeInvalidDictKey), tree.TypeErrors[0].typ)
}

func TestAnalyseDict_UnknownKeyPassesThroughAsAny(t *testing.T) {
	dot, pkg := loadOrder(t)
	// A dict only records its known keys; unknown keys may still be populated
	// downstream, so an unknown key resolves to any and further access on it
	// must not cascade a second error (just the single advisory info).
	tree := analyseWithDot(
		t,
		`{{ (dict "addr" .Address).nope.deeper.field }}`,
		dot,
		pkg,
		funcsWithDict(),
	)
	require.Len(t, tree.TypeErrors, 1, allErrs(tree))
	require.Equal(t, ErrorType(ErrorTypeInvalidDictKey), tree.TypeErrors[0].typ)
}

func TestAnalyseDict_ValueFieldResolvesOnVar(t *testing.T) {
	dot, pkg := loadOrder(t)
	// The same access also resolves through a variable binding.
	tree := analyseWithDot(
		t,
		`{{ $d := dict "addr" .Address }}{{ $d.addr.City }}`,
		dot,
		pkg,
		funcsWithDict(),
	)
	require.Empty(t, tree.TypeErrors, allErrs(tree))
}

func TestAnalyseDict_OddArgs(t *testing.T) {
	dot, pkg := loadOrder(t)
	tree := analyseWithDot(t, `{{ dict "a" }}`, dot, pkg, funcsWithDict())
	require.NotEmpty(t, tree.TypeErrors)
	require.Contains(t, allErrs(tree), "even number")
}

func TestAnalyseDict_NonStringKey(t *testing.T) {
	dot, pkg := loadOrder(t)
	// .ID is a field access, not a string literal, so it cannot be a dict key.
	tree := analyseWithDot(t, `{{ dict .ID .Address }}`, dot, pkg, funcsWithDict())
	require.NotEmpty(t, tree.TypeErrors)
	require.Contains(t, allErrs(tree), "string literal")
}

// --- map field-key access ------------------------------------------------

func TestAnalyseMapFieldKey_StringKeyResolves(t *testing.T) {
	dot, pkg := loadOrder(t)
	// .Meta is map[string]string, so .Meta.Region is a string-keyed access.
	tree := analyseWithDot(t, `{{ .Meta.Region }}`, dot, pkg, GlobalFuncs())
	require.Empty(t, tree.TypeErrors, allErrs(tree))
}

func TestAnalyseMapFieldKey_NonStringKeyFlagged(t *testing.T) {
	dot, pkg := loadOrder(t)
	// .Counts is map[int]string; a field-name (string) key is not assignable.
	tree := analyseWithDot(t, `{{ .Counts.Region }}`, dot, pkg, GlobalFuncs())
	require.Len(t, tree.TypeErrors, 1, allErrs(tree))
	require.Contains(t, tree.TypeErrors[0].Err, "string-compatible key")
}
