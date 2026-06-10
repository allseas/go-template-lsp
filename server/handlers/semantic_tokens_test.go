package handlers

import (
	"go/types"
	"testing"

	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"golang.org/x/tools/go/packages"
)

// semanticOrderType loads the Order type from the templ-tests model package.
func semanticOrderType(t *testing.T) *serverTypes.Tree {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedFiles,
		Dir:  "../../test/resources/templ-tests",
	}
	pkgs, err := packages.Load(cfg, "cg/model")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	pkg := pkgs[0]
	obj := pkg.Types.Scope().Lookup("Order")
	require.NotNil(t, obj)
	named, ok := obj.Type().(*types.Named)
	require.True(t, ok)
	return &serverTypes.Tree{
		DotType: named,
		Pkg:     pkg.Types,
		Fset:    pkg.Fset,
	}
}

// setDocWithTypedTree inserts a document with a fully built typed tree into the store,
// bypassing the type-hint loading mechanism that requires a workspace root.
func setDocWithTypedTree(t *testing.T, uri, src string, lt *serverTypes.Tree) {
	t.Helper()
	tree, err := tryParse(src)
	require.NoError(t, err)
	typedTree := buildTypedTree(tree, lt)
	store.mu.Lock()
	store.docs[uri] = &document{text: src, tree: tree, loadedType: lt, typedTree: typedTree}
	store.mu.Unlock()
}

// TestSemanticTokensFull exercises the full handler path via the document store.
func TestSemanticTokensFull(t *testing.T) {
	lt := semanticOrderType(t)

	tests := []struct {
		name    string
		src     string
		wantLen int // expected number of tokens * 5 (LSP encoded format)
		wantNil bool
	}{
		{
			name:    "dot",
			src:     `{{.}}`,
			wantLen: 5, // one token
		},
		{
			name:    "struct field",
			src:     `{{.CustomerName}}`,
			wantLen: 5,
		},
		{
			name:    "method call",
			src:     `{{.DisplayName}}`,
			wantLen: 5,
		},
		{
			name:    "chained field.field",
			src:     `{{.Address.City}}`,
			wantLen: 10, // two tokens
		},
		{
			name:    "chained field.method",
			src:     `{{.Address.Line}}`,
			wantLen: 10,
		},
		{
			name:    "no typed tree when server disabled",
			src:     `{{.}}`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "file:///sem-" + tt.name + ".tmpl"

			if tt.wantNil {
				// document not in store.
				result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
				require.NoError(t, err)
				assert.Nil(t, result)
				return
			}

			setDocWithTypedTree(t, uri, tt.src, lt)
			defer store.Delete(uri)

			result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Data, tt.wantLen)
		})
	}
}

// TestFieldNodeTokenTypes verifies that struct fields are emitted as ttProperty
// and method calls are emitted as ttFunction.
// The LSP semantic token data is a flat uint32 array with 5 values per token:
func TestFieldNodeTokenTypes(t *testing.T) {
	lt := semanticOrderType(t)

	tests := []struct {
		name           string
		src            string
		noType         bool // when true, don't pass type info
		wantTokenCount int
		// tokenType for each token in order, indexed as result.Data[n*5+3]
		wantTypes []uint32
	}{
		{
			// .CustomerName is a plain struct field (string)
			name:           "struct field emits property token",
			src:            `{{.CustomerName}}`,
			wantTokenCount: 1,
			wantTypes:      []uint32{ttProperty},
		},
		{
			// .DisplayName is a method (func returning string)
			name:           "method emits function token",
			src:            `{{.DisplayName}}`,
			wantTokenCount: 1,
			wantTypes:      []uint32{ttFunction},
		},
		{
			// .Address is a struct field, .City is a struct field on Address
			name:           "chained field.field both emit property tokens",
			src:            `{{.Address.City}}`,
			wantTokenCount: 2,
			wantTypes:      []uint32{ttProperty, ttProperty},
		},
		{
			// .Address is a struct field, .Line is a method on Address
			name:           "chained field.method emits property then function token",
			src:            `{{.Address.Line}}`,
			wantTokenCount: 2,
			wantTypes:      []uint32{ttProperty, ttFunction},
		},
		{
			// .Address.Info.Desc, three-level all-field chain
			name:           "three-level field chain all property tokens",
			src:            `{{.Address.Info.Desc}}`,
			wantTokenCount: 3,
			wantTypes:      []uint32{ttProperty, ttProperty, ttProperty},
		},
		{
			// no type info, all segments fall back to ttProperty
			name:           "without type info field falls back to property",
			src:            `{{.SomeField}}`,
			noType:         true,
			wantTokenCount: 1,
			wantTypes:      []uint32{ttProperty},
		},
		{
			// .IsLargeOrder is a method on Order returning bool
			name:           "bool-returning method emits function token",
			src:            `{{.IsLargeOrder}}`,
			wantTokenCount: 1,
			wantTypes:      []uint32{ttFunction},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "file:///field-tok-" + tt.name + ".tmpl"
			useLt := lt
			if tt.noType {
				useLt = nil
			}
			setDocWithTypedTree(t, uri, tt.src, useLt)
			defer store.Delete(uri)

			result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Data, tt.wantTokenCount*5, "unexpected token count")

			for i, wantType := range tt.wantTypes {
				assert.Equal(t, wantType, result.Data[i*5+3], "token[%d] tokenType", i)
			}
		})
	}
}

