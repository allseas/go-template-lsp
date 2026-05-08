// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"errors"
	"fmt"
	"strings"
	"text/template/parse"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func hover(ctx *glsp.Context, params *protocol.HoverParams) (hover *protocol.Hover, err error) {
	// Get document content

	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || doc.tree == nil {
		log.Debug().Msg("doc or tree is nil")
		err = errors.New("document not found or failed to parse")
		return
	}

	offset := positionToOffset(doc.text, params.Position)
	target := nodeFind(doc.tree.Root, parse.Pos(offset))
	if target == nil {
		err = errors.New("node not found")
		return
	}
	nodeRange := nodeToRange(target, doc.text)

	hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "",
		},
		Range: &nodeRange,
	}

	switch target.(type) {
	case *parse.ActionNode:
		log.Debug().Msg("Hover on ActionNode")
	case *parse.CommandNode:
		log.Debug().Msg("Hover on CommandNode")
	case *parse.DotNode:
		log.Debug().Msg("Hover on DotNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "**Cursor** - `.`\n\nReturns the current context",
		}
	case *parse.FieldNode:
		log.Debug().Msg("Hover on FieldNode")
		target := target.(*parse.FieldNode)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("**Field Access** - `%s`\n\nAccesses the `%s` field of the `.%s` context.", target.String(), target.Ident[0], strings.Join(target.Ident[1:], ".")),
		}
	case *parse.IdentifierNode:
		log.Debug().Msg("Hover on IdentifierNode")
		target := target.(*parse.IdentifierNode)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("**Identifier** - `%s`\n\nRepresents an identifier in a command or action.", target.Ident),
		}
	case *parse.PipeNode:
		log.Debug().Msg("Hover on PipeNode")
	case *parse.VariableNode:
		target := target.(*parse.VariableNode)
		log.Debug().Msg("Hover on VariableNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("**Variable** - `%s`\n\n", target.Ident[0]),
		}
	default:
		log.Debug().Msgf("Hover on unknown node type: %T", target)
	}

	// Analyze position
	// Build hover content

	return
	// &protocol.Hover{
	// 	Contents: protocol.MarkupContent{
	// 		Kind:  protocol.MarkupKindMarkdown,
	// 		Value: "Not Implemented Yet",
	// 	},
	// 	Range: &protocol.Range{
	// 		Start: protocol.Position{Line: 0, Character: 0},
	// 		End:   protocol.Position{Line: 0, Character: 3},
	// 	},
	// }, nil
}
