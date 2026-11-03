package types

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadHintType resolves a hint to its dot type (the dict for a top-level dict).
func loadHintType(t *testing.T, hint string) types.Type {
	t.Helper()
	lt, err := LoadTypeFromHint(hint, nestFixtureRoot)
	require.NoError(t, err)
	if lt.DictType != nil {
		return lt.DictType
	}
	require.NotNil(t, lt.DotType)
	return lt.DotType
}

// analyseNoPanic analyses text against dot and inputs, failing the test on panic.
func analyseNoPanic(t *testing.T, text string, dot types.Type, inputs map[string]types.Type) *Tree {
	t.Helper()
	rt := parseTreeSet(t, text)["t"]
	require.NotNil(t, rt)
	var tree Tree
	require.NotPanics(t, func() { tree = NewTree(*rt, GlobalFuncs(), dot, nil, inputs) })
	return &tree
}

func errorTypes(tree *Tree) []ErrorType {
	out := make([]ErrorType, 0, len(tree.TypeErrors))
	for _, e := range tree.TypeErrors {
		out = append(out, e.typ)
	}
	return out
}

// Template arguments whose types nest dicts used to panic inside go/types.
func TestDictSafe_TemplateArgNestedDicts(t *testing.T) {
	view := pkgModel + ".View"
	cases := []struct {
		name, arg, want string
		wantErr         bool
	}{
		{"slice value mismatch", `[]map{"a": int}`, `[]map{"a": string}`, true},
		{"slice missing key", `[]map{"a": int}`, `[]map{"a": int, "b": int}`, true},
		{"slice extra key", `[]map{"a": int, "b": int}`, `[]map{"a": int}`, false},
		{"slice same shape", `[]map{"a": int}`, `[]map{"a": int}`, false},
		{"slice to []map[string]any", `[]map{"a": int}`, `[]map[string]any`, false},
		{"map value mismatch", `map[string]map{"a": int}`, `map[string]map{"a": string}`, true},
		{"map same shape", `map[string]map{"a": int}`, `map[string]map{"a": int}`, false},
		{"generic arg mismatch", view + `[map{"a": int}]`, view + `[map{"a": string}]`, true},
		{"generic arg same shape", view + `[map{"a": int}]`, view + `[map{"a": int}]`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tree := analyseNoPanic(t, `{{ template "child" . }}`,
				loadHintType(t, tc.arg), map[string]types.Type{"child": loadHintType(t, tc.want)})
			if tc.wantErr {
				assert.Equal(t, []ErrorType{ErrorTypeInvalidTemplateArg}, errorTypes(tree),
					typeErrorMessages(tree))
			} else {
				assert.Empty(t, tree.TypeErrors, typeErrorMessages(tree))
			}
		})
	}
}

// A real pointer to a dict (e.g. from a generic `*T` field with T = map{...})
// must not panic and follows text/template's pointer rules: *T is accepted
// where T is expected, but not the reverse.
func TestDictSafe_TemplateArgPointerToDict(t *testing.T) {
	d1 := loadHintType(t, `map{"a": int}`)
	d2 := loadHintType(t, `map{"a": int}`)
	p1, p2 := types.NewPointer(d1), types.NewPointer(d2)

	tree := analyseNoPanic(t, `{{ template "child" . }}`, p1, map[string]types.Type{"child": p2})
	assert.Empty(t, tree.TypeErrors, typeErrorMessages(tree))

	tree = analyseNoPanic(t, `{{ template "child" . }}`, p1, map[string]types.Type{"child": d2})
	assert.Empty(t, tree.TypeErrors, typeErrorMessages(tree))

	tree = analyseNoPanic(t, `{{ template "child" . }}`, d1, map[string]types.Type{"child": p2})
	assert.Equal(
		t,
		[]ErrorType{ErrorTypeInvalidTemplateArg},
		errorTypes(tree),
		typeErrorMessages(tree),
	)

	other := types.NewPointer(loadHintType(t, `map{"b": int}`))
	tree = analyseNoPanic(t, `{{ template "child" . }}`, p1, map[string]types.Type{"child": other})
	assert.Equal(
		t,
		[]ErrorType{ErrorTypeInvalidTemplateArg},
		errorTypes(tree),
		typeErrorMessages(tree),
	)
}

