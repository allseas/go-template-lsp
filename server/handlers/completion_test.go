package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCompletionLogic(t *testing.T) {
	uri := "file:///scope-test.tmpl"
	content := `
		{{ $top := . }}
		{{ range $i, $v := .Items }}
			{{ $he
		{{ end }}
		{{ $late := . }}
		`

	store.Set(uri, content)

	params := &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 5},
		},
	}

	resp, err := completion(nil, params)

	require.NoError(t, err)

	list, ok := resp.(protocol.CompletionList)
	require.True(t, ok, "response should be a CompletionList")

	var labels []string
	for _, item := range list.Items {
		labels = append(labels, item.Label)
	}

	assert.Contains(t, labels, "$top", "Top-level variable should be visible")
	assert.Contains(t, labels, "$i", "Index variable should be visible inside range")
	assert.Contains(t, labels, "$v", "Value variable should be visible inside range")
	assert.NotContains(t, labels, "$late", "$late should be out of scope (defined after cursor)")

	assert.Subset(t, labels, []string{"len", "printf"}, "Global functions should be included")

	store.Remove(uri)
}
