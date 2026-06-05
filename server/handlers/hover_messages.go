// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"fmt"
	"go/types"
	"strings"
	parse "text-template-parser"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// MessageElse generates a hover message for an else tag of a BranchNode, including the branch type and line number where the if statement starts.
func MessageElse(n *parse.Node, pos protocol.Position) string {
	const withBranch = "```go\nelse\n```\nFrom `%s` at line %d."
	const withoutBranch = "```go\nelse\n```\nUnknown start."

	switch (*n).(type) {
	case *parse.IfNode:
		return withLink(fmt.Sprintf(withBranch, "if", pos.Line+1))
	case *parse.RangeNode:
		return withLink(fmt.Sprintf(withBranch, "range", pos.Line+1))
	case *parse.WithNode:
		return withLink(fmt.Sprintf(withBranch, "with", pos.Line+1))
	default:
		return withLink(withoutBranch)
	}
}

// MessageEnd generates a hover message for an end tag of a BranchNode, including the branch type and line number where the branch starts.
// TODO: should hyperlink the line number to the start tag of the branch
func MessageEnd(n parse.Node, pos protocol.Position) string {
	const withBranch = "```go\nend\n```\nFrom `%s` at line %d."
	const withoutBranch = "```go\nelse\n```\nFrom `unknown`."

	switch node := n.(type) {
	case *parse.IfNode:
		return fmt.Sprintf(withBranch, "if", pos.Line+1)
	case *parse.RangeNode:
		return fmt.Sprintf(withBranch, "range", pos.Line+1)
	case *parse.WithNode:
		return fmt.Sprintf(withBranch, "with", pos.Line+1)
	case *parse.TemplateNode:
		return fmt.Sprintf(withBranch, "template "+node.Name, node.Line+1)
	default:
		return withoutBranch
	}
}

// MessageBranch generates a hover message for a BranchNode, including the branch type and relevant pipeline information.
func MessageBranch(n *parse.BranchNode) string {
	const ifMessage = "```go\nif %s\n```\nIf the value of the pipeline is empty, no output is generated; Otherwise, inside is executed."
	const rangeMessage = "```go\nrange %s\n```\nBranch executed for each item in a collection."
	const withMessage = "```go\nwith %s\n```\nBranch executed with a new context."

	switch n.NodeType {
	case parse.NodeIf:
		return withLink(fmt.Sprintf(ifMessage, n.Pipe.String()))
	case parse.NodeRange:
		return withLink(fmt.Sprintf(rangeMessage, n.Pipe.String()))
	case parse.NodeWith:
		return withLink(fmt.Sprintf(withMessage, n.Pipe.String()))
	}

	// this should be unreachable
	return ""
}

// MessageDot generates a hover message for a DotNode.
func MessageDot(_ *parse.DotNode) string {
	const dotMessage = "```go\ndot\n```\nReturns the current context."

	return withLink(dotMessage)
}

// MessageField generates a hover message for a FieldNode, including the field name and context.
func MessageField(n *parse.FieldNode) string {
	const fieldMessage = "```go\nfield %s\n```\nAccesses the `%s` field of the `.%s` context."

	ctx := ""
	if len(n.Ident) > 1 {
		ctx = strings.Join(n.Ident[1:], ".")
	}
	field := ""
	if len(n.Ident) > 0 {
		field = n.Ident[0]
	}
	return withLink(fmt.Sprintf(fieldMessage, n.String(), field, ctx))
}

// MessageIdentifier generates a hover message for an IdentifierNode, including the identifier name.
func MessageIdentifier(n *parse.IdentifierNode) string {
	const identifierMessage = "```go\n%s\n```\nRepresents an identifier in a command or action."

	// TODO: add more special messages
	specialMessages := map[string]string{
		"and": "```go\nand\n```\nA built-in function that returns the first argument if it is false, and the last argument otherwise.",
		"len": "```go\nlen\n```\nA built-in function that returns the length of its argument.",
		"not": "```go\nnot\n```\nA built-in function that returns the boolean negation of its argument.",
		"or":  "```go\nor\n```\nA built-in function that returns the first argument if it is true, and the last argument otherwise.",
	}

	if msg, ok := specialMessages[n.Ident]; ok {
		return withLink(msg)
	}
	return withLink(fmt.Sprintf(identifierMessage, n.Ident))
}

// MessageIndexVariable generates a hover message for a VariableNode that serves as an index variable in a range loop, including the variable name.
func MessageIndexVariable(n *parse.VariableNode) string {
	const indexMessage = "```go\nvar %s int\n```\nServes as the index variable in the `range` loop, representing the current iteration count."

	ident := ""
	if len(n.Ident) > 0 {
		ident = n.Ident[0]
	}
	return fmt.Sprintf(indexMessage, ident)
}

// MessageVariable generates a hover message for a VariableNode, including the variable name.
func MessageVariable(n *parse.VariableNode, varValue any, typ types.Type) string {
	const withValue = "```go\nvar %s %T = %v\n```"
	const withType = "```go\nvar %s %v\n```"
	const unknownType = "```go\nvar %s (unknown)\n```"

	ident := ""
	if len(n.Ident) > 0 {
		ident = n.Ident[0]
	}

	if varValue != nil {
		return fmt.Sprintf(withValue, ident, varValue, varValue)
	}
	if typ != nil {
		return fmt.Sprintf(withType, ident, typ)
	}

	return fmt.Sprintf(unknownType, ident)
}

// Because both `block` and `template` parse to TemplateNode, it's impossible to distinguish them
// the information that would be provided is not that informative, so it's easier to not provide any

// // MessageTemplate generates a hover message for a TemplateNode, including the template name.
// func MessageTemplate(n *parse.TemplateNode) string {
// 	const templateMessage = "```go\ntemplate \"%s\"\n```\nExecutes a template named `%s` with the pipelined data."

// 	return withLink(fmt.Sprintf(templateMessage, n.Name, n.Name))
// }

// MessageNil generates a hover message for a NilNode, which represents a nil value in Go templates.
func MessageNil(_ *parse.NilNode) string {
	return "```go\nvar nil\n```\nnil is a predeclared identifier representing the zero value for a pointer, channel, func, interface, map, or slice type."
}

func withLink(str string) string {
	return str + "\n***\n[text/template reference](https://pkg.go.dev/text/template)"
}
