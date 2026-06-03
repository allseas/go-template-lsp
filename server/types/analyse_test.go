package types

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyze(t *testing.T) {
	for _, tc := range analyseTestCases {
		t.Run(tc.name, func(t *testing.T) {
			tree := NewTreeWithType(tc.parseTree, tc.funcs, tc.dotType, tc.pkg)
			if len(tree.Errors) != len(tc.expectedErrors) {
				t.Fatalf("Expected %d errors, got %d: %v", len(tc.expectedErrors), len(tree.Errors), tree.Errors)
			}
			if diff := CompareTypeTrees(tree, tc.resTree); diff != "" {
				t.Fatalf("type tree mismatch: %s", diff)
			}
		})
	}
}

// CompareTypeTrees reports where two type trees first differ, returning an
// empty string if they are structurally and type-annotation identical.
// Parent pointers are ignored so hand-built expected values in tests do not
// need to be fully wired up.
func CompareTypeTrees(a, b Tree) string {
	return compareTypeListNodes(a.Root, b.Root, "root")
}

func compareTypeListNodes(a, b *ListNode, path string) string {
	if a == nil && b == nil {
		return ""
	}
	if a == nil {
		return fmt.Sprintf("%s: got nil, want non-nil ListNode", path)
	}
	if b == nil {
		return fmt.Sprintf("%s: got non-nil ListNode, want nil", path)
	}
	if diff := compareValueTypes(a.ValueType(), b.ValueType(), path+".typ"); diff != "" {
		return diff
	}
	if len(a.Nodes) != len(b.Nodes) {
		return fmt.Sprintf("%s: got %d nodes, want %d", path, len(a.Nodes), len(b.Nodes))
	}
	for i := range a.Nodes {
		if diff := compareTypeNodes(a.Nodes[i], b.Nodes[i], fmt.Sprintf("%s[%d]", path, i)); diff != "" {
			return diff
		}
	}
	return ""
}

func compareTypePipeNodes(a, b *PipeNode, path string) string {
	if a == nil && b == nil {
		return ""
	}
	if a == nil {
		return fmt.Sprintf("%s: got nil, want non-nil PipeNode", path)
	}
	if b == nil {
		return fmt.Sprintf("%s: got non-nil PipeNode, want nil", path)
	}
	if diff := compareValueTypes(a.ValueType(), b.ValueType(), path+".typ"); diff != "" {
		return diff
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
		if diff := compareTypeNodes(a.Decl[i], b.Decl[i], fmt.Sprintf("%s.Decl[%d]", path, i)); diff != "" {
			return diff
		}
	}
	for i := range a.Cmds {
		if diff := compareTypeCommandNodes(a.Cmds[i], b.Cmds[i], fmt.Sprintf("%s.Cmds[%d]", path, i)); diff != "" {
			return diff
		}
	}
	return ""
}

func compareTypeCommandNodes(a, b *CommandNode, path string) string {
	if a == nil && b == nil {
		return ""
	}
	if a == nil {
		return fmt.Sprintf("%s: got nil, want non-nil CommandNode", path)
	}
	if b == nil {
		return fmt.Sprintf("%s: got non-nil CommandNode, want nil", path)
	}
	if diff := compareValueTypes(a.ValueType(), b.ValueType(), path+".typ"); diff != "" {
		return diff
	}
	if len(a.Args) != len(b.Args) {
		return fmt.Sprintf("%s.Args: got %d items, want %d", path, len(a.Args), len(b.Args))
	}
	for i := range a.Args {
		if diff := compareTypeNodes(a.Args[i], b.Args[i], fmt.Sprintf("%s.Args[%d]", path, i)); diff != "" {
			return diff
		}
	}
	return ""
}

