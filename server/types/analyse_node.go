//go:build !allseas

package types

import (
	"fmt"
	parse "text-template-parser"
)

// analyseNode converts a parse Node to a typed Node.
func analyseNode(node parse.Node, parent Node, ctx *analysisCtx) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *parse.ListNode:
		return analyseList(n, parent, ctx)

	case *parse.TextNode:
		return analyseText(n, parent)

	case *parse.ActionNode:

		return analyseAction(n, parent, ctx)

	case *parse.CommandNode:
		return analyseCommand(n, parent, ctx)

	case *parse.FieldNode:
		return analyseField(n, parent, ctx)

	case *parse.VariableNode:
		return analyseVariable(n, parent, ctx)

	case *parse.IdentifierNode:
		return analyseIdentifier(n, parent, ctx)

	case *parse.ChainNode:
		return analyseChain(n, parent, ctx)

	case *parse.DotNode:
		return analyseDot(n, parent, ctx)

	case *parse.NilNode:
		return analyseNil(n, parent, ctx)

	case *parse.BoolNode:
		return analyseBool(n, parent, ctx)

	case *parse.NumberNode:
		return analyseNumber(n, parent, ctx)

	case *parse.StringNode:
		return analyseString(n, parent, ctx)

	case *parse.CommentNode:
		return analyseComment(n, parent, ctx)

	case *parse.IfNode:
		return analyseIf(n, parent, ctx)

	case *parse.RangeNode:
		return analyseRange(n, parent, ctx)

	case *parse.WithNode:
		return analyseWith(n, parent, ctx)

	case *parse.TemplateNode:
		return analyseTemplate(n, parent, ctx)

	case *parse.BreakNode:
		return analyseBreak(n, parent, ctx)

	case *parse.ContinueNode:
		return analyseContinue(n, parent, ctx)
	case *parse.PipeNode:
		return analysePipe(n, parent, ctx)
	case *parse.UndefinedNode:
		return analyseUndefined(n, parent)
	default:
		panic(fmt.Sprintf("unknown node type: %T", node))
	}
}
