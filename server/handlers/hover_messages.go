// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"fmt"
	"strings"
	parse "text-template-parser"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var nodeMessage = map[parse.NodeType]string{
	parse.NodeAction:     "**Action** - `{{ %s }}`\n\nAction running a command or pipeline.",
	parse.NodeIf:         "**If Branch** - `{{ if  }}`\n\n",
	parse.NodeRange:      "**Range Branch** - `{{ range %s }}`\n\nBranch executed for each item in a collection.",
	parse.NodeWith:       "**With Branch** - `{{ with %s }}`\n\nBranch executed with a new context.",
	parse.NodeCommand:    "**Command** - `%s`\n\nA command within an action.",
	parse.NodeDot:        "**Cursor** - `.`\n\nReturns the current context",
	parse.NodeField:      "**Field Access** - `%s`\n\nAccesses the `%s` field of the `.%s` context.",
	parse.NodeIdentifier: "**Identifier** - `%s`\n\nRepresents an identifier in a command or action.",
	parse.NodePipe:       "**Pipeline** - `%s`\n\nA sequence of commands connected by `|`.",
	parse.NodeVariable:   "**Variable** - `%s`\n\n",
	parse.NodeText:       "**Text** - `%.15s`\n\nPlain text content.",
	parse.NodeTemplate:   "**Template** - `%s`\n\nDefines a template named `%s`.",
	parse.NodeList:       "***Document root*** - \n\n Root node of the parse tree, containing all other nodes.",
	parse.NodeBool:       "**Boolean literal** - `%v`\n\nA literal value.",
	parse.NodeNumber:     "**Number literal** - `%s`\n\nA literal numeric value.",
	parse.NodeString:     "**String literal** - `%s`\n\nA literal string value.",
	parse.NodeNil:        "**Nil literal** - `nil`\n\nRepresents a nil value.",
	parse.NodeUndefined:  "**Undefined** - \n\nError during parsing, node type is undefined: %s",
}

var specialMessages = map[string]string{
	"and":   "**And Function** - `and`\n\nA built-in function that returns the first argument if it is false, and the last argument otherwise.",
	"index": "**Index variable** - `%s`\n\n serves as the index variable in the range loop, representing the current iteration count.",
	"len":   "**Len Function** - `len`\n\nA built-in function that returns the length of its argument.",
	"not":   "**Not Function** - `not`\n\nA built-in function that returns the boolean negation of its argument.",
	"or":    "**Or Function** - `or`\n\nA built-in function that returns the first argument if it is true, and the last argument otherwise.",
	"end":   "**End Tag** - \n\nMarks the end of %s, which started at line %d.",
}

// MessageEnd generates a hover message for an end tag of a BranchNode, including the branch type and line number where the branch starts.
// TODO: should hyperlink the line number to the start tag of the branch
func MessageEnd(n parse.Node, pos protocol.Position) string {
	switch node := n.(type) {
	case *parse.IfNode:
		return fmt.Sprintf(specialMessages["end"], "if", pos.Line)
	case *parse.RangeNode:
		return fmt.Sprintf(specialMessages["end"], "range", pos.Line)
	case *parse.WithNode:
		return fmt.Sprintf(specialMessages["end"], "with", pos.Line)
	case *parse.TemplateNode:
		return fmt.Sprintf(specialMessages["end"], "template "+node.Name, node.Line)
	default:
		return ""
	}
}

// MessageAction generates a hover message for an ActionNode, including the full action string.
func MessageAction(n *parse.ActionNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeAction], n.Pipe.String())
}

// MessageBranch generates a hover message for a BranchNode, including the branch type and relevant pipeline information.
func MessageBranch(n *parse.BranchNode) string {
	switch n.NodeType {
	case parse.NodeIf:
		return fmt.Sprintf(nodeMessage[parse.NodeIf], n.Pipe.String())
	case parse.NodeRange:
		return fmt.Sprintf(nodeMessage[parse.NodeRange], n.Pipe.String())
	case parse.NodeWith:
		return fmt.Sprintf(nodeMessage[parse.NodeWith], n.Pipe.String())
	default:
		return ""
	}
}

// MessageCommand generates a hover message for a CommandNode, including the full command string.
func MessageCommand(n *parse.CommandNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeCommand], n.String())
}

// MessageDot generates a hover message for a DotNode.
func MessageDot(_ *parse.DotNode) string {
	return nodeMessage[parse.NodeDot]
}

// MessageField generates a hover message for a FieldNode, including the field name and context.
func MessageField(n *parse.FieldNode) string {
	ctx := ""
	if len(n.Ident) > 1 {
		ctx = strings.Join(n.Ident[1:], ".")
	}
	field := ""
	if len(n.Ident) > 0 {
		field = n.Ident[0]
	}
	return fmt.Sprintf(nodeMessage[parse.NodeField], n.String(), field, ctx)
}

// MessageIdentifier generates a hover message for an IdentifierNode, including the identifier name.
func MessageIdentifier(n *parse.IdentifierNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeIdentifier], n.Ident)
}

// MessagePipe generates a hover message for a PipeNode, including the full pipeline string.
func MessagePipe(n *parse.PipeNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodePipe], n.String())
}

// MessageIndexVariable generates a hover message for a VariableNode that serves as an index variable in a range loop, including the variable name.
func MessageIndexVariable(n *parse.VariableNode) string {
	ident := ""
	if len(n.Ident) > 0 {
		ident = n.Ident[0]
	}
	return fmt.Sprintf(specialMessages["index"], ident)
}

// MessageVariable generates a hover message for a VariableNode, including the variable name.
func MessageVariable(n *parse.VariableNode) string {
	ident := ""
	if len(n.Ident) > 0 {
		ident = n.Ident[0]
	}
	return fmt.Sprintf(nodeMessage[parse.NodeVariable], ident)
}

// MessageBool generates a hover message for a BoolNode, including the boolean value.
func MessageBool(n *parse.BoolNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeBool], n.True)
}

// MessageText generates a hover message for a TextNode, including a truncated version of the text content.
func MessageText(n *parse.TextNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeText], n.Text)
}

// MessageTemplate generates a hover message for a TemplateNode, including the template name.
func MessageTemplate(n *parse.TemplateNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeTemplate], n.Name, n.Name)
}

// MessageList generates a hover message for a ListNode, including the number of nodes in the list.
func MessageList(n *parse.ListNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeList], len(n.Nodes))
}

// MessageNumber generates a hover message for a NumberNode, including the numeric value.
func MessageNumber(n *parse.NumberNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeNumber], n.Text)
}

// MessageString generates a hover message for a StringNode, including the string value.
func MessageString(n *parse.StringNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeString], n.Text)
}

// MessageNil generates a hover message for a NilNode, which represents a nil value in Go templates.
func MessageNil(_ *parse.NilNode) string {
	return nodeMessage[parse.NodeNil]
}

// MessageUndefined generates a hover message for an undefined node type, including the node type information.
func MessageUndefined(n *parse.UndefinedNode) string {
	return fmt.Sprintf(nodeMessage[parse.NodeUndefined], n.String())
}
