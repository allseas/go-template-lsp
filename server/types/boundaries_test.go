package types

import (
	"go/types"
	"testing"
	parse "text-template-parser"

	"github.com/stretchr/testify/require"
)

type boundariesTestCase struct {
	name     string
	input    string
	expected []struct{ start, end Pos }
}

var setEndsForTreeTestCases = []boundariesTestCase{
	{
		name:  "simple action",
		input: `{{.}}`,
		expected: []struct{ start, end Pos }{
			{start: 0, end: 5},
			{start: 0, end: 5},
			{start: 2, end: 3},
			{start: 2, end: 3},
			{start: 2, end: 3},
		},
	},
	{
		name:  "variable declaration in range",
		input: `{{range $i, $e := .}}{{end}}`,
		expected: []struct{ start, end Pos }{
			{start: 0, end: 28},
			{start: 2, end: 7},
			{start: 8, end: 19},
			{start: 8, end: 10},
			{start: 12, end: 14},
			{start: 18, end: 19},
			{start: 18, end: 19},
			{start: 21, end: 28},
		},
	},
	{
		name:  "if with else and text",
		input: `{{if .}}true{{else}}false{{end}}`,
		expected: []struct{ start, end Pos }{
			{start: 0, end: 32},
			{start: 2, end: 4},
			{start: 5, end: 6},
			{start: 5, end: 6},
			{start: 5, end: 6},
			{start: 8, end: 20},
			{start: 8, end: 12},
			{start: 20, end: 32},
			{start: 20, end: 25},
		},
	},
	{
		name:  "literals in acitons",
		input: `{{ "hello" }}{{ 123 }}{{ true }}{{ false }}{{ nil }}`,
		expected: []struct{ start, end Pos }{
			{start: 0, end: 52},
			{start: 0, end: 12},
			{start: 3, end: 12},
			{start: 3, end: 12},
			{start: 3, end: 10},
			{start: 13, end: 21},
			{start: 16, end: 21},
			{start: 16, end: 21},
			{start: 16, end: 19},
			{start: 22, end: 31},
			{start: 25, end: 31},
			{start: 25, end: 31},
			{start: 25, end: 29},
			{start: 32, end: 42},
			{start: 35, end: 42},
			{start: 35, end: 42},
			{start: 35, end: 40},
			{start: 43, end: 52},
			{start: 46, end: 50},
			{start: 46, end: 50},
			{start: 46, end: 49},
		},
	},
	{
		name:  "comment node",
		input: `{{/* hello */}}`,
		expected: []struct{ start, end Pos }{
			{start: 2, end: 15}, // ListNode
			{start: 2, end: 15}, // CommentNode
		},
	},
	{
		name:  "break node",
		input: `{{range .}}{{break}}{{end}}`,
		expected: []struct{ start, end Pos }{
			{start: 0, end: 27},  // ListNode (root)
			{start: 2, end: 7},   // RangeNode
			{start: 8, end: 9},   // PipeNode
			{start: 8, end: 9},   // CommandNode
			{start: 8, end: 9},   // DotNode
			{start: 11, end: 27}, // ListNode (body)
			{start: 13, end: 20}, // BreakNode
		},
	},
	{
		name:  "continue node",
		input: `{{range .}}{{continue}}{{end}}`,
		expected: []struct{ start, end Pos }{
			{start: 0, end: 30},  // ListNode (root)
			{start: 2, end: 7},   // RangeNode
			{start: 8, end: 9},   // PipeNode
			{start: 8, end: 9},   // CommandNode
			{start: 8, end: 9},   // DotNode
			{start: 11, end: 30}, // ListNode (body)
			{start: 13, end: 23}, // ContinueNode
		},
	},
	{
		name:  "with node",
		input: `{{with .}}{{end}}`,
		expected: []struct{ start, end Pos }{
			{start: 0, end: 17},  // ListNode (root)
			{start: 2, end: 6},   // WithNode
			{start: 7, end: 8},   // PipeNode
			{start: 7, end: 8},   // CommandNode
			{start: 7, end: 8},   // DotNode
			{start: 10, end: 17}, // ListNode (body, empty)
		},
	},
	{
		name:  "chain node",
		input: `{{(.Name).IsLocal}}`,
		expected: []struct{ start, end Pos }{
			{start: 0, end: 19}, // ListNode (root)
			{start: 0, end: 19}, // ActionNode
			{start: 2, end: 17}, // PipeNode (outer)
			{start: 2, end: 17}, // CommandNode
			{start: 3, end: 10}, // ChainNode
			{start: 3, end: 3},  // PipeNode (inner, parenthesised)
			{start: 3, end: 3},  // CommandNode (inner)
			{start: 3, end: 4},  // FieldNode (.Name)
		},
	},
}

func TestSetEndsForTree(t *testing.T) {
	for _, tc := range setEndsForTreeTestCases {
		t.Run(tc.name, func(t *testing.T) {
			tree := parse.New("test")
			tree.Mode = parse.SkipFuncCheck | parse.ParsePartial | parse.ParseComments
			_, err := tree.Parse(tc.input, "{{", "}}", map[string]*parse.Tree{})
			require.NoError(t, err)

			ttree := NewTree(*tree, map[string]*types.Func{}, nil, nil)

			SetEndsForTree(ttree, Pos(len(tc.input)), &tc.input)

			nodes := collectNodes(ttree.Root)
			require.Equal(t, len(tc.expected), len(nodes))

			for i, node := range nodes {
				require.Equal(t, tc.expected[i].start, node.Position(), node.String())
				require.Equal(t, tc.expected[i].end, node.End(), node.String())
			}
		})
	}
}

func collectNodes(node Node) []Node {
	var nodes []Node
	var collect func(Node)
	collect = func(n Node) {
		nodes = append(nodes, n)
		for _, child := range children(n) {
			collect(child)
		}
	}
	collect(node)
	return nodes
}

func children(node Node) []Node {
	switch n := node.(type) {
	case *ListNode:
		return n.Nodes
	case *PipeNode:
		l := []Node{}
		for _, decl := range n.Decl {
			l = append(l, decl)
		}
		for _, cmd := range n.Cmds {
			l = append(l, cmd)
		}
		return l
	case *ActionNode:
		return []Node{n.Pipe}
	case *CommandNode:
		return n.Args
	case *ChainNode:
		return []Node{n.Node}
	case *IfNode:
		if n.ElseList != nil {
			return []Node{n.Pipe, n.List, n.ElseList}
		}
		return []Node{n.Pipe, n.List}
	case *RangeNode:
		if n.ElseList != nil {
			return []Node{n.Pipe, n.List, n.ElseList}
		}
		return []Node{n.Pipe, n.List}
	case *WithNode:
		if n.ElseList != nil {
			return []Node{n.Pipe, n.List, n.ElseList}
		}
		return []Node{n.Pipe, n.List}
	case *TemplateNode:
		return []Node{n.Pipe}
	case *FieldNode, *VariableNode, *IdentifierNode, *DotNode,
		*NilNode, *BoolNode, *NumberNode, *StringNode, *CommentNode,
		*BreakNode, *ContinueNode, *TextNode:
		return nil
	default:
		panic(n.Type())
	}
}