func compareTypeNodes(a, b Node, path string) string {
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
	if diff := compareValueTypes(a.ValueType(), b.ValueType(), path+".typ"); diff != "" {
		return diff
	}
	switch a := a.(type) {
	case *ListNode:
		return compareTypeListNodes(a, b.(*ListNode), path)

	case *PipeNode:
		return compareTypePipeNodes(a, b.(*PipeNode), path)

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
		return compareTypePipeNodes(a.Pipe, bv.Pipe, path+".Pipe")

	case *IfNode:
		bv := b.(*IfNode)
		if diff := compareTypePipeNodes(a.Pipe, bv.Pipe, path+".Pipe"); diff != "" {
			return diff
		}
		if diff := compareTypeListNodes(a.List, bv.List, path+".List"); diff != "" {
			return diff
		}
		return compareTypeListNodes(a.ElseList, bv.ElseList, path+".ElseList")

	case *RangeNode:
		bv := b.(*RangeNode)
		if diff := compareTypePipeNodes(a.Pipe, bv.Pipe, path+".Pipe"); diff != "" {
			return diff
		}
		if diff := compareTypeListNodes(a.List, bv.List, path+".List"); diff != "" {
			return diff
		}
		return compareTypeListNodes(a.ElseList, bv.ElseList, path+".ElseList")

	case *WithNode:
		bv := b.(*WithNode)
		if diff := compareTypePipeNodes(a.Pipe, bv.Pipe, path+".Pipe"); diff != "" {
			return diff
		}
		if diff := compareTypeListNodes(a.List, bv.List, path+".List"); diff != "" {
			return diff
		}
		return compareTypeListNodes(a.ElseList, bv.ElseList, path+".ElseList")

	case *TemplateNode:
		bv := b.(*TemplateNode)
		if a.Name != bv.Name {
			return fmt.Sprintf("%s.Name: got %q, want %q", path, a.Name, bv.Name)
		}
		return compareTypePipeNodes(a.Pipe, bv.Pipe, path+".Pipe")

	case *IdentifierNode:
		bv := b.(*IdentifierNode)
		if a.Ident != bv.Ident {
			return fmt.Sprintf("%s.Ident: got %q, want %q", path, a.Ident, bv.Ident)
		}

	case *VariableNode:
		bv := b.(*VariableNode)
		if strings.Join(a.Ident, ".") != strings.Join(bv.Ident, ".") {
			return fmt.Sprintf("%s.Ident: got %v, want %v", path, a.Ident, bv.Ident)
		}

	case *FieldNode:
		bv := b.(*FieldNode)
		if strings.Join(a.Ident, ".") != strings.Join(bv.Ident, ".") {
			return fmt.Sprintf("%s.Ident: got %v, want %v", path, a.Ident, bv.Ident)
		}

	case *ChainNode:
		bv := b.(*ChainNode)
		if strings.Join(a.Field, ".") != strings.Join(bv.Field, ".") {
			return fmt.Sprintf("%s.Field: got %v, want %v", path, a.Field, bv.Field)
		}
		return compareTypeNodes(a.Node, bv.Node, path+".Node")

	case *BoolNode:
		bv := b.(*BoolNode)
		if a.True != bv.True {
			return fmt.Sprintf("%s.True: got %v, want %v", path, a.True, bv.True)
		}

	case *NumberNode:
		bv := b.(*NumberNode)
		if a.Text != bv.Text {
			return fmt.Sprintf("%s.Text: got %q, want %q", path, a.Text, bv.Text)
		}

	case *StringNode:
		bv := b.(*StringNode)
		if a.Text != bv.Text {
			return fmt.Sprintf("%s.Text: got %q, want %q", path, a.Text, bv.Text)
		}

	case *DotNode, *NilNode, *BreakNode, *ContinueNode, *UndefinedNode:
		// no additional fields to compare beyond type and ValueType checked above

	default:
		return fmt.Sprintf("%s: unhandled node type %T", path, a)
	}
	return ""
}

// compareValueTypes returns a diff string if two types.Type values differ.
// Both nil is considered equal. Typed-nil (interface holding a nil concrete
// value) is also treated as nil.
func compareValueTypes(a, b types.Type, path string) string {
	if isNilType(a) && isNilType(b) {
		return ""
	}
	if isNilType(a) {
		return fmt.Sprintf("%s: got nil type, want %v", path, b)
	}
	if isNilType(b) {
		return fmt.Sprintf("%s: got %v, want nil type", path, a)
	}
	if !types.Identical(a, b) {
		return fmt.Sprintf("%s: got type %v, want %v", path, a, b)
	}
	return ""
}

func isNilType(t types.Type) bool {
	if t == nil {
		return true
	}
	v := reflect.ValueOf(t)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}
