package handlers

import (
	"go/types"
	"strings"
	"testing"

	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"golang.org/x/tools/go/packages"
)

func TestDefinitionVariable(t *testing.T) {
	src := "{{ $test := 0 }}\n{{ $test }}"
	uri := "file:///def-var.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	assert.Len(t, locations, 1)
	assert.Equal(t, uri, locations[0].URI)
	assert.Equal(t, position(0, 3), locations[0].Range.Start)
}

func TestDefinitionVariableRedeclaration(t *testing.T) {
	src := "{{ $test := 0 }}\n{{ $test }}\n{{ $test := 1 }}\n{{ $test }}"
	uri := "file:///def-redecl.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(3, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	assert.Len(t, locations, 2)
}

func TestDefinitionVariableOnDeclaration(t *testing.T) {
	src := "{{ $test := 0 }}\n{{ $test }}"
	uri := "file:///def-on-decl.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	assert.Len(t, locations, 1)
	assert.Equal(
		t,
		protocol.Location{
			URI: uri,
			Range: protocol.Range{
				Start: protocol.Position{Line: 0x0, Character: 0x3},
				End:   protocol.Position{Line: 0x0, Character: 0x8},
			},
		},
		locations[0],
	)
}

func TestDefinitionDotInRange(t *testing.T) {
	src := "{{- range .Join }}\n{{ . }}\n{{- end }}"
	uri := "file:///def-dot-range.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 3),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.Equal(t, uri, loc.URI)

	assert.Equal(t, uint32(0), loc.Range.Start.Line)
}

func TestDefinitionDotInWith(t *testing.T) {
	src := "{{- with .Obj }}\n{{ . }}\n{{- end }}"
	uri := "file:///def-dot-with.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 3),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.Equal(t, uint32(0), loc.Range.Start.Line)
	assert.Equal(t, uint32(0), loc.Range.End.Line)
}

func TestDefinitionDotOutsideRange(t *testing.T) {
	src := "{{ . }}"
	uri := "file:///def-dot-top.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 3),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinitionField(t *testing.T) {
	src := "{{ .Name }}"
	uri := "file:///def-field.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinitionNoNode(t *testing.T) {
	src := "hello {{ $x := 1 }}"
	uri := "file:///def-nonode.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 1),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinitionNoDefinition(t *testing.T) {
	src := "{{ $variable }}\ntext"
	uri := "file:///def-no-def.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// setDocWithType inserts a document into the store with a pre-loaded type,
// bypassing the type-hint loading mechanism that requires a workspace root.
func setDocWithType(t *testing.T, uri, src string, lt *serverTypes.Tree) {
	t.Helper()
	tree, _, err := tryParse(src)
	require.NoError(t, err)
	store.mu.Lock()
	store.docs[uri] = &document{text: src, tree: tree, loadedType: lt}
	store.mu.Unlock()
}

func definitionOrderType(t *testing.T) *serverTypes.Tree {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedFiles,
		Dir:  "../../test/resources/definition-tests-server",
	}
	pkgs, err := packages.Load(cfg, "text-template-server/src/model")
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

func TestDefinitionFieldWithType(t *testing.T) {
	src := "{{ .CustomerName }}"
	uri := "file:///def-field-type.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(70), loc.Range.Start.Line)
}

func TestDefinitionFieldMethodWithType(t *testing.T) {
	src := "{{ .DisplayName }}"
	uri := "file:///def-field-method.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(79), loc.Range.Start.Line)
}

func TestDefinitionNestedFieldFirstIdent(t *testing.T) {
	// cursor on "Address"
	src := "{{ .Address.City }}"
	uri := "file:///def-nested-first.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	// "{{ ." is 4 chars, so char 5 is inside "Address"
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(72), loc.Range.Start.Line)
}

func TestDefinitionNestedFieldSecondIdent(t *testing.T) {
	// cursor on "City" (the second identifier)
	src := "{{ .Address.City }}"
	uri := "file:///def-nested-second.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	// "{{ .Address." is 12 chars, so char 13 is inside "City"
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 13),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(7), loc.Range.Start.Line)
}

func TestDefinitionFieldUnknownField(t *testing.T) {
	src := "{{ .NonExistent }}"
	uri := "file:///def-field-unknown.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinitionFieldNoLoadedType(t *testing.T) {
	src := "{{ .CustomerName }}"
	uri := "file:///def-field-notype.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestFunctionReturn(t *testing.T) {
	src := "{{ .Address.Copy.Copy.Street }}"
	uri := "file:///def-field-func.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 24),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(6), loc.Range.Start.Line)
}
