// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse builds parse trees for templates as defined by text/template
// and html/template. Clients should use those packages to construct templates
// rather than this one, which provides shared internal data structures not
// intended for general use.
package parse

import (
	"fmt"
	"strings"
	"testing"
)

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

// tree comparison was ai generated for protyping

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
		if a.Err == nil && bv.Err != nil {
			return fmt.Sprintf("%s.Err: got nil, want %v", path, bv.Err)
		}
		if a.Err != nil && bv.Err == nil {
			return fmt.Sprintf("%s.Err: got %v, want nil", path, a.Err)
		}
		if a.Err != nil && bv.Err != nil && a.Err.Error() != bv.Err.Error() {
			return fmt.Sprintf("%s.Err: got %q, want %q", path, a.Err.Error(), bv.Err.Error())
		}
		if a.str != bv.str {
			return fmt.Sprintf("%s.str: got %q, want %q", path, a.str, bv.str)
		}

	default:
		return fmt.Sprintf("%s: unhandled node type %T", path, a)
	}
	return ""
}