func TestDictSafe_CompatibilityHelpersWithPointerToDict(t *testing.T) {
	p1 := types.NewPointer(loadHintType(t, `map{"a": int}`))
	p2 := types.NewPointer(loadHintType(t, `map{"a": int}`))
	mapStringString := types.NewMap(types.Typ[types.String], types.Typ[types.String])

	require.NotPanics(t, func() {
		assert.True(t, TypeConvertibleTo(p1, mapStringAnyType()))
		assert.False(t, TypeConvertibleTo(p1, mapStringString))
		assert.True(t, typesCompatible(p2, p1))
		assert.True(t, typesCompatible(AnyType(), p1))
	})
}

// Reassigning a variable compares old and new types with go/types.Identical,
// which panicked when the old type was (or contained) a dict.
func TestDictSafe_VariableReassignment(t *testing.T) {
	dot := loadHintType(t, `map{"a": []map{"x": int}, "b": []map{"y": int}, "c": []map{"x": int}}`)
	cases := []struct {
		src  string
		warn bool
	}{
		{`{{ $x := . }}{{ $x = 1 }}`, true},
		{`{{ $x := .a }}{{ $x = .b }}`, true},
		{`{{ $x := .a }}{{ $x = .c }}`, false},
		{`{{ $x := . }}{{ $x = . }}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			tree := analyseNoPanic(t, tc.src, dot, nil)
			if tc.warn {
				assert.Equal(t, []ErrorType{ErrorTypeVariableReassigned}, errorTypes(tree),
					typeErrorMessages(tree))
			} else {
				assert.Empty(t, tree.TypeErrors, typeErrorMessages(tree))
			}
		})
	}
}

func TestDictSafe_Relations(t *testing.T) {
	d := loadHintType(t, `map{"a": int}`)
	dSame := loadHintType(t, `map{"a": int}`)
	dWider := loadHintType(t, `map{"a": int, "b": string}`)
	msa := mapStringAnyType()
	mss := types.NewMap(types.Typ[types.String], types.Typ[types.String])
	errIface := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	view := func(arg string) types.Type { return loadHintType(t, pkgModel+`.View[`+arg+`]`) }

	require.NotPanics(t, func() {
		// Identity: same keys and identical values, never against a real map.
		assert.True(t, DictSafeIdentical(d, dSame))
		assert.True(t, DictSafeIdentical(types.NewSlice(d), types.NewSlice(dSame)))
		assert.True(t, DictSafeIdentical(view(`map{"a": int}`), view(`map{"a": int}`)))
		assert.False(t, DictSafeIdentical(d, dWider))
		assert.False(t, DictSafeIdentical(dWider, d))
		assert.False(t, DictSafeIdentical(d, msa))
		assert.False(t, DictSafeIdentical(msa, d))

		// Assignability: dicts relate to map[string]any and by key coverage.
		assert.True(t, DictSafeAssignableTo(d, AnyType()))
		assert.True(t, DictSafeAssignableTo(d, msa))
		assert.True(t, DictSafeAssignableTo(msa, d))
		assert.False(t, DictSafeAssignableTo(d, mss))
		assert.True(t, DictSafeAssignableTo(dWider, d))
		assert.False(t, DictSafeAssignableTo(d, dWider))
		assert.True(t, DictSafeAssignableTo(types.NewSlice(d), types.NewSlice(msa)))
		assert.False(t, DictSafeAssignableTo(types.NewSlice(d), types.NewSlice(AnyType())))
		assert.True(
			t,
			DictSafeAssignableTo(view(`map{"a": int, "b": string}`), view(`map{"a": int}`)),
		)
		assert.False(t, DictSafeConvertibleTo(d, mss))

		// Implements: a dict has no methods; View[dict] keeps its real ones.
		assert.False(t, DictSafeImplements(d, errIface))
		assert.False(t, DictSafeImplements(view(`map{"a": int}`), errIface))

		// Types without dicts are delegated to go/types unchanged.
		assert.True(t, DictSafeIdentical(msa, mapStringAnyType()))
		assert.True(t, DictSafeAssignableTo(types.Typ[types.String], types.Typ[types.String]))
		assert.False(t, DictSafeAssignableTo(types.Typ[types.Int], types.Typ[types.String]))
		assert.True(t, DictSafeConvertibleTo(types.Typ[types.Int], types.Typ[types.Float64]))
	})
}
