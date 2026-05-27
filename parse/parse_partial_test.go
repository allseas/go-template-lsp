// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse builds parse trees for templates as defined by text/template
// and html/template. Clients should use those packages to construct templates
// rather than this one, which provides shared internal data structures not
// intended for general use.
package parse

import (
	"errors"
	"strings"
	"testing"
)

type robustTreeTest struct {
	name    string
	input   string
	result  *Tree
	ok      bool
	message string
}

var robustTreeTests = []robustTreeTest{
	{
		name:  "unclosed action",
		input: "{{",
		result: &Tree{
			Root: &ListNode{
				NodeType: NodeList,
				Pos:      0,
				Nodes: []Node{
					&UndefinedNode{
						NodeType: NodeUndefined,
						Pos:      trimMarkerLen,
						Err:      errors.New("unclosed action"),
						str:      "{{",
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1: unexpected token in command: unclosed action"),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "empty action",
		input: "{{}}",
		result: &Tree{
			Root: &ListNode{
				NodeType: NodeList,
				Pos:      0,
				Nodes: []Node{

					&ActionNode{
						NodeType: NodeAction,
						Pos:      0,
						Pipe: &PipeNode{
							NodeType: NodePipe,
							Pos:      0,
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      2,
											Err:      errors.New("template: robust:1: missing value for command"),
											str:      "",
										},
									},
								},
							},
						},
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1: missing value for command"),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "unclosed action with command",
		input: "m {{ 3 {{3}} t",
		result: &Tree{
			Root: &ListNode{
				NodeType: NodeList,
				Pos:      0,
				Nodes: []Node{
					&TextNode{
						NodeType: NodeText,
						Pos:      0,
						Text:     []byte("m "),
					},
					&UndefinedNode{
						NodeType: NodeUndefined,
						Pos:      2,
						Err:      errors.New("template: robust:1: unexpected token in action: \"{{\""),
						str:      "\"3\"\"{{\"",
					},
					&ActionNode{
						NodeType: NodeAction,
						Pos:      7,
						Pipe: &PipeNode{
							NodeType: NodePipe,
							Pos:      9,
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&NumberNode{
											NodeType: NodeNumber,
											Pos:      10,
											Text:     "3",
										},
									},
									Pos: 9,
								},
							},
						},
					},
					&TextNode{
						NodeType: NodeText,
						Pos:      15,
						Text:     []byte(" t"),
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1: unexpected token in command: unclosed action with command"),
			},
		},
		ok:      true,
		message: "",
	},
	// {
	// 	name:  "unclosed action with command and pipe",
	// 	input: "m {{ 3 |",
	// },
	// {
	// 	name:  "command and pipe missing command",
	// 	input: "m {{ 3 | }}",
	// },
	// {
	// 	name:  "command missing value",
	// 	input: "m {{ | printf }}",
	// },
	// {
	// 	name:  "variable declaration missing value",
	// 	input: "m {{ $x := }}",
	// },
	// {
	// 	name:  "unclosed variable declaration",
	// 	input: "m {{ $x :=",
	// },
	// {
	// 	name:  "variable declaration missing =",
	// 	input: "m {{ $x : }}",
	// },
	// {
	// 	name:  "unclosed variable declaration with pipe",
	// 	input: "m {{ $x := 3 |",
	// },
	// {
	// 	name:  "variable declaration with pipe and missing command",
	// 	input: "m {{ $x :=  | printf }}",
	// },
}

func TestTreeRobustness(t *testing.T) {
	for _, test := range robustTreeTests {
		t.Run(test.name, func(t *testing.T) {
			tr := New("robust")
			tr.Mode = ParsePartial
			_, err := tr.Parse(test.input, "", "", make(map[string]*Tree), nil)
			if err != nil && test.ok {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil && !test.ok {
				t.Errorf("expected error, got none")
			}
			if err != nil && !strings.Contains(err.Error(), test.message) {
				t.Errorf("error %q does not contain %q", err, test.message)
			}
			if !CompareTrees(tr, test.result) {
				t.Errorf("unexpected parse tree: got %#v, want %#v", tr, test.result)
			}
		})
	}
}

// CompareTrees reports whether two trees are structurally identical,
// comparing all node fields that carry semantic meaning. Tree-internal
// pointers (tr *Tree) are ignored so that hand-built expected values in
// tests do not need to be fully wired up.
func CompareTrees(a, b *Tree) bool {
	if a == nil || b == nil {
		return a == b
	}
	return compareListNodes(a.Root, b.Root)
}

func compareListNodes(a, b *ListNode) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.Nodes) != len(b.Nodes) {
		return false
	}
	for i := range a.Nodes {
		if !compareNodes(a.Nodes[i], b.Nodes[i]) {
			return false
		}
	}
	return true
}

func comparePipeNodes(a, b *PipeNode) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.IsAssign != b.IsAssign || len(a.Decl) != len(b.Decl) || len(a.Cmds) != len(b.Cmds) {
		return false
	}
	for i := range a.Decl {
		if !compareNodes(a.Decl[i], b.Decl[i]) {
			return false
		}
	}
	for i := range a.Cmds {
		if !compareCommandNodes(a.Cmds[i], b.Cmds[i]) {
			return false
		}
	}
	return true
}

func compareCommandNodes(a, b *CommandNode) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if !compareNodes(a.Args[i], b.Args[i]) {
			return false
		}
	}
	return true
}

