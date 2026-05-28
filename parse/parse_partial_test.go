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
	"fmt"
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
	// {
	// 	name:  "unclosed action",
	// 	input: "{{",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{
	// 				&UndefinedNode{
	// 					NodeType: NodeUndefined,
	// 					Pos:      trimMarkerLen,
	// 					Err:      errors.New("unclosed action"),
	// 					str:      "{{",
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: unexpected token in command: unclosed action"),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	// {
	// 	name:  "empty action",
	// 	input: "{{}}",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{

	// 				&ActionNode{
	// 					NodeType: NodeAction,
	// 					Pos:      0,
	// 					Pipe: &PipeNode{
	// 						NodeType: NodePipe,
	// 						Pos:      0,
	// 						Cmds: []*CommandNode{
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&UndefinedNode{
	// 										NodeType: NodeUndefined,
	// 										Pos:      2,
	// 										Err:      errors.New("template: robust:1: missing value for command"),
	// 										str:      "",
	// 									},
	// 								},
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: missing value for command"),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	// {
	// 	name:  "unclosed action with command",
	// 	input: "m {{ 3 {{3}} t",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{
	// 				&TextNode{
	// 					NodeType: NodeText,
	// 					Pos:      0,
	// 					Text:     []byte("m "),
	// 				},
	// 				&UndefinedNode{
	// 					NodeType: NodeUndefined,
	// 					Pos:      2,
	// 					Err:      errors.New("template: robust:1: unexpected token in action: \"{{\""),
	// 					str:      "\"3\"\"{{\"",
	// 				},
	// 				&ActionNode{
	// 					NodeType: NodeAction,
	// 					Pos:      7,
	// 					Pipe: &PipeNode{
	// 						NodeType: NodePipe,
	// 						Pos:      9,
	// 						Cmds: []*CommandNode{
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&NumberNode{
	// 										NodeType: NodeNumber,
	// 										Pos:      10,
	// 										Text:     "3",
	// 									},
	// 								},
	// 								Pos: 9,
	// 							},
	// 						},
	// 					},
	// 				},
	// 				&TextNode{
	// 					NodeType: NodeText,
	// 					Pos:      15,
	// 					Text:     []byte(" t"),
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: unexpected token in command: unclosed action with command"),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	// {
	// 	name:  "unclosed action with command and pipe",
	// 	input: "m {{ 3 |",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{
	// 				&TextNode{
	// 					NodeType: NodeText,
	// 					Pos:      0,
	// 					Text:     []byte("m "),
	// 				},
	// 				&ActionNode{
	// 					NodeType: NodeAction,
	// 					Pos:      2,
	// 					Pipe: &PipeNode{
	// 						NodeType: NodePipe,
	// 						Pos:      4,
	// 						Cmds: []*CommandNode{
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&NumberNode{
	// 										NodeType: NodeNumber,
	// 										Pos:      5,
	// 										Text:     "3",
	// 									},
	// 								},
	// 								Pos: 4,
	// 							},
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&UndefinedNode{
	// 										NodeType: NodeUndefined,
	// 										Pos:      8,
	// 										Err:      errors.New("template: robust:1: missing value for command"),
	// 										str:      "",
	// 									},
	// 								},
	// 								Pos: 6,
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: unexpected token in command: unclosed action with command and pipe"),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	// {
	// 	name:  "command and pipe missing command",
	// 	input: "m {{ 3 | }}",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{
	// 				&TextNode{
	// 					NodeType: NodeText,
	// 					Pos:      0,
	// 					Text:     []byte("m "),
	// 				},
	// 				&ActionNode{
	// 					NodeType: NodeAction,
	// 					Pos:      2,
	// 					Pipe: &PipeNode{
	// 						NodeType: NodePipe,
	// 						Pos:      4,
	// 						Cmds: []*CommandNode{
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&NumberNode{
	// 										NodeType: NodeNumber,
	// 										Pos:      5,
	// 										Text:     "3",
	// 									},
	// 								},
	// 								Pos: 4,
	// 							},
	// 							// Trailing pipes are apparently not a syntax error,
	// 							// we have to keep this behaviour for backwards compatibility.
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: unexpected token in command: command and pipe missing command"),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	// {
	// 	name:  "unfinished field chain",
	// 	input: "m {{ .A. }}",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{
	// 				&TextNode{
	// 					NodeType: NodeText,
	// 					Pos:      0,
	// 					Text:     []byte("m "),
	// 				},
	// 				&ActionNode{
	// 					NodeType: NodeAction,
	// 					Pos:      2,
	// 					Pipe: &PipeNode{
	// 						NodeType: NodePipe,
	// 						Pos:      4,
	// 						Cmds: []*CommandNode{
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&FieldNode{
	// 										NodeType: NodeField,
	// 										Pos:      5,
	// 										Ident:    []string{"A"},
	// 									},
	// 									&UndefinedNode{
	// 										NodeType: NodeUndefined,
	// 										Pos:      7,
	// 										Err:      errors.New("template: robust:1: unexpected token in field chain: ."),
	// 										str:      ".",
	// 									},
	// 								},
	// 								Pos: 4,
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: unexpected token in field chain: ."),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	// {
	// 	name:  "command missing value",
	// 	input: "m {{ | print }}",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{
	// 				&TextNode{
	// 					NodeType: NodeText,
	// 					Pos:      0,
	// 					Text:     []byte("m "),
	// 				},
	// 				&ActionNode{
	// 					NodeType: NodeAction,
	// 					Pos:      2,
	// 					Pipe: &PipeNode{
	// 						NodeType: NodePipe,
	// 						Pos:      4,
	// 						Cmds: []*CommandNode{
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&UndefinedNode{
	// 										NodeType: NodeUndefined,
	// 										Pos:      5,
	// 										Err:      errors.New("template: robust:1: missing value for command"),
	// 										str:      "",
	// 									},
	// 								},
	// 								Pos: 4,
	// 							},
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&IdentifierNode{
	// 										NodeType: NodeIdentifier,
	// 										Pos:      8,
	// 										Ident:    "print",
	// 									},
	// 								},
	// 								Pos: 6,
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: unexpected token in command: command missing value"),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	// {
	// 	name:  "variable declaration missing value",
	// 	input: "m {{ $x := }}",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{
	// 				&TextNode{
	// 					NodeType: NodeText,
	// 					Pos:      0,
	// 					Text:     []byte("m "),
	// 				},
	// 				&ActionNode{
	// 					NodeType: NodeAction,
	// 					Pos:      2,
	// 					Pipe: &PipeNode{
	// 						NodeType: NodePipe,
	// 						Pos:      4,
	// 						IsAssign: false,
	// 						Decl: []*VariableNode{
	// 							{
	// 								NodeType: NodeVariable,
	// 								Pos:      5,
	// 								Ident:    []string{"$x"},
	// 							},
	// 						},
	// 						Cmds: []*CommandNode{
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&UndefinedNode{
	// 										NodeType: NodeUndefined,
	// 										Pos:      5,
	// 										Err:      errors.New("template: robust:1: missing value for command"),
	// 										str:      "",
	// 									},
	// 								},
	// 								Pos: 6,
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: missing value for command"),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	// {
	// 	name:  "unclosed variable declaration",
	// 	input: "m {{ $x :=",
	// 	result: &Tree{
	// 		Root: &ListNode{
	// 			NodeType: NodeList,
	// 			Pos:      0,
	// 			Nodes: []Node{
	// 				&TextNode{
	// 					NodeType: NodeText,
	// 					Pos:      0,
	// 					Text:     []byte("m "),
	// 				},
	// 				&ActionNode{
	// 					NodeType: NodeAction,
	// 					Pos:      2,
	// 					Pipe: &PipeNode{
	// 						NodeType: NodePipe,
	// 						Pos:      4,
	// 						IsAssign: false,
	// 						Decl: []*VariableNode{
	// 							{
	// 								NodeType: NodeVariable,
	// 								Pos:      5,
	// 								Ident:    []string{"$x"},
	// 							},
	// 						},
	// 						Cmds: []*CommandNode{
	// 							{
	// 								NodeType: NodeCommand,
	// 								Args: []Node{
	// 									&UndefinedNode{
	// 										NodeType: NodeUndefined,
	// 										Pos:      10,
	// 										Err:      errors.New("template: robust:1: unclosed variable declaration"),
	// 										str:      "",
	// 									},
	// 								},
	// 								Pos: 6,
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		Mode: ParsePartial,
	// 		Errors: []error{
	// 			errors.New("template: robust:1: unclosed variable declaration"),
	// 		},
	// 	},
	// 	ok:      true,
	// 	message: "",
	// },
	{
		name:  "variable declaration missing =",
		input: "m {{ $x : }}",
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
					&ActionNode{
						NodeType: NodeAction,
						Pos:      2,
						Pipe: &PipeNode{
							NodeType: NodePipe,
							Pos:      4,
							IsAssign: false,
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      5,
											Err:      errors.New("template: robust:1: undefined variable: $x"),
										},
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      8,
											Err:      errors.New("template: robust:1: unexpected token in command: expected :="),
											str:      ":",
										},
									},
								},
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      5,
											Err:      errors.New("template: robust:1: variable declaration missing ="),
											str:      "",
										},
									},
									Pos: 6,
								},
							},
						},
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1: variable declaration missing ="),
			},
		},
		ok:      true,
		message: "",
	},
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
			tr.Mode = ParsePartial | SkipFuncCheck
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
			if diff := CompareTrees(tr, test.result); diff != "" {
				t.Errorf("trees differ at %s\n  got:  %#v\n  want: %#v", diff, tr, test.result)
			}
		})
	}
}

