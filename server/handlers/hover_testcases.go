package handlers

import (
	serverTypes "text-template-server/types"

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
	{{- end -}}
	{{- end -}}
{{- end -}}
`

var docElseText = `
{{ if . }}
a
{{ range .Items }}
l
{{ end }}
{{- else if . }}
b
{{- else }}
c
{{ end }}
`

var (
	docRootNode      = parseTypedRoot(docText)
	shortDocRootNode = parseTypedRoot(shortDocText)
	elseRootNode     = parseTypedRoot(docElseText)
)

// parseTypedRoot parses src and returns the analysed (typed) tree, or nil if
// parsing failed. Used by the hover testcases to construct expected hover
// values that reference the typed AST.
func parseTypedRoot(src string) *serverTypes.Tree {
	tree, _, _ := tryParse(src)
	if tree == nil {
		return nil
	}
	return buildTypedTree(tree, nil, nil)
}

var shortDocText = `
{{range $i, $v := .Items }}
	{{$i}} - {{ $v }} 
{{ end }}`

var hoverTestCases = []hoverTestCase{
	{
		name:                   "else if hover",
		documentText:           docElseText,
		positionLine:           8,
		endLine:                8,
		positionCharacterStart: 0,
		positionCharacterEnd:   7,
		positionRangeEnd:       8,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageElse(
					&elseRootNode.Root.Nodes[1],
					protocol.Position{Line: 6, Character: 0},
				),
			},
		},
	},
	{
		name:                   "else hover",
		documentText:           docElseText,
		positionLine:           8,
		endLine:                8,
		positionCharacterStart: 0,
		positionCharacterEnd:   7,
		positionRangeEnd:       8,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageElse(
					&elseRootNode.Root.Nodes[1],
					protocol.Position{Line: 6, Character: 0},
				),
			},
		},
	},
	{
		name:                   "triple end tags",
		documentText:           docText,
		positionLine:           15,
		endLine:                15,
		positionCharacterStart: 0,
		positionCharacterEnd:   10,
		positionRangeEnd:       11,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageEnd(
					docRootNode.Root.Nodes[1],
					protocol.Position{Line: 1, Character: 3},
				),
			},
		},
		expectingError: false,
	},
	{
		name:                   "double end tags",
		documentText:           docText,
		positionLine:           14,
		endLine:                14,
		positionCharacterStart: 1,
		positionCharacterEnd:   11,
		positionRangeEnd:       12,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageEnd(
					docRootNode.Root.Nodes[1].(*serverTypes.WithNode).List.Nodes[6],
					protocol.Position{Line: 7, Character: 3},
				),
			},
		},
		expectingError: false,
	},
	{
		name:                   "end tag :cc",
		documentText:           shortDocText,
		positionLine:           3,
		endLine:                3,
		positionCharacterStart: 0,
		positionCharacterEnd:   9,
		positionRangeEnd:       9,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageEnd(
					shortDocRootNode.Root.Nodes[1],
					protocol.Position{Line: 1, Character: 0},
				),
			},
		},
		expectingError: false,
	},
	{
		name:                   "end tag - hover on end tag of if statement",
		documentText:           docText,
		positionLine:           15,
		endLine:                15,
		positionCharacterStart: 0,
		positionCharacterEnd:   8,
		positionRangeEnd:       11,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageEnd(
					docRootNode.Root.Nodes[1],
					protocol.Position{Line: 1, Character: 3},
				),
			},
		},
		expectingError: false,
	},
	{
		name:                   "end tag - hover on end tag of if statement",
		documentText:           docText,
		positionLine:           6,
		endLine:                6,
		positionCharacterStart: 1,
		positionCharacterEnd:   10,
		positionRangeEnd:       11,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageEnd(
					docRootNode.Root.Nodes[1].(*serverTypes.WithNode).List.Nodes[4],
					protocol.Position{Line: 4, Character: 3},
				),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Index variable hover use - index variable used in range loop body",
		documentText:           shortDocText,
		positionLine:           2,
		endLine:                2,
		positionCharacterStart: 3,
		positionCharacterEnd:   5,
		positionRangeEnd:       5,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIndexVariable(&serverTypes.VariableNode{Ident: []string{"$i"}}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Index variable hover - range loop index variable",
		documentText:           shortDocText,
		positionLine:           1,
		endLine:                1,
		positionCharacterStart: 8,
		positionCharacterEnd:   10,
		positionRangeEnd:       10,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIndexVariable(&serverTypes.VariableNode{Ident: []string{"$i"}}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Value variable hover - range loop value variable",
		documentText:           shortDocText,
		positionLine:           1,
		endLine:                1,
		positionCharacterStart: 12,
		positionCharacterEnd:   14,
		positionRangeEnd:       14,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageVariable(&serverTypes.VariableNode{Ident: []string{"$v"}}, nil, nil),
			},
		},
		expectingError: false,
	},

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
				Value: MessageField(&serverTypes.FieldNode{Ident: []string{"Name"}}, nil),
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
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageVariable(
					&serverTypes.VariableNode{Ident: []string{"$lastLogin"}},
					nil,
					nil,
				),
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
		positionCharacterEnd:   1,
		positionRangeEnd:       5,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageBranch(
					&serverTypes.BranchNode{
						NodeType: serverTypes.NodeIf,
						Pipe: &serverTypes.PipeNode{
							Cmds: []*serverTypes.CommandNode{
								{
									Args: []serverTypes.Node{
										&serverTypes.IdentifierNode{Ident: ".IsActive"},
									},
								},
							},
						},
					},
				),
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
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageBranch(
					&serverTypes.BranchNode{
						NodeType: serverTypes.NodeRange,
						Pipe: &serverTypes.PipeNode{
							Cmds: []*serverTypes.CommandNode{
								{
									Args: []serverTypes.Node{
										&serverTypes.IdentifierNode{Ident: ".Roles"},
									},
								},
							},
						},
					},
				),
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
		positionRangeEnd:       32,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown,
				Value: MessageBranch(
					&serverTypes.BranchNode{
						NodeType: serverTypes.NodeWith,
						Pipe: &serverTypes.PipeNode{
							Cmds: []*serverTypes.CommandNode{
								{
									Args: []serverTypes.Node{
										&serverTypes.IdentifierNode{Ident: ".User"},
									},
								},
							},
						},
					},
				),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Variable hover - index in range loop",
		documentText:           docText,
		positionLine:           11,
		endLine:                11,
		positionCharacterStart: 9,
		positionCharacterEnd:   11,
		positionRangeEnd:       11,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIndexVariable(&serverTypes.VariableNode{Ident: []string{"$i"}}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Function hover - and function",
		documentText:           docText,
		positionLine:           8,
		endLine:                8,
		positionCharacterStart: 4,
		positionCharacterEnd:   7,
		positionRangeEnd:       7,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIdentifier(&serverTypes.IdentifierNode{Ident: "and"}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Function hover - not function",
		documentText:           docText,
		positionLine:           8,
		endLine:                8,
		positionCharacterStart: 43,
		positionCharacterEnd:   46,
		positionRangeEnd:       46,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIdentifier(&serverTypes.IdentifierNode{Ident: "not"}),
			},
		},
		expectingError: false,
	},
	{
		name:                   "Function hover - ge function",
		documentText:           docText,
		positionLine:           8,
		endLine:                8,
		positionCharacterStart: 18,
		positionCharacterEnd:   20,
		positionRangeEnd:       20,
		expectedHover: &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIdentifier(&serverTypes.IdentifierNode{Ident: "ge"}),
			},
		},
		expectingError: false,
	},
}

// hoverMultiDefineCase covers hover behaviour when a single document contains
// multiple {{define}} blocks, each (optionally) preceded by its own gotype hint.
type hoverMultiDefineCase struct {
	name           string
	posSubStr      string
	posOccurrence  int
	wantSubstrings []string
}

var hoverMultiDefineCases = []hoverMultiDefineCase{
	{
		name:          "field inside Order define resolves against Order type",
		posSubStr:     "CustomerName",
		posOccurrence: 0,
		wantSubstrings: []string{
			"field .CustomerName",
			"Accesses a field from the `model.Order` dot context",
		},
	},
	{
		name:          "field inside Address define resolves against Address type",
		posSubStr:     "Street",
		posOccurrence: 0,
		wantSubstrings: []string{
			"field .Street",
			"Accesses a field from the `model.Address` dot context",
		},
	},
	{
		name:           "variable inside no-hint define still hovers",
		posSubStr:      "$local }}",
		posOccurrence:  0,
		wantSubstrings: []string{"$local"},
	},
	{
		name:          "field in root template resolves against root Address type",
		posSubStr:     ".Country",
		posOccurrence: 0,
		wantSubstrings: []string{
			"field .Country",
			"Accesses a field from the `model.Address` dot context",
		},
	},
}
