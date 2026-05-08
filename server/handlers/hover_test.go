package handlers

import (
	"testing"
	"text/template/parse"

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
				Value: MessageDot(&parse.DotNode{}),
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
				Value: MessageVariable(&parse.VariableNode{Ident: []string{"$lastLogin"}}),
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
				Value: MessageBranch(&parse.BranchNode{NodeType: parse.NodeIf}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Control structure hover - range statement",
		documentText:           docText,
		positionLine:           5,
		positionCharacterStart: 3,
		positionCharacterEnd:   10,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageBranch(&parse.BranchNode{NodeType: parse.NodeRange}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Control structure hover - with statement",
		documentText:           docText,
		positionLine:           1,
		positionCharacterStart: 3,
		positionCharacterEnd:   10,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageBranch(&parse.BranchNode{NodeType: parse.NodeWith}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Variable hover - index in range loop",
		documentText:           docText,
		positionLine:           6,
		positionCharacterStart: 5,
		positionCharacterEnd:   10,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIndexVariable(&parse.VariableNode{Ident: []string{"$i"}}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Function hover - and function",
		documentText:           docText,
		positionLine:           8,
		positionCharacterStart: 3,
		positionCharacterEnd:   6,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIdentifier(&parse.IdentifierNode{Ident: "and"}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Function hover - not function",
		documentText:           docText,
		positionLine:           8,
		positionCharacterStart: 11,
		positionCharacterEnd:   14,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIdentifier(&parse.IdentifierNode{Ident: "not"}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Function hover - ge function",
		documentText:           docText,
		positionLine:           8,
		positionCharacterStart: 16,
		positionCharacterEnd:   18,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIdentifier(&parse.IdentifierNode{Ident: "ge"}),
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
