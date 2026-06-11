package utils

import (
	"testing"
	parse "text-template-parser"
	serverTypes "text-template-server/types"
)

type nilParseNodeTest struct {
	name     string
	node     parse.Node
	expected bool
}

type nilTypeNodeTest struct {
	name     string
	node     serverTypes.Node
	expected bool
}

var nilParseNodeTests = []nilParseNodeTest{
	{name: "nil ListNode", node: (*parse.ListNode)(nil), expected: true},
	{name: "non-nil ListNode", node: &parse.ListNode{}, expected: false},
	{name: "nil ActionNode", node: (*parse.ActionNode)(nil), expected: true},
	{name: "non-nil ActionNode", node: &parse.ActionNode{}, expected: false},
	{name: "nil PipeNode", node: (*parse.PipeNode)(nil), expected: true},
	{name: "non-nil PipeNode", node: &parse.PipeNode{}, expected: false},
	{name: "nil CommandNode", node: (*parse.CommandNode)(nil), expected: true},
	{name: "non-nil CommandNode", node: &parse.CommandNode{}, expected: false},
	{name: "nil ChainNode", node: (*parse.ChainNode)(nil), expected: true},
	{name: "non-nil ChainNode", node: &parse.ChainNode{}, expected: false},
	{name: "nil IfNode", node: (*parse.IfNode)(nil), expected: true},
	{name: "non-nil IfNode", node: &parse.IfNode{}, expected: false},
	{name: "nil RangeNode", node: (*parse.RangeNode)(nil), expected: true},
	{name: "non-nil RangeNode", node: &parse.RangeNode{}, expected: false},
	{name: "nil WithNode", node: (*parse.WithNode)(nil), expected: true},
	{name: "non-nil WithNode", node: &parse.WithNode{}, expected: false},
	{name: "nil TemplateNode", node: (*parse.TemplateNode)(nil), expected: true},
	{name: "non-nil TemplateNode", node: &parse.TemplateNode{}, expected: false},
	{name: "nil VariableNode", node: (*parse.VariableNode)(nil), expected: true},
	{name: "non-nil VariableNode", node: &parse.VariableNode{}, expected: false},
	{name: "nil FieldNode", node: (*parse.FieldNode)(nil), expected: true},
	{name: "non-nil FieldNode", node: &parse.FieldNode{}, expected: false},
	{name: "nil IdentifierNode", node: (*parse.IdentifierNode)(nil), expected: true},
	{name: "non-nil IdentifierNode", node: &parse.IdentifierNode{}, expected: false},
	{name: "nil UndefinedNode", node: (*parse.UndefinedNode)(nil), expected: true},
	{name: "non-nil UndefinedNode", node: &parse.UndefinedNode{}, expected: false},
	{name: "nil CommentNode", node: (*parse.CommentNode)(nil), expected: true},
	{name: "non-nil CommentNode", node: &parse.CommentNode{}, expected: false},
	{name: "nil BreakNode", node: (*parse.BreakNode)(nil), expected: true},
	{name: "non-nil BreakNode", node: &parse.BreakNode{}, expected: false},
	{name: "nil ContinueNode", node: (*parse.ContinueNode)(nil), expected: true},
	{name: "non-nil ContinueNode", node: &parse.ContinueNode{}, expected: false},
}

var nilTypeNodeTests = []nilTypeNodeTest{
	{name: "nil ActionNode", node: (*serverTypes.ActionNode)(nil), expected: true},
	{name: "non-nil ActionNode", node: &serverTypes.ActionNode{}, expected: false},
	{name: "nil IfNode", node: (*serverTypes.IfNode)(nil), expected: true},
	{name: "non-nil IfNode", node: &serverTypes.IfNode{}, expected: false},
	{name: "nil RangeNode", node: (*serverTypes.RangeNode)(nil), expected: true},
	{name: "non-nil RangeNode", node: &serverTypes.RangeNode{}, expected: false},
	{name: "nil WithNode", node: (*serverTypes.WithNode)(nil), expected: true},
	{name: "non-nil WithNode", node: &serverTypes.WithNode{}, expected: false},
	{name: "nil TemplateNode", node: (*serverTypes.TemplateNode)(nil), expected: true},
	{name: "non-nil TemplateNode", node: &serverTypes.TemplateNode{}, expected: false},
	{name: "nil VariableNode", node: (*serverTypes.VariableNode)(nil), expected: true},
	{name: "non-nil VariableNode", node: &serverTypes.VariableNode{}, expected: false},
	{name: "nil FieldNode", node: (*serverTypes.FieldNode)(nil), expected: true},
	{name: "non-nil FieldNode", node: &serverTypes.FieldNode{}, expected: false},
	{name: "nil IdentifierNode", node: (*serverTypes.IdentifierNode)(nil), expected: true},
	{name: "non-nil IdentifierNode", node: &serverTypes.IdentifierNode{}, expected: false},
	{name: "nil CommentNode", node: (*serverTypes.CommentNode)(nil), expected: true},
	{name: "non-nil CommentNode", node: &serverTypes.CommentNode{}, expected: false},
	{name: "nil BreakNode", node: (*serverTypes.BreakNode)(nil), expected: true},
	{name: "non-nil BreakNode", node: &serverTypes.BreakNode{}, expected: false},
	{name: "nil ContinueNode", node: (*serverTypes.ContinueNode)(nil), expected: true},
	{name: "non-nil ContinueNode", node: &serverTypes.ContinueNode{}, expected: false},
}

func TestNilParseNode(t *testing.T) {
	for _, tc := range nilParseNodeTests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNilNode(tc.node); got != tc.expected {
				t.Errorf("isNilParseNode(%T) = %v, want %v", tc.node, got, tc.expected)
			}
		})
	}
}

func TestNilTypeNode(t *testing.T) {
	for _, tc := range nilTypeNodeTests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNilTypeNode(tc.node); got != tc.expected {
				t.Errorf("isNilTypeNode(%T) = %v, want %v", tc.node, got, tc.expected)
			}
		})
	}
}
