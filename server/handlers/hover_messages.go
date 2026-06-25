// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"fmt"
	"go/types"
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
	// Empty interface (any) means the type is unknown
	if iface, ok := t.Underlying().(*types.Interface); ok && iface.NumMethods() == 0 {
		return ""
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
	const fieldMessageCtx = "```go\nfield %s\n```\nAccesses a field from the `%s` dot context."
	const fieldMessageTypedCtx = "```go\nfield %s %s\n```\nAccesses a field from the `%s` dot context."

	ctxTypeName := formatType(n.DotType())
	fieldTypeName := formatType(typ)

	if ctxTypeName != "" {
		if fieldTypeName != "" {
			return withLink(
				fmt.Sprintf(fieldMessageTypedCtx, n.String(), fieldTypeName, ctxTypeName),
			)
		}
		return withLink(fmt.Sprintf(fieldMessageCtx, n.String(), ctxTypeName))
	}
	return ""
}

// MessageIdentifier generates a hover message for an IdentifierNode, including the identifier name and (when known) the resolved Go type.
func MessageIdentifier(n *serverTypes.IdentifierNode, typ types.Type) string {
	const identifierMessage = "```go\n%s\n```\nRepresents an identifier in a command or action."
	const identifierMessageTyped = "```go\n%s %s\n```\nFunction call with return type."

	// TODO: add more special messages
	specialMessages := map[string]string{
		// Logical
		"and": "```go\nand(arg0 any, args ...any) any\n```\nReturns the first empty argument, or the last argument if none are empty. Short-circuits; result is one of the args (not a bool).",
		"or":  "```go\nor(arg0 any, args ...any) any\n```\nReturns the first non-empty argument, or the last argument if all are empty. Short-circuits; result is one of the args (not a bool).",
		"not": "```go\nnot(arg any) bool\n```\nReturns the boolean negation of its argument. `true` when empty (false / 0 / nil / zero-length), `false` otherwise.",

		// Length / indexing
		"len":   "```go\nlen(item any) int\n```\nReturns the integer length of a string, array, slice, map, or channel.",
		"index": "```go\nindex(item any, indices ...any) any\n```\nReturns the result of indexing into `item` with the given keys, e.g. `index x 1 2 3` is `x[1][2][3]`.",
		"slice": "```go\nslice(item any, indices ...any) any\n```\nSlices a string, array, slice, or pointer-to-array. With 1-3 indices, behaves like `item[i:]`, `item[i:j]`, or `item[i:j:k]`.",

		// Formatting
		"print":   "```go\nprint(args ...any) string\n```\nFormats its arguments using default formats and returns the resulting string. Spaces are added between operands when neither is a string.",
		"printf":  "```go\nprintf(format string, args ...any) string\n```\nFormats its arguments according to the format string and returns the resulting string.",
		"println": "```go\nprintln(args ...any) string\n```\nFormats its arguments using default formats and returns the resulting string. Spaces are always added between operands and a newline is appended.",

		// Escapers
		"html":     "```go\nhtml(args ...any) string\n```\nReturns the escaped HTML equivalent of the textual representation of its arguments.",
		"js":       "```go\njs(args ...any) string\n```\nReturns the escaped JavaScript equivalent of the textual representation of its arguments.",
		"urlquery": "```go\nurlquery(args ...any) string\n```\nReturns the escaped value of the textual representation of its arguments, suitable for embedding in a URL query.",

		// Comparison (return bool; first arg's kind determines the rest)
		"eq": "```go\neq(arg1 any, arg2 ...any) bool\n```\nReports whether `arg1 == argN` for any of the provided arguments.",
		"ne": "```go\nne(arg1, arg2 any) bool\n```\nReports whether `arg1 != arg2`.",
		"lt": "```go\nlt(arg1, arg2 any) bool\n```\nReports whether `arg1 <  arg2`.",
		"le": "```go\nle(arg1, arg2 any) bool\n```\nReports whether `arg1 <= arg2`.",
		"gt": "```go\ngt(arg1, arg2 any) bool\n```\nReports whether `arg1 >  arg2`.",
		"ge": "```go\nge(arg1, arg2 any) bool\n```\nReports whether `arg1 >= arg2`.",

		// Invocation
		"call": "```go\ncall(fn any, args ...any) any\n```\nCalls `fn` (a function value) with the given args. The function must return either one value, or one value plus an error.",
	}

	if msg, ok := specialMessages[n.Ident]; ok {
		return withLink(msg)
	}

	typStr := formatType(typ)
	if typStr != "" {
		return withLink(fmt.Sprintf(identifierMessageTyped, n.Ident, typStr))
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

	ident := n.String()
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
