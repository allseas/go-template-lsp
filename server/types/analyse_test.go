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
			tree := NewTreeWithType(
				tc.parseTree,
				tc.funcs,
				tc.dotType,
				tc.pkg,
				tc.templateInputTypes,
			)
			if len(tree.TypeErrors) != len(tc.expectedErrors) {
				t.Fatalf(
					"Name: %s: Expected %d type errors, got %d: %v",
					tc.name,
					len(tc.expectedErrors),
					len(tree.TypeErrors),
					tree.TypeErrors,
				)
			}
			for i, want := range tc.expectedErrors {
				if tree.TypeErrors[i].typ != want.typ {
					t.Fatalf(
						"type error [%d]: got category %d (%q), want category %d",
						i,
						tree.TypeErrors[i].typ,
						tree.TypeErrors[i].Err,
						want.typ,
					)
				}
			}
			if diff := CompareTypeTrees(tree, tc.resTree); diff != "" {
				t.Fatalf("type tree mismatch: %s", diff)
			}
		})
	}
}

func TestAnalyseNode_UnknownNodeTypePanics(t *testing.T) {
	for _, tc := range analyseNodePanicTestCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic, got none")
				}
				msg, ok := r.(string)
				if !ok {
					t.Fatalf("expected panic value to be a string, got %T: %v", r, r)
				}
				if msg != tc.wantPanic {
					t.Fatalf("panic message: got %q, want %q", msg, tc.wantPanic)
				}
			}()

			ctx := &analysisCtx{
				funcs: make(map[string]*types.Func),
				vars:  []*VariableNode{},
			}
			analyseNode(tc.node, nil, ctx)
		})
	}
}