// CompareTrees reports where two trees first differ, returning an empty string
// if they are structurally identical. Tree-internal pointers (tr *Tree) are
// ignored so that hand-built expected values in tests do not need to be fully
// wired up.
func CompareTrees(a, b *Tree) string {
	if a == nil && b == nil {
		return ""
	}
	if a == nil {
		return "root: got nil tree, want non-nil"
	}
	if b == nil {
		return "root: got non-nil tree, want nil"
	}
	return compareListNodes(a.Root, b.Root, "root")
}

func compareListNodes(a, b *ListNode, path string) string {
	if a == nil && b == nil {
		return ""
	}
	if a == nil {
		return fmt.Sprintf("%s: got nil, want non-nil ListNode", path)
	}
	if b == nil {
		return fmt.Sprintf("%s: got non-nil ListNode, want nil", path)
	}
	if len(a.Nodes) != len(b.Nodes) {
		return fmt.Sprintf("%s: got %d nodes, want %d", path, len(a.Nodes), len(b.Nodes))
	}
	for i := range a.Nodes {
		if diff := compareNodes(a.Nodes[i], b.Nodes[i], fmt.Sprintf("%s[%d]", path, i)); diff != "" {
			return diff
		}
	}
	return ""
}

func comparePipeNodes(a, b *PipeNode, path string) string {
	if a == nil && b == nil {
		return ""
	}
	if a == nil {
		return fmt.Sprintf("%s: got nil, want non-nil PipeNode", path)
	}
	if b == nil {
		return fmt.Sprintf("%s: got non-nil PipeNode, want nil", path)
	}
	if a.IsAssign != b.IsAssign {
		return fmt.Sprintf("%s.IsAssign: got %v, want %v", path, a.IsAssign, b.IsAssign)
	}
	if len(a.Decl) != len(b.Decl) {
		return fmt.Sprintf("%s.Decl: got %d items, want %d", path, len(a.Decl), len(b.Decl))
	}
	if len(a.Cmds) != len(b.Cmds) {
		return fmt.Sprintf("%s.Cmds: got %d items, want %d", path, len(a.Cmds), len(b.Cmds))
	}
	for i := range a.Decl {
		if diff := compareNodes(a.Decl[i], b.Decl[i], fmt.Sprintf("%s.Decl[%d]", path, i)); diff != "" {
			return diff
		}
	}
	for i := range a.Cmds {
		if diff := compareCommandNodes(a.Cmds[i], b.Cmds[i], fmt.Sprintf("%s.Cmds[%d]", path, i)); diff != "" {
			return diff
		}
	}
	return ""
}

