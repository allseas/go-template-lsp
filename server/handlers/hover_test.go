package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type hoverTestCase struct {
	name                   string
	documentText           string
	positionLine           uint32
	positionCharacterStart uint32
	positionCharacterEnd   uint32
	expectedHover          *protocol.Hover
	expectingError         bool
}

var docText = `
{{ with .User }}
	Name: {{ .Name }}
	Age: {{.Age}}
	{{- range .Roles }}
    	- {{ . }}
	{{- end }}
	{{ if .IsActive }}
	{{ and (.Likes) (ge len .Permissions 5) | not }}
	{{ else }}
	{{ $lastLogin := .LastLogin }}
	{{range $i, $v := .LoginHistory }}
		{{ $i }}: {{ $v }} - {{ $lastLogin }}
	{{ end }}
	{{ end }}
{{ end }}
`

var hoverTestCases = []hoverTestCase{
	{
		name:                   "FieldNode hover - simple field access",
		documentText:           docText,
		positionLine:           2,
		positionCharacterStart: 10,
		positionCharacterEnd:   15,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "**Field Access** - `.Name`\n\nAccesses the `Name` field of the `.` context.",
			},
		},
		expectingError: false,
	},
	{
		name:                   "IdentifierNode hover - variable identifier",
		documentText:           docText,
		positionLine:           12,
		positionCharacterStart: 26,
		positionCharacterEnd:   36,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "**Variable** - `$lastLogin`\n\n",
			},
		},
		expectingError: false,
	},
	{
		name:                   "Control structure hover - if statement",
		documentText:           docText,
		positionLine:           7,
		positionCharacterStart: 3,
		positionCharacterEnd:   10,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "if statement - `if` is used to conditionally execute a block of code based on the truthiness of the expression that follows it.",
			},
		},
		expectingError: false,
	},
}

func TestHover(t *testing.T) {
	for _, tc := range hoverTestCases {
		t.Run(tc.name, func(t *testing.T) {
			for c := tc.positionCharacterStart; c <= tc.positionCharacterEnd; c++ {
				enableServer(t)

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
				hoverResult, err := hover(nil, params)

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
						Line:      tc.positionLine,
						Character: tc.positionCharacterEnd,
					},
				}, hoverResult.Range)
			}
		})
	}
}
