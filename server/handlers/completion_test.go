package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCompletionLogic(t *testing.T) {
	uri := "file:///scope-test.tmpl"
	// Removed leading spaces to ensure Line/Char math is predictable
	content := `{{ $top := . }}
		{{ range $i, $v := .Items }}
		{{ $he
		{{ end }}
		{{ $late := . }}`

	// Manually set the store for the test
	store.Set(uri, content)

	params := &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position: protocol.Position{
				Line:      2, // 0-indexed: Line 2 is "{{ $he"
				Character: 5, // After "{{ $he" (including the space)
			},
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

	// Scope checks
	assert.Contains(t, labels, "$top", "Top-level variable should be visible")
	assert.Contains(t, labels, "$i", "Index variable should be visible inside range")
	assert.Contains(t, labels, "$v", "Value variable should be visible inside range")

	// Ensure variables defined AFTER the cursor are NOT visible
	assert.NotContains(t, labels, "$late", "$late should be out of scope (defined after cursor)")

	// Global checks
	assert.Contains(t, labels, "len", "Global functions should be included")
	assert.Contains(t, labels, "range", "Keywords should be included")

	// Cleanup
	store.Remove(uri)
}
