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
						Pos:      2,
						Err:      errors.New("template: robust:1:3: unexpected token in command: unclosed action"),
						str:      "unclosed action",
					},
					&UndefinedNode{
						NodeType: NodeUndefined,
						Pos:      2,
						Err:      errors.New("template: robust:1:3: unclosed action"),
						str:      "unclosed action",
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1:3: unexpected token in command: unclosed action"),
				errors.New("template: robust:1:3: unclosed action"),
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
											Err:      errors.New("template: robust:1:3: missing value for command"),
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
				errors.New("template: robust:1:2: missing value for command"),
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
						Err:      errors.New("template: robust:1:6: unexpected token in action: \"{{\""),
						str:      "3{{",
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
											Pos:      9,
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
						Pos:      12,
						Text:     []byte(" t"),
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1:6: unexpected token in command: unclosed action with command"),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "unclosed action with command and pipe",
		input: "m {{ 3 |",
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
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&NumberNode{
											NodeType: NodeNumber,
											Pos:      5,
											Text:     "3",
										},
									},
									Pos: 4,
								},
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      8,
											Err:      errors.New("template: robust:1:9: unexpected token in command: unclosed action"),
											str:      "unclosed action",
										},
									},
									Pos: 6,
								},
							},
						},
					},
					&UndefinedNode{
						NodeType: NodeUndefined,
						Pos:      8,
						Err:      errors.New("template: robust:1:9: unclosed action"),
						str:      "unclosed action",
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1:9: unexpected token in command: unclosed action with command and pipe"),
				errors.New("template: robust:1:9: unclosed action"),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "command and pipe missing command",
		input: "m {{ 3 | }}",
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
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&NumberNode{
											NodeType: NodeNumber,
											Pos:      5,
											Text:     "3",
										},
									},
									Pos: 4,
								},
								// Trailing pipes are apparently not a syntax error,
								// we have to keep this behaviour for backwards compatibility.
							},
						},
					},
				},
			},
			Mode:   ParsePartial,
			Errors: []error{},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "unfinished field chain",
		input: "m {{ .A. }}",
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
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&FieldNode{
											NodeType: NodeField,
											Pos:      5,
											Ident:    []string{"A"},
										},
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      7,
											Err:      errors.New("template: robust:1:8: unexpected token in command: <.>"),
											str:      ".",
										},
									},
									Pos: 4,
								},
							},
						},
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1:8: unexpected token in field chain: <.>"),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "command missing value",
		input: "m {{ | print }}",
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
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      5,
											Err:      errors.New("template: robust:1:6: unexpected token in pipeline: \"|\""),
											str:      "|",
										},
									},
									Pos: 4,
								},
								{
									NodeType: NodeCommand,
									Args: []Node{
										&IdentifierNode{
											NodeType: NodeIdentifier,
											Pos:      7,
											Ident:    "print",
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
				errors.New("template: robust:1:6: unexpected token in command: command missing value"),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "variable declaration missing value",
		input: "m {{ $x := }}",
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
							Decl: []*VariableNode{
								{
									NodeType: NodeVariable,
									Pos:      5,
									Ident:    []string{"$x"},
								},
							},
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      5,
											Err:      errors.New("template: robust:1:6: missing value for command"),
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
				errors.New("template: robust:1:6: missing value for command"),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "unclosed variable declaration",
		input: "m {{ $x :=",
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
							Decl: []*VariableNode{
								{
									NodeType: NodeVariable,
									Pos:      5,
									Ident:    []string{"$x"},
								},
							},
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      10,
											Err:      errors.New("template: robust:1:11: unexpected token in command: unclosed action"),
											str:      "unclosed action",
										},
									},
									Pos: 6,
								},
							},
						},
					},
					&UndefinedNode{
						NodeType: NodeUndefined,
						Pos:      10,
						Err:      errors.New("template: robust:1:11: unclosed action"),
						str:      "unclosed action",
					},
				},
			},
			Mode: ParsePartial,
			Errors: []error{
				errors.New("template: robust:1:11: unexpected token in command: unclosed action"),
				errors.New("template: robust:1:11: unclosed action"),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "variable declaration missing =",
		input: "m {{ $x : 3}}",
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
											Err:      errors.New("template: robust:1:6: undefined variable \"$x\""),
											str:      "$x",
										},
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      8,
											Err:      errors.New("template: robust:1:9: unexpected token in command: \":\""),
											str:      ":",
										},
									},
								},
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      10,
											Err:      errors.New("template: robust:1:11: unexpected literal operand in command: 3"),
											str:      "3",
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
				errors.New("template: robust:1:11: variable declaration missing ="),
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "assignment to an undeclared variable",
		input: "m {{ $x = 4 }}",
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
							IsAssign: true,
							Decl: []*VariableNode{
								{
									NodeType: NodeVariable,
									Pos:      5,
									Ident:    []string{"$x"},
								},
							},
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Pos:      6,
									Args: []Node{
										&NumberNode{
											NodeType: NodeNumber,
											Pos:      10,
											Text:     "4",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		ok:      true,
		message: "",
	},
	{
		name:  "variable declaration with pipe and missing command",
		input: "m {{ $x :=  | printf }}",
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
							Decl: []*VariableNode{
								{
									NodeType: NodeVariable,
									Pos:      5,
									Ident:    []string{"$x"},
								},
							},
							Cmds: []*CommandNode{
								{
									NodeType: NodeCommand,
									Args: []Node{
										&UndefinedNode{
											NodeType: NodeUndefined,
											Pos:      12,
											Err:      errors.New("template: robust:1:13: unexpected token in pipeline: \"|\""),
											str:      "|",
										},
									},
									Pos: 4,
								},
								{
									NodeType: NodeCommand,
									Args: []Node{
										&IdentifierNode{
											NodeType: NodeIdentifier,
											Pos:      14,
											Ident:    "printf",
										},
									},
									Pos: 6,
								},
							},
						},
					},
				},
			},
		},
		ok:      true,
		message: "",
	},
}
