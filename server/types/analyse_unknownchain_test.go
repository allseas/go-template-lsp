package types

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestChain_UnknownIntermediateTypePassesThrough is a regression test: when a
// field/method chain reaches an intermediate value of unknown type (the empty
// interface, e.g. a dict key whose value type could not be resolved, or a dot
// with no gotype hint), further access must pass through as any WITHOUT a false
// "type interface{} has no field or method X" diagnostic.
func TestChain_UnknownIntermediateTypePassesThrough(t *testing.T) {
	cases := []struct {
		name string
		dot  types.Type
	}{
		{"nil dot", nil},
		{"any dot", AnyType()},
		{"dict value is any", &DictType{Fields: map[string]types.Type{"Block": AnyType()}}},
		{"dict value is nil", &DictType{Fields: map[string]types.Type{"Block": nil}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// $.Block.Members exercises the variable chain (analyseVariable);
			// .Block.Members exercises the field chain (analyseField).
			for _, src := range []string{`{{ $.Block.Members }}`, `{{ .Block.Members }}`} {
				treeSet := parseTreeSet(t, src)
				rt := treeSet["t"]
				require.NotNil(t, rt)
				tree := NewTree(*rt, BuiltinFuncs(), c.dot, nil, nil)
				require.Emptyf(t, tree.TypeErrors,
					"src %q dot %s: %v", src, c.name, typeErrorMessages(&tree))
			}
		})
	}
}