func TestAnalyseTree_NilDotTypeResolvesToEmptyInterface(t *testing.T) {
	tree := NewTreeWithType(tree("test", list(varn("$"))), nil, nil, nil, nil)
	if tree.Root == nil {
		t.Fatal("expected non-nil Root, got nil")
	}
	if len(tree.Root.Nodes) != 1 {
		t.Fatalf("expected 1 node in Root, got %d", len(tree.Root.Nodes))
	}
	if tree.Root.Nodes[0].ValueType() == nil {
		t.Fatal("expected non-nil DotType, got nil")
	}
	if !types.Identical(
		tree.Root.Nodes[0].ValueType(),
		types.NewInterfaceType(nil, nil).Complete(),
	) {
		t.Fatalf("DotType: got %v, want empty interface", tree.Root.Nodes[0].ValueType())
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
		if diff := compareTypeNodes(
			a.Nodes[i],
			b.Nodes[i],
			fmt.Sprintf("%s[%d]", path, i),
		); diff != "" {
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
		if diff := compareTypeNodes(
			a.Decl[i],
			b.Decl[i],
			fmt.Sprintf("%s.Decl[%d]", path, i),
		); diff != "" {
			return diff
		}
	}
	for i := range a.Cmds {
		if diff := compareTypeCommandNodes(
			a.Cmds[i],
			b.Cmds[i],
			fmt.Sprintf("%s.Cmds[%d]", path, i),
		); diff != "" {
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
		if diff := compareTypeNodes(
			a.Args[i],
			b.Args[i],
			fmt.Sprintf("%s.Args[%d]", path, i),
		); diff != "" {
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
		return fmt.Sprintf(
			"%s: got node type %v (%T), want %v (%T)",
			path,
			a.Type(),
			a,
			b.Type(),
			b,
		)
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
		return compareTypeTextNodes(a, b.(*TextNode), path)
	case *CommentNode:
		return compareTypeCommentNodes(a, b.(*CommentNode), path)
	case *ActionNode:
		return compareTypePipeNodes(a.Pipe, b.(*ActionNode).Pipe, path+".Pipe")
	case *IfNode:
		return compareTypeBranchNodes(&a.BranchNode, &b.(*IfNode).BranchNode, path)
	case *RangeNode:
		return compareTypeBranchNodes(&a.BranchNode, &b.(*RangeNode).BranchNode, path)
	case *WithNode:
		return compareTypeBranchNodes(&a.BranchNode, &b.(*WithNode).BranchNode, path)
	case *TemplateNode:
		return compareTypeTemplateNodes(a, b.(*TemplateNode), path)
	case *IdentifierNode:
		return compareTypeIdentifierNodes(a, b.(*IdentifierNode), path)
	case *VariableNode:
		return compareTypeIdentSliceNodes(a.Ident, b.(*VariableNode).Ident, path+".Ident")
	case *FieldNode:
		return compareTypeIdentSliceNodes(a.Ident, b.(*FieldNode).Ident, path+".Ident")
	case *ChainNode:
		return compareTypeChainNodes(a, b.(*ChainNode), path)
	case *BoolNode:
		return compareTypeBoolNodes(a, b.(*BoolNode), path)
	case *NumberNode:
		return compareTypeNumberNodes(a, b.(*NumberNode), path)
	case *StringNode:
		return compareTypeStringNodes(a, b.(*StringNode), path)
	case *DotNode, *NilNode, *BreakNode, *ContinueNode, *UndefinedNode:
		// no additional fields to compare beyond type and ValueType checked above
		return ""
	default:
		return fmt.Sprintf("%s: unhandled node type %T", path, a)
	}
}

func compareTypeTextNodes(a, b *TextNode, path string) string {
	if string(a.Text) != string(b.Text) {
		return fmt.Sprintf("%s: got text %q, want %q", path, a.Text, b.Text)
	}
	return ""
}

func compareTypeCommentNodes(a, b *CommentNode, path string) string {
	if a.Text != b.Text {
		return fmt.Sprintf("%s: got comment %q, want %q", path, a.Text, b.Text)
	}
	return ""
}

func compareTypeBranchNodes(a, b *BranchNode, path string) string {
	if diff := compareTypePipeNodes(a.Pipe, b.Pipe, path+".Pipe"); diff != "" {
		return diff
	}
	if diff := compareTypeListNodes(a.List, b.List, path+".List"); diff != "" {
		return diff
	}
	return compareTypeListNodes(a.ElseList, b.ElseList, path+".ElseList")
}

func compareTypeTemplateNodes(a, b *TemplateNode, path string) string {
	if a.Name != b.Name {
		return fmt.Sprintf("%s.Name: got %q, want %q", path, a.Name, b.Name)
	}
	return compareTypePipeNodes(a.Pipe, b.Pipe, path+".Pipe")
}

func compareTypeIdentifierNodes(a, b *IdentifierNode, path string) string {
	if a.Ident != b.Ident {
		return fmt.Sprintf("%s.Ident: got %q, want %q", path, a.Ident, b.Ident)
	}
	return ""
}

func compareTypeIdentSliceNodes(a, b []string, path string) string {
	if strings.Join(a, ".") != strings.Join(b, ".") {
		return fmt.Sprintf("%s: got %v, want %v", path, a, b)
	}
	return ""
}

func compareTypeChainNodes(a, b *ChainNode, path string) string {
	if strings.Join(a.Field, ".") != strings.Join(b.Field, ".") {
		return fmt.Sprintf("%s.Field: got %v, want %v", path, a.Field, b.Field)
	}
	return compareTypeNodes(a.Node, b.Node, path+".Node")
}

func compareTypeBoolNodes(a, b *BoolNode, path string) string {
	if a.True != b.True {
		return fmt.Sprintf("%s.True: got %v, want %v", path, a.True, b.True)
	}
	return ""
}

func compareTypeNumberNodes(a, b *NumberNode, path string) string {
	if a.Text != b.Text {
		return fmt.Sprintf("%s.Text: got %q, want %q", path, a.Text, b.Text)
	}
	return ""
}

func compareTypeStringNodes(a, b *StringNode, path string) string {
	if a.Text != b.Text {
		return fmt.Sprintf("%s.Text: got %q, want %q", path, a.Text, b.Text)
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
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}
