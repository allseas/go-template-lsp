// Package utils implements various utils for the server
package utils

import (
	parse "text-template-parser"
	serverTypes "text-template-server/types"
)

// IsNilNode checks if a parse.Node is nil or contains nil pointers in its concrete type.
func IsNilNode(n parse.Node) bool {
	if n == nil {
		return true
	}

	switch node := n.(type) {
	case *parse.ListNode:
		return node == nil
	case *parse.ActionNode:
		return node == nil
	case *parse.PipeNode:
		return node == nil
	case *parse.CommandNode:
		return node == nil
	case *parse.ChainNode:
		return node == nil
	case *parse.IfNode:
		return node == nil
	case *parse.RangeNode:
		return node == nil
	case *parse.WithNode:
		return node == nil
	case *parse.TemplateNode:
		return node == nil
	case *parse.VariableNode:
		return node == nil
	case *parse.FieldNode:
		return node == nil
	case *parse.IdentifierNode:
		return node == nil
	case *parse.UndefinedNode:
		return node == nil
	case *parse.CommentNode:
		return node == nil
	case *parse.BreakNode:
		return node == nil
	case *parse.ContinueNode:
		return node == nil
	default:
		return false
	}
}

// IsNilTypeNode checks if a serverTypes.Node is nil or contains nil pointers in its concrete type.
func IsNilTypeNode(n serverTypes.Node) bool {
	if n == nil {
		return true
	}

	switch node := n.(type) {
	case *serverTypes.ActionNode:
		return node == nil
	case *serverTypes.PipeNode:
		return node == nil
	case *serverTypes.CommandNode:
		return node == nil
	case *serverTypes.ChainNode:
		return node == nil
	case *serverTypes.IfNode:
		return node == nil
	case *serverTypes.RangeNode:
		return node == nil
	case *serverTypes.WithNode:
		return node == nil
	case *serverTypes.TemplateNode:
		return node == nil
	case *serverTypes.VariableNode:
		return node == nil
	case *serverTypes.FieldNode:
		return node == nil
	case *serverTypes.IdentifierNode:
		return node == nil
	case *serverTypes.UndefinedNode:
		return node == nil
	case *serverTypes.CommentNode:
		return node == nil
	case *serverTypes.BreakNode:
		return node == nil
	case *serverTypes.ContinueNode:
		return node == nil
	default:
		return false
	}
}
