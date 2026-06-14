// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"fmt"
	"go/types"
	"strings"
	serverTypes "text-template-server/types"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// formatType renders a Go type using the package's short name (e.g.
// `model.Address`) rather than the full import path. Untyped nil renders as
// just `nil`. Returns "" if t is nil.
func formatType(t types.Type) string {
	if t == nil {
		return ""
	}
	if b, ok := t.(*types.Basic); ok && b.Kind() == types.UntypedNil {
		return "nil"
	}
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Name()
	})
}

// MessageElse generates a hover message for an else tag of a BranchNode, including the branch type and line number where the if statement starts.
func MessageElse(n *serverTypes.Node, pos protocol.Position) string {
	const withBranch = "```go\nelse\n```\nFrom `%s` at line %d."
	const withoutBranch = "```go\nelse\n```\nUnknown start."

	switch (*n).(type) {
	case *serverTypes.IfNode:
		return withLink(fmt.Sprintf(withBranch, "if", pos.Line+1))
	case *serverTypes.RangeNode:
		return withLink(fmt.Sprintf(withBranch, "range", pos.Line+1))
	case *serverTypes.WithNode:
		return withLink(fmt.Sprintf(withBranch, "with", pos.Line+1))
	default:
		return withLink(withoutBranch)
	}
}

// MessageEnd generates a hover message for an end tag of a BranchNode, including the branch type and line number where the branch starts.
// TODO: should hyperlink the line number to the start tag of the branch
func MessageEnd(n serverTypes.Node, pos protocol.Position) string {
	const withBranch = "```go\nend\n```\nFrom `%s` at line %d."
	const withoutBranch = "```go\nelse\n```\nFrom `unknown`."

	switch node := n.(type) {
	case *serverTypes.IfNode:
		return fmt.Sprintf(withBranch, "if", pos.Line+1)
	case *serverTypes.RangeNode:
		return fmt.Sprintf(withBranch, "range", pos.Line+1)
	case *serverTypes.WithNode:
		return fmt.Sprintf(withBranch, "with", pos.Line+1)
	case *serverTypes.TemplateNode:
		return fmt.Sprintf(withBranch, "template "+node.Name, node.Line+1)
	default:
		return withoutBranch
	}
}

// MessageBranch generates a hover message for a BranchNode, including the branch type and relevant pipeline information.
func MessageBranch(n *serverTypes.BranchNode) string {
	const ifMessage = "```go\nif %s\n```\nIf the value of the pipeline is empty, no output is generated; Otherwise, inside is executed."
	const rangeMessage = "```go\nrange %s\n```\nBranch executed for each item in a collection."
	const withMessage = "```go\nwith %s\n```\nBranch executed with a new context."

	switch n.NodeType {
	case serverTypes.NodeIf:
		return withLink(fmt.Sprintf(ifMessage, n.Pipe.String()))
	case serverTypes.NodeRange:
		return withLink(fmt.Sprintf(rangeMessage, n.Pipe.String()))
	case serverTypes.NodeWith:
		return withLink(fmt.Sprintf(withMessage, n.Pipe.String()))
	}

	// this should be unreachable
	return ""
}

// MessageDot generates a hover message for a DotNode. If typ is non-nil the
// Go type of the current context is included.
func MessageDot(_ *serverTypes.DotNode, typ types.Type) string {
	const dotMessage = "```go\ndot\n```\nReturns the current context."
	const dotMessageTyped = "```go\ndot %s\n```\nReturns the current context."

	if s := formatType(typ); s != "" {
		return withLink(fmt.Sprintf(dotMessageTyped, s))
	}
	return withLink(dotMessage)
}

// MessageField generates a hover message for a FieldNode, including the field
// name and (when known) the resolved Go type of the full field chain.
func MessageField(n *serverTypes.FieldNode, typ types.Type) string {
	const fieldMessage = "```go\nfield %s\n```\nAccesses the `%s` field of the `.%s` context."
	const fieldMessageTyped = "```go\nfield %s %s\n```\nAccesses the `%s` field of the `.%s` context."

	ctx := ""
	if len(n.Ident) > 1 {
		ctx = strings.Join(n.Ident[1:], ".")
	}
	field := ""
	if len(n.Ident) > 0 {
		field = n.Ident[0]
	}
	if s := formatType(typ); s != "" {
		return withLink(fmt.Sprintf(fieldMessageTyped, n.String(), s, field, ctx))
	}
	return withLink(fmt.Sprintf(fieldMessage, n.String(), field, ctx))
}

// MessageIdentifier generates a hover message for an IdentifierNode, including the identifier name.
func MessageIdentifier(n *serverTypes.IdentifierNode) string {
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
func MessageIndexVariable(n *serverTypes.VariableNode) string {
	const indexMessage = "```go\nvar %s int\n```\nServes as the index variable in the `range` loop, representing the current iteration count."

	ident := ""
	if len(n.Ident) > 0 {
		ident = n.Ident[0]
	}
	return fmt.Sprintf(indexMessage, ident)
}

// MessageVariable generates a hover message for a VariableNode, including the
// variable name and (when known) its resolved Go type and/or constant value.
func MessageVariable(n *serverTypes.VariableNode, varValue any, typ types.Type) string {
	const withValueAndType = "```go\nvar %s %s = %v\n```"
	const withValue = "```go\nvar %s = %v\n```"
	const withType = "```go\nvar %s %s\n```"
	const unknownType = "```go\nvar %s (unknown)\n```"

	ident := ""
	if len(n.Ident) > 0 {
		ident = n.Ident[0]
	}
	typStr := formatType(typ)
	switch {
	case varValue != nil && typStr != "":
		return fmt.Sprintf(withValueAndType, ident, typStr, varValue)
	case varValue != nil:
		return fmt.Sprintf(withValue, ident, varValue)
	case typStr != "":
		return fmt.Sprintf(withType, ident, typStr)
	default:
		return fmt.Sprintf(unknownType, ident)
	}
}

// Because both `block` and `template` parse to TemplateNode, it's impossible to distinguish them
// the information that would be provided is not that informative, so it's easier to not provide any

// // MessageTemplate generates a hover message for a TemplateNode, including the template name.
// func MessageTemplate(n *serverTypes.TemplateNode) string {
// 	const templateMessage = "```go\ntemplate \"%s\"\n```\nExecutes a template named `%s` with the pipelined data."

// 	return withLink(fmt.Sprintf(templateMessage, n.Name, n.Name))
// }

// MessageNil generates a hover message for a NilNode, which represents a nil value in Go templates.
func MessageNil(_ *serverTypes.NilNode) string {
	return "```go\nvar nil\n```\nnil is a predeclared identifier representing the zero value for a pointer, channel, func, interface, map, or slice type."
}

func withLink(str string) string {
	return str + "\n***\n[text/template reference](https://pkg.go.dev/text/template)"
}
