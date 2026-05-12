// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"errors"
	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func hover(_ *glsp.Context, params *protocol.HoverParams) (hover *protocol.Hover, err error) {
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

	switch target := target.(type) {
	case *parse.ActionNode:
		log.Debug().Msg("Hover on ActionNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageAction(target),
		}
	case *parse.BranchNode:
		{
			log.Debug().Msg("Hover on BranchNode")
			hover.Contents = protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageBranch(target),
			}
		}
	case *parse.CommandNode:
		log.Debug().Msg("Hover on CommandNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageCommand(target),
		}
	case *parse.DotNode:
		log.Debug().Msg("Hover on DotNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageDot(target),
		}
	case *parse.FieldNode:
		log.Debug().Msg("Hover on FieldNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageField(target),
		}
	case *parse.IdentifierNode:
		log.Debug().Msg("Hover on IdentifierNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageIdentifier(target),
		}
	case *parse.PipeNode:
		log.Debug().Msg("Hover on PipeNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessagePipe(target),
		}
	case *parse.VariableNode:
		log.Debug().Msg("Hover on VariableNode")

		if isIndexVariable(target, doc.tree.Root) {
			hover.Contents = protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIndexVariable(target),
			}
		} else {
			hover.Contents = protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageVariable(target),
			}
		}
	case *parse.TextNode:
		log.Debug().Msg("Hover on TextNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageText(target),
		}
	case *parse.TemplateNode:
		log.Debug().Msg("Hover on TemplateNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageTemplate(target),
		}
	case *parse.ListNode:
		log.Debug().Msg("Hover on ListNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageList(target),
		}
	case *parse.BoolNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageBool(target),
		}
	case *parse.NumberNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageNumber(target),
		}
	case *parse.StringNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageString(target),
		}
	case *parse.NilNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageNil(target),
		}
	case *parse.IfNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageBranch(&target.BranchNode),
		}
	case *parse.RangeNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageBranch(&target.BranchNode),
		}
	case *parse.WithNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageBranch(&target.BranchNode),
		}
	default:
		log.Debug().Msgf("Hover on unknown node type: %T", target)
	}

	// Analyze position
	// Build hover content

	return
}

func isIndexVariable(target *parse.VariableNode, root *parse.ListNode) bool {
	ctx := &Context{}
	buildPath(root, target, ctx)

	path := ctx.Path
	branch := path[len(path)-3] // branch is the second to last element in the path
	if _, ok := branch.(*parse.BranchNode); !ok {
		return false
	}
	branchNode := branch.(*parse.BranchNode)
	if branchNode.NodeType != parse.NodeRange {
		return false
	}
	pipe := branchNode.Pipe

	return pipe.Decl[1] == target

}
