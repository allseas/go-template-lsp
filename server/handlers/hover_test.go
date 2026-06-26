package handlers

import (
	"go/token"
	"go/types"
	"testing"

	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestHover(t *testing.T) {
	for _, tc := range hoverTestCases {
		t.Run(tc.name, func(t *testing.T) {
			for c := tc.positionCharacterStart; c <= tc.positionCharacterEnd; c++ {
				enableHover(t)

				uri := "file:///test/document.go"
				content := tc.documentText

				store.Set(uri, content)
				t.Cleanup(func() { store.Remove(uri) })

				// Create hover params
				params := &protocol.HoverParams{
					TextDocumentPositionParams: protocol.TextDocumentPositionParams{
						TextDocument: protocol.TextDocumentIdentifier{
							URI: uri,
						},
						Position: protocol.Position{
							Line:      tc.positionLine,
							Character: c,
						},
					},
				}

				// Call the hover handler
				hoverResult, err := Hover(nil, params)

				if tc.expectingError {
					require.Error(t, err)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tc.expectedHover.Contents, hoverResult.Contents)
				assert.Equal(t, &protocol.Range{
					Start: protocol.Position{
						Line:      tc.positionLine,
						Character: tc.positionCharacterStart,
					},
					End: protocol.Position{
						Line:      tc.endLine,
						Character: tc.positionRangeEnd,
					},
				}, hoverResult.Range)
			}
		})
	}
}

// TestHoverMultiDefines exercises hover inside a document with multiple
// {{define}} blocks, each (optionally) preceded by its own gotype hint, to
// verify that the per-tree loaded type is used for resolution.
func TestHoverMultiDefines(t *testing.T) {
	loaded := loadModelTypes(t, "Order", "Address")
	perTree := map[string]*serverTypes.Tree{
		"t":          loaded["Address"],
		"OrderTpl":   loaded["Order"],
		"AddressTpl": loaded["Address"],
	}

	src := multiDefinesTemplate
	uri := "file:///hover-multidefines.tmpl"
	enableHover(t)
	setDocMulti(t, uri, src, perTree)
	t.Cleanup(func() { store.Remove(uri) })

	for _, tc := range hoverMultiDefineCases {
		t.Run(tc.name, func(t *testing.T) {
			pos := posOfSubStr(t, src, tc.posSubStr, tc.posOccurrence)
			pos.Character++ // land inside the identifier rather than on its first byte

			params := &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     pos,
				},
			}

			result, err := Hover(nil, params)
			require.NoError(t, err)
			require.NotNil(t, result, "expected non-nil hover result")

			mc, ok := result.Contents.(protocol.MarkupContent)
			require.True(t, ok, "expected MarkupContent, got %T", result.Contents)
			for _, want := range tc.wantSubstrings {
				assert.Contains(t, mc.Value, want)
			}
		})
	}
}

// TestMessageVariableChainedIdent tests that hover must show the full chain "$order.TotalAmount float64", not just "$order float64".
func TestMessageVariableChainedIdent(t *testing.T) {
	float64Type := types.Typ[types.Float64]

	msg := MessageVariable(
		&serverTypes.VariableNode{Ident: []string{"$order", "TotalAmount"}},
		nil,
		float64Type,
	)

	assert.Contains(t, msg, "$order.TotalAmount")
	assert.Contains(t, msg, "float64")
	assert.NotContains(t, msg, "var $order float64",
		"should show full chain, not just the base variable name")
}

// newSigFunc builds a synthetic *types.Func with the given parameter and
// result types so we can exercise MessageGlobalFunc without loading a real
// Go workspace.
func newSigFunc(name string, params, results []types.Type) *types.Func {
	mk := func(ts []types.Type) *types.Tuple {
		vars := make([]*types.Var, len(ts))
		for i, t := range ts {
			vars[i] = types.NewVar(token.NoPos, nil, "", t)
		}
		return types.NewTuple(vars...)
	}
	sig := types.NewSignatureType(nil, nil, nil, mk(params), mk(results), false)
	return types.NewFunc(token.NoPos, nil, name, sig)
}

func TestMessageGlobalFunc_WithSignatureAndDoc(t *testing.T) {
	fn := newSigFunc(
		"upper",
		[]types.Type{types.Typ[types.String]},
		[]types.Type{types.Typ[types.String]},
	)
	entry := serverTypes.GlobalFuncEntry{Func: fn, Doc: "Upper returns s upper-cased."}

	got := MessageGlobalFunc("upper", entry)

	assert.Contains(t, got, "```go\nupper(string) string\n```")
	assert.Contains(t, got, "Upper returns s upper-cased.")
	assert.NotContains(t, got, "text/template reference")
}

func TestMessageGlobalFunc_NoDoc(t *testing.T) {
	fn := newSigFunc(
		"shout",
		[]types.Type{types.Typ[types.String]},
		[]types.Type{types.Typ[types.String]},
	)
	got := MessageGlobalFunc("shout", serverTypes.GlobalFuncEntry{Func: fn})

	assert.Contains(t, got, "```go\nshout(string) string\n```")
	assert.Contains(t, got, "User-defined template function")
	assert.NotContains(t, got, "text/template reference")
}

func TestMessageGlobalFunc_NoSignature(t *testing.T) {
	got := MessageGlobalFunc("mystery", serverTypes.GlobalFuncEntry{})
	assert.Contains(t, got, "```go\nmystery\n```")
	assert.Contains(t, got, "User-defined template function")
}

func TestMessageIdentifier_PrefersGlobalFuncOverGeneric(t *testing.T) {
	t.Cleanup(func() { serverTypes.SetGlobalFuncEntries(nil) })

	fn := newSigFunc(
		"upper",
		[]types.Type{types.Typ[types.String]},
		[]types.Type{types.Typ[types.String]},
	)
	serverTypes.SetGlobalFuncEntries(map[string]serverTypes.GlobalFuncEntry{
		"upper": {Func: fn, Doc: "Upper returns s upper-cased."},
	})

	got := MessageIdentifier(&serverTypes.IdentifierNode{Ident: "upper"}, nil)

	assert.Contains(t, got, "upper(string) string")
	assert.Contains(t, got, "Upper returns s upper-cased.")
	assert.NotContains(t, got, "Represents an identifier in a command or action.")
	assert.NotContains(t, got, "text/template reference")
}

func TestMessageIdentifier_BuiltinsStillTakePrecedence(t *testing.T) {
	t.Cleanup(func() { serverTypes.SetGlobalFuncEntries(nil) })

	// Even if a user re-registered `and` (which ComputeGlobalFuncs would
	// drop), the builtin special message must win.
	serverTypes.SetGlobalFuncEntries(map[string]serverTypes.GlobalFuncEntry{
		"and": {Doc: "should not appear"},
	})

	got := MessageIdentifier(&serverTypes.IdentifierNode{Ident: "and"}, nil)

	assert.Contains(t, got, "Returns the first empty argument")
	assert.NotContains(t, got, "should not appear")
}
