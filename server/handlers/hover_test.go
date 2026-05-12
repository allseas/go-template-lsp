package handlers

import (
	"testing"
	parse "text-template-parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type hoverTestCase struct {
	name                   string
	documentText           string
	positionLine           uint32
	endLine                uint32
	positionCharacterStart uint32
	positionCharacterEnd   uint32
	positionRangeEnd       uint32
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
		endLine:                2,
		positionCharacterStart: 10,
		positionCharacterEnd:   15,
		positionRangeEnd:       15,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageField(&parse.FieldNode{Ident: []string{"Name"}}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "IdentifierNode hover - variable identifier",
		documentText:           docText,
		positionLine:           12,
		endLine:                12,
		positionCharacterStart: 26,
		positionCharacterEnd:   36,
		positionRangeEnd:       36,
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
		endLine:                13,
		positionCharacterStart: 4,
		positionCharacterEnd:   5,
		positionRangeEnd:       5,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageBranch(&parse.BranchNode{NodeType: parse.NodeIf, Pipe: &parse.PipeNode{Cmds: []*parse.CommandNode{{Args: []parse.Node{&parse.IdentifierNode{Ident: ".IsActive"}}}}}}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Control structure hover - range statement",
		documentText:           docText,
		positionLine:           4,
		endLine:                6,
		positionCharacterStart: 5,
		positionCharacterEnd:   5,
		positionRangeEnd:       5,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageBranch(&parse.BranchNode{NodeType: parse.NodeRange, Pipe: &parse.PipeNode{Cmds: []*parse.CommandNode{{Args: []parse.Node{&parse.IdentifierNode{Ident: ".Roles"}}}}}}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Control structure hover - with statement",
		documentText:           docText,
		positionLine:           1,
		endLine:                12,
		positionCharacterStart: 3,
		positionCharacterEnd:   7,
		positionRangeEnd:       37,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageBranch(&parse.BranchNode{NodeType: parse.NodeWith, Pipe: &parse.PipeNode{Cmds: []*parse.CommandNode{{Args: []parse.Node{&parse.IdentifierNode{Ident: ".User"}}}}}}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Variable hover - index in range loop",
		documentText:           docText,
		positionLine:           11,
		endLine:                11,
		positionCharacterStart: 5,
		positionCharacterEnd:   10,
		positionRangeEnd:       10,
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
		endLine:                8,
		positionCharacterStart: 3,
		positionCharacterEnd:   6,
		positionRangeEnd:       6,
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
		endLine:                8,
		positionCharacterStart: 11,
		positionCharacterEnd:   14,
		positionRangeEnd:       14,
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
		endLine:                8,
		positionCharacterStart: 16,
		positionCharacterEnd:   18,
		positionRangeEnd:       18,
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
						Line:      tc.endLine,
						Character: tc.positionRangeEnd,
					},
				}, hoverResult.Range)
			}
		})
	}
}