func compareNodes(a, b Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type() != b.Type() {
		return false
	}
	// After the Type() check the concrete types are guaranteed to match,
	// so the unchecked assertions below are safe.
	switch a := a.(type) {
	case *ListNode:
		return compareListNodes(a, b.(*ListNode))

	case *TextNode:
		return string(a.Text) == string(b.(*TextNode).Text)

	case *CommentNode:
		return a.Text == b.(*CommentNode).Text

	case *ActionNode:
		bv := b.(*ActionNode)
		return a.Pos == bv.Pos && comparePipeNodes(a.Pipe, bv.Pipe)

	case *IfNode:
		bv := b.(*IfNode)
		return a.Pos == bv.Pos &&
			comparePipeNodes(a.Pipe, bv.Pipe) &&
			compareListNodes(a.List, bv.List) &&
			compareListNodes(a.ElseList, bv.ElseList)

	case *RangeNode:
		bv := b.(*RangeNode)
		return a.Pos == bv.Pos &&
			comparePipeNodes(a.Pipe, bv.Pipe) &&
			compareListNodes(a.List, bv.List) &&
			compareListNodes(a.ElseList, bv.ElseList)

	case *WithNode:
		bv := b.(*WithNode)
		return a.Pos == bv.Pos &&
			comparePipeNodes(a.Pipe, bv.Pipe) &&
			compareListNodes(a.List, bv.List) &&
			compareListNodes(a.ElseList, bv.ElseList)

	case *TemplateNode:
		bv := b.(*TemplateNode)
		return a.Pos == bv.Pos &&
			a.Name == bv.Name &&
			comparePipeNodes(a.Pipe, bv.Pipe)

	case *BreakNode:
		bv := b.(*BreakNode)
		return a.Pos == bv.Pos

	case *ContinueNode:
		bv := b.(*ContinueNode)
		return a.Pos == bv.Pos

	case *IdentifierNode:
		return a.Ident == b.(*IdentifierNode).Ident

	case *VariableNode:
		bv := b.(*VariableNode)
		if len(a.Ident) != len(bv.Ident) {
			return false
		}
		for i := range a.Ident {
			if a.Ident[i] != bv.Ident[i] {
				return false
			}
		}
		return true

	case *FieldNode:
		bv := b.(*FieldNode)
		if len(a.Ident) != len(bv.Ident) {
			return false
		}
		for i := range a.Ident {
			if a.Ident[i] != bv.Ident[i] {
				return false
			}
		}
		return true

	case *ChainNode:
		bv := b.(*ChainNode)
		if len(a.Field) != len(bv.Field) {
			return false
		}
		for i := range a.Field {
			if a.Field[i] != bv.Field[i] {
				return false
			}
		}
		return compareNodes(a.Node, bv.Node)

	case *DotNode:
		return true

	case *NilNode:
		return true

	case *BoolNode:
		return a.True == b.(*BoolNode).True

	case *NumberNode:
		return a.Text == b.(*NumberNode).Text

	case *StringNode:
		bv := b.(*StringNode)
		return a.Quoted == bv.Quoted && a.Text == bv.Text

	case *UndefinedNode:
		return a.Pos == b.(*UndefinedNode).Pos

	default:
		return false
	}
}