func compareCommandNodes(a, b *CommandNode, path string) string {
	if a == nil && b == nil {
		return ""
	}
	if a == nil {
		return fmt.Sprintf("%s: got nil, want non-nil CommandNode", path)
	}
	if b == nil {
		return fmt.Sprintf("%s: got non-nil CommandNode, want nil", path)
	}
	if len(a.Args) != len(b.Args) {
		return fmt.Sprintf("%s.Args: got %d items, want %d", path, len(a.Args), len(b.Args))
	}
	for i := range a.Args {
		if diff := compareNodes(a.Args[i], b.Args[i], fmt.Sprintf("%s.Args[%d]", path, i)); diff != "" {
			return diff
		}
	}
	return ""
}

func compareNodes(a, b Node, path string) string {
	if a == nil && b == nil {
		return ""
	}
	if a == nil {
		return fmt.Sprintf("%s: got nil, want %T", path, b)
	}
	if b == nil {
		return fmt.Sprintf("%s: got %T, want nil", path, a)
	}
	if a.Type() != b.Type() {
		return fmt.Sprintf("%s: got node type %v (%T), want %v (%T)", path, a.Type(), a, b.Type(), b)
	}
	switch a := a.(type) {
	case *ListNode:
		return compareListNodes(a, b.(*ListNode), path)

	case *TextNode:
		bv := b.(*TextNode)
		if string(a.Text) != string(bv.Text) {
			return fmt.Sprintf("%s: got text %q, want %q", path, a.Text, bv.Text)
		}

	case *CommentNode:
		bv := b.(*CommentNode)
		if a.Text != bv.Text {
			return fmt.Sprintf("%s: got comment %q, want %q", path, a.Text, bv.Text)
		}

	case *ActionNode:
		bv := b.(*ActionNode)
		if a.Pos != bv.Pos {
			return fmt.Sprintf("%s.Pos: got %v, want %v", path, a.Pos, bv.Pos)
		}
		return comparePipeNodes(a.Pipe, bv.Pipe, path+".Pipe")

	case *IfNode:
		bv := b.(*IfNode)
		if a.Pos != bv.Pos {
			return fmt.Sprintf("%s.Pos: got %v, want %v", path, a.Pos, bv.Pos)
		}
		if diff := comparePipeNodes(a.Pipe, bv.Pipe, path+".Pipe"); diff != "" {
			return diff
		}
		if diff := compareListNodes(a.List, bv.List, path+".List"); diff != "" {
			return diff
		}
		return compareListNodes(a.ElseList, bv.ElseList, path+".ElseList")

	case *RangeNode:
		bv := b.(*RangeNode)
		if a.Pos != bv.Pos {
			return fmt.Sprintf("%s.Pos: got %v, want %v", path, a.Pos, bv.Pos)
		}
		if diff := comparePipeNodes(a.Pipe, bv.Pipe, path+".Pipe"); diff != "" {
			return diff
		}
		if diff := compareListNodes(a.List, bv.List, path+".List"); diff != "" {
			return diff
		}
		return compareListNodes(a.ElseList, bv.ElseList, path+".ElseList")

	case *WithNode:
		bv := b.(*WithNode)
		if a.Pos != bv.Pos {
			return fmt.Sprintf("%s.Pos: got %v, want %v", path, a.Pos, bv.Pos)
		}
		if diff := comparePipeNodes(a.Pipe, bv.Pipe, path+".Pipe"); diff != "" {
			return diff
		}
		if diff := compareListNodes(a.List, bv.List, path+".List"); diff != "" {
			return diff
		}
		return compareListNodes(a.ElseList, bv.ElseList, path+".ElseList")

	case *TemplateNode:
		bv := b.(*TemplateNode)
		if a.Pos != bv.Pos {
			return fmt.Sprintf("%s.Pos: got %v, want %v", path, a.Pos, bv.Pos)
		}
		if a.Name != bv.Name {
			return fmt.Sprintf("%s.Name: got %q, want %q", path, a.Name, bv.Name)
		}
		return comparePipeNodes(a.Pipe, bv.Pipe, path+".Pipe")

	case *BreakNode:
		bv := b.(*BreakNode)
		if a.Pos != bv.Pos {
			return fmt.Sprintf("%s.Pos: got %v, want %v", path, a.Pos, bv.Pos)
		}

	case *ContinueNode:
		bv := b.(*ContinueNode)
		if a.Pos != bv.Pos {
			return fmt.Sprintf("%s.Pos: got %v, want %v", path, a.Pos, bv.Pos)
		}

	case *IdentifierNode:
		bv := b.(*IdentifierNode)
		if a.Ident != bv.Ident {
			return fmt.Sprintf("%s: got identifier %q, want %q", path, a.Ident, bv.Ident)
		}

	case *VariableNode:
		bv := b.(*VariableNode)
		if len(a.Ident) != len(bv.Ident) {
			return fmt.Sprintf("%s.Ident: got %d parts, want %d", path, len(a.Ident), len(bv.Ident))
		}
		for i := range a.Ident {
			if a.Ident[i] != bv.Ident[i] {
				return fmt.Sprintf("%s.Ident[%d]: got %q, want %q", path, i, a.Ident[i], bv.Ident[i])
			}
		}

	case *FieldNode:
		bv := b.(*FieldNode)
		if len(a.Ident) != len(bv.Ident) {
			return fmt.Sprintf("%s.Ident: got %d parts, want %d", path, len(a.Ident), len(bv.Ident))
		}
		for i := range a.Ident {
			if a.Ident[i] != bv.Ident[i] {
				return fmt.Sprintf("%s.Ident[%d]: got %q, want %q", path, i, a.Ident[i], bv.Ident[i])
			}
		}

	case *ChainNode:
		bv := b.(*ChainNode)
		if len(a.Field) != len(bv.Field) {
			return fmt.Sprintf("%s.Field: got %d parts, want %d", path, len(a.Field), len(bv.Field))
		}
		for i := range a.Field {
			if a.Field[i] != bv.Field[i] {
				return fmt.Sprintf("%s.Field[%d]: got %q, want %q", path, i, a.Field[i], bv.Field[i])
			}
		}
		return compareNodes(a.Node, bv.Node, path+".Node")

	case *DotNode:
		// no fields to compare

	case *NilNode:
		// no fields to compare

	case *BoolNode:
		bv := b.(*BoolNode)
		if a.True != bv.True {
			return fmt.Sprintf("%s: got bool %v, want %v", path, a.True, bv.True)
		}

	case *NumberNode:
		bv := b.(*NumberNode)
		if a.Text != bv.Text {
			return fmt.Sprintf("%s: got number %q, want %q", path, a.Text, bv.Text)
		}

	case *StringNode:
		bv := b.(*StringNode)
		if a.Quoted != bv.Quoted {
			return fmt.Sprintf("%s.Quoted: got %q, want %q", path, a.Quoted, bv.Quoted)
		}
		if a.Text != bv.Text {
			return fmt.Sprintf("%s.Text: got %q, want %q", path, a.Text, bv.Text)
		}

	case *UndefinedNode:
		bv := b.(*UndefinedNode)
		if a.Pos != bv.Pos {
			return fmt.Sprintf("%s.Pos: got %v, want %v", path, a.Pos, bv.Pos)
		}

	default:
		return fmt.Sprintf("%s: unhandled node type %T", path, a)
	}
	return ""
}
