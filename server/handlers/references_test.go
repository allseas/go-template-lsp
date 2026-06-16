package handlers

import (
	"testing"
	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// helpers

func position(line, char uint32) protocol.Position {
	return protocol.Position{Line: line, Character: char}
}

// references tests

func TestReferencesFindsAllOccurrences(t *testing.T) {
	src := `{{ $x := 1 }}
			{{ $x }}
			{{ $x }}`
	uri := "file:///test.tmpl"
	store.Set(uri, src)

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 17),
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}

	results, err := References(nil, params)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestReferencesIdentifier(t *testing.T) {
	src := `{{ printf "a" }}
			{{ printf "b" }}`
	uri := "file:///test2.tmpl"
	store.Set(uri, src)

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 4),
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}

	results, err := References(nil, params)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	store.Delete(uri)
}

func TestReferencesCursorOnNonNode(t *testing.T) {
	src := `hello {{ $x := 1 }}`
	uri := "file:///test3.tmpl"
	store.Set(uri, src)

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 1),
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}

	results, err := References(nil, params)
	require.NoError(t, err)
	assert.Empty(t, results)
	store.Delete(uri)
}

func TestReferencesMultiline(t *testing.T) {
	src := "{{ $x := 1 }}\n" +
		"{{ $x }}"
	uri := "file:///multiline.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 3),
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	}

	results, err := References(nil, params)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// nodeKey tests

func TestNodeKeyVariable(t *testing.T) {
	node := &serverTypes.VariableNode{Ident: []string{"$x"}}
	key, ok := nodeKey(node)
	assert.True(t, ok)
	assert.Equal(t, "var:$x", key)
}

func TestNodeKeyIdentifier(t *testing.T) {
	n := &serverTypes.IdentifierNode{Ident: "printf"}
	key, ok := nodeKey(n)
	assert.True(t, ok)
	assert.Equal(t, "id:printf", key)
}

func TestNodeKeyFieldNode(t *testing.T) {
	n := &serverTypes.FieldNode{Ident: []string{"Name"}}
	_, ok := nodeKey(n)
	assert.False(t, ok)
}

func TestNodeKeyEmptyVariable(t *testing.T) {
	n := &serverTypes.VariableNode{Ident: []string{}}
	_, ok := nodeKey(n)
	assert.False(t, ok)
}

// isVarDecl tests

func TestIsVarDeclTrue(t *testing.T) {
	n := &serverTypes.VariableNode{Ident: []string{"$x"}}
	assert.True(t, isVarDecl(n, "var:$x"))
}

func TestIsVarDeclWrongKey(t *testing.T) {
	n := &serverTypes.VariableNode{Ident: []string{"$x"}}
	assert.False(t, isVarDecl(n, "var:$y"))
}

func TestIsVarDeclNotVariable(t *testing.T) {
	n := &serverTypes.IdentifierNode{Ident: "printf"}
	assert.False(t, isVarDecl(n, "id:printf"))
}

// TestReferencesMultiDefines tests References inside a document with
// multiple {{define}} blocks.
func TestReferencesMultiDefines(t *testing.T) {
	src := `{{- define "A" -}}
{{- /*gotype: text-template-server/src/model.Order*/ -}}
{{ $local := .CustomerName }}
A: {{ $local }} {{ $local }}
{{- end -}}
{{- define "B" -}}
{{ $local := "b" }}
B: {{ $local }}
{{- end -}}
`
	uri := "file:///ref-multidefines.tmpl"
	setDocMulti(t, uri, src, nil)
	t.Cleanup(func() { store.Delete(uri) })

	cases := []struct {
		name          string
		posSubStr     string
		posOccurrence int
		wantCount     int
	}{
		{
			name:          "cursor on $local inside A finds only A's three refs",
			posSubStr:     "$local",
			posOccurrence: 0, // first $local: declaration in A
			wantCount:     3,
		},
		{
			name:          "cursor on $local inside B finds only B's two refs",
			posSubStr:     "$local",
			posOccurrence: 3, // first $local in B: declaration
			wantCount:     2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := posOfSubStr(t, src, tc.posSubStr, tc.posOccurrence)
			pos.Character++ // land inside the identifier

			params := &protocol.ReferenceParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     pos,
				},
				Context: protocol.ReferenceContext{IncludeDeclaration: true},
			}

			results, err := References(nil, params)
			require.NoError(t, err)
			assert.Len(t, results, tc.wantCount)
			for _, loc := range results {
				assert.Equal(t, uri, loc.URI)
			}
		})
	}
}
