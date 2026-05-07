package handlers

import (
	"testing"
	"text/template/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// helpers
// mustParse helps get a clearer fail in case the test fails for any reason
func mustParse(t *testing.T, src string) *parse.Tree {
	t.Helper()
	tr := parse.New("t")
	tr.Mode = parse.SkipFuncCheck
	treeSet := map[string]*parse.Tree{}
	_, err := tr.Parse(src, "{{", "}}", treeSet)
	require.NoError(t, err)
	return tr
}

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

	results, err := references(nil, params)
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

	results, err := references(nil, params)
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

	results, err := references(nil, params)
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

	results, err := references(nil, params)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// nodeKey tests

func TestNodeKeyVariable(t *testing.T) {
	node := &parse.VariableNode{Ident: []string{"$x"}}
	key, ok := nodeKey(node)
	assert.True(t, ok)
	assert.Equal(t, "var:$x", key)
}

func TestNodeKeyIdentifier(t *testing.T) {
	n := &parse.IdentifierNode{Ident: "printf"}
	key, ok := nodeKey(n)
	assert.True(t, ok)
	assert.Equal(t, "id:printf", key)
}

func TestNodeKeyFieldNode(t *testing.T) {
	n := &parse.FieldNode{Ident: []string{"Name"}}
	_, ok := nodeKey(n)
	assert.False(t, ok)
}

func TestNodeKeyEmptyVariable(t *testing.T) {
	n := &parse.VariableNode{Ident: []string{}}
	_, ok := nodeKey(n)
	assert.False(t, ok)
}

// nodeFind tests

func TestNodeFindVariable(t *testing.T) {
	src := `{{ $x := 1 }}{{ $x }}`
	tr := mustParse(t, src)
	offset := positionToOffset(src, position(0, 17))
	found := nodeFind(tr.Root, parse.Pos(offset))
	require.NotNil(t, found)
	v, ok := found.(*parse.VariableNode)
	require.True(t, ok)
	assert.Equal(t, "$x", v.Ident[0])
}

func TestNodeFindIdentifier(t *testing.T) {
	src := `{{ printf "hello" }}`
	tr := mustParse(t, src)
	offset := positionToOffset(src, position(0, 4))
	found := nodeFind(tr.Root, parse.Pos(offset))
	require.NotNil(t, found)
	id, ok := found.(*parse.IdentifierNode)
	require.True(t, ok)
	assert.Equal(t, "printf", id.Ident)
}

func TestNodeFindOutsideTemplate(t *testing.T) {
	src := `hello {{ $x := 1 }}`
	tr := mustParse(t, src)
	offset := positionToOffset(src, position(0, 1))
	found := nodeFind(tr.Root, parse.Pos(offset))
	assert.NotNil(t, found)
}

// inspect tests

func TestInspectCollectsAll(t *testing.T) {
	src := `{{ $x := 1 }}
			{{ $x }}
			{{ $x }}`
	tr := mustParse(t, src)

	var vars []string
	inspect(tr.Root, func(n parse.Node) bool {
		if v, ok := n.(*parse.VariableNode); ok {
			vars = append(vars, v.Ident[0])
		}
		return true
	})

	assert.Equal(t, []string{"$x", "$x", "$x"}, vars)
}

func TestInspectSkipsChildrenWhenFalse(t *testing.T) {
	src := `{{ if true }}
			{{ $x := 1 }}
			{{ end }}`
	tr := mustParse(t, src)

	var vars []string
	inspect(tr.Root, func(n parse.Node) bool {
		if _, ok := n.(*parse.IfNode); ok {
			return false // skip if body
		}
		if v, ok := n.(*parse.VariableNode); ok {
			vars = append(vars, v.Ident[0])
		}
		return true
	})

	assert.Empty(t, vars)
}

func TestInspectElseList(t *testing.T) {
	src := `{{ if .Cond }}
			{{ $x := 1 }}
			{{ else }}
			{{ $y := 2 }}
			{{ end }}`
	tr := mustParse(t, src)

	var vars []string
	inspect(tr.Root, func(n parse.Node) bool {
		if v, ok := n.(*parse.VariableNode); ok {
			vars = append(vars, v.Ident[0])
		}
		return true
	})

	assert.Contains(t, vars, "$x")
	assert.Contains(t, vars, "$y")
}

// nodeToRange tests

func TestNodeToRangeVariable(t *testing.T) {
	src := `{{ $myVar := 2 }}
          	{{ $myVar }}`
	tr := mustParse(t, src)
	var found *parse.VariableNode
	inspect(tr.Root, func(n parse.Node) bool {
		if found != nil {
			return false
		}
		if v, ok := n.(*parse.VariableNode); ok {
			found = v
		}
		return true
	})
	require.NotNil(t, found)
	r := nodeToRange(found, src)
	assert.Equal(t, uint32(0), r.Start.Line)
	assert.Equal(t, uint32(0), r.End.Line)
	assert.Equal(t, uint32(3), r.Start.Character)
	assert.Equal(t, uint32(9), r.End.Character)
}

// isVarDecl tests

func TestIsVarDeclTrue(t *testing.T) {
	n := &parse.VariableNode{Ident: []string{"$x"}}
	assert.True(t, isVarDecl(n, "var:$x"))
}

func TestIsVarDeclWrongKey(t *testing.T) {
	n := &parse.VariableNode{Ident: []string{"$x"}}
	assert.False(t, isVarDecl(n, "var:$y"))
}

func TestIsVarDeclNotVariable(t *testing.T) {
	n := &parse.IdentifierNode{Ident: "printf"}
	assert.False(t, isVarDecl(n, "id:printf"))
}
