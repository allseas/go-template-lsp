package handlers

import (
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
	enableServer(t)
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
