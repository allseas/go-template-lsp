package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
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

	result, err := definition(nil, params)
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

	result, err := definition(nil, params)
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

	result, err := definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	assert.Len(t, locations, 1)
	assert.Equal(
		t,
		protocol.Location(
			protocol.Location{
				URI: uri,
				Range: protocol.Range{
					Start: protocol.Position{Line: 0x0, Character: 0x3},
					End:   protocol.Position{Line: 0x0, Character: 0x8},
				},
			},
		),
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

	result, err := definition(nil, params)
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

	result, err := definition(nil, params)
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

	result, err := definition(nil, params)
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

	result, err := definition(nil, params)
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

	result, err := definition(nil, params)
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

	result, err := definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}