func makeSemanticParams(uri string) *protocol.SemanticTokensParams {
	return &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}
}

// TestSemanticTokenNodeTypes covers each node kind handled by walkSemanticNode.
// Each case specifies the exact token types (result.Data[i*5+3]) and, when relevant,
// the token modifiers (result.Data[i*5+4]) expected in source order.
func TestSemanticTokenNodeTypes(t *testing.T) {
	lt := semanticOrderType(t)

	tests := []struct {
		name          string
		src           string
		noType        bool
		wantTypes     []uint32 // result.Data[i*5+3] for each token i
		wantModifiers []uint32 // result.Data[i*5+4]; nil means don't check
	}{
		{
			name:      "comment",
			src:       `{{/* hello */}}`,
			noType:    true,
			wantTypes: []uint32{ttComment},
		},
		{
			name:      "dot",
			src:       `{{.}}`,
			noType:    true,
			wantTypes: []uint32{ttVariable},
		},
		{
			name:          "variable declaration",
			src:           `{{$x := .}}`,
			noType:        true,
			wantTypes:     []uint32{ttVariable, ttVariable},
			wantModifiers: []uint32{tmDeclaration, 0},
		},
		{
			name:      "variable usage",
			src:       `{{$x := .}}{{$x}}`,
			noType:    true,
			wantTypes: []uint32{ttVariable, ttVariable, ttVariable},
		},
		{
			name:          "builtin identifier",
			src:           `{{len .}}`,
			noType:        true,
			wantTypes:     []uint32{ttFunction, ttVariable},
			wantModifiers: []uint32{tmDefaultLibrary, 0},
		},
		{
			name:      "user function identifier",
			src:       `{{printf "%s" "x"}}`,
			noType:    true,
			wantTypes: []uint32{ttFunction, ttString, ttString},
		},
		{
			name:      "string",
			src:       `{{"hello"}}`,
			noType:    true,
			wantTypes: []uint32{ttString},
		},
		{
			name:      "number",
			src:       `{{42}}`,
			noType:    true,
			wantTypes: []uint32{ttNumber},
		},
		{
			name:      "bool true",
			src:       `{{true}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword},
		},
		{
			name:      "bool false",
			src:       `{{false}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword},
		},
		{
			name:      "nil",
			src:       `{{nil}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword},
		},
		{
			name:      "if/end",
			src:       `{{if .}}{{end}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword, ttVariable, ttKeyword},
		},
		{
			name:      "if/else/end",
			src:       `{{if .}}{{else}}{{end}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword, ttVariable, ttKeyword, ttKeyword},
		},
		{
			// .Items is a []Item field on Order
			name:      "range/end",
			src:       `{{range .Items}}{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword},
		},
		{
			// .Address is a struct field on Order
			name:      "with/end",
			src:       `{{with .Address}}{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword},
		},
		{
			name:      "template",
			src:       `{{template "sub"}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword, ttString},
		},
		{
			name:      "break",
			src:       `{{range .Items}}{{break}}{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword, ttKeyword},
		},
		{
			name:      "continue",
			src:       `{{range .Items}}{{continue}}{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword, ttKeyword},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "file:///node-tok-" + tt.name + ".tmpl"
			useLt := lt
			if tt.noType {
				useLt = nil
			}
			setDocWithTypedTree(t, uri, tt.src, useLt)
			defer store.Delete(uri)

			result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Data, len(tt.wantTypes)*5, "unexpected token count")

			for i, wantType := range tt.wantTypes {
				assert.Equal(t, wantType, result.Data[i*5+3], "token[%d] tokenType", i)
			}
			for i, wantMod := range tt.wantModifiers {
				assert.Equal(t, wantMod, result.Data[i*5+4], "token[%d] modifiers", i)
			}
		})
	}
}
