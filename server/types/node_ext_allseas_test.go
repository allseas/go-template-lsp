//go:build allseas

// ai
package types

import (
	"fmt"
	"go/types"
	"reflect"
	"testing"
	parse "text-template-parser"
)

// parseAllseasTemplate parses src with the allseas-extended parser, in
// partial+comment mode (matching other type-package tests).
func parseAllseasTemplate(t *testing.T, src string) *parse.Tree {
	t.Helper()
	tree := parse.New("test")
	tree.Mode = parse.SkipFuncCheck | parse.ParsePartial | parse.ParseComments
	if _, err := tree.Parse(src, "{{", "}}", map[string]*parse.Tree{}); err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	return tree
}

// firstTable returns the first TableNode reachable from root, or nil.
func firstTable(root Node) *TableNode {
	var found *TableNode
	Inspect(root, func(n Node) bool {
		if t, ok := n.(*TableNode); ok && found == nil {
			found = t
		}
		return found == nil
	})
	return found
}

// --- TableNode accessors -------------------------------------------------

func TestTableNodeAccessors(t *testing.T) {
	parent := &ListNode{NodeType: NodeList, Pos: 0}
	stringType := types.Typ[types.String]
	tn := &TableNode{
		NodeType: NodeTable,
		Pos:      14,
		endPos:   28,
		Format:   "fmt",
		parent:   parent,
		typ:      stringType,
		isElse:   true,
	}
	if got := tn.Position(); got != 14 {
		t.Errorf("Position(): got %d, want 14", got)
	}
	if got := tn.End(); got != 28 {
		t.Errorf("End(): got %d, want 28", got)
	}
	if got := tn.ValueType(); got != stringType {
		t.Errorf("ValueType(): got %v, want %v", got, stringType)
	}
	if got := tn.Parent(); got != parent {
		t.Errorf("Parent(): got %v, want %v", got, parent)
	}
	if !tn.IsElseList() {
		t.Error("IsElseList(): got false, want true")
	}
	if got := tn.Type(); got != NodeTable {
		t.Errorf("Type(): got %v, want NodeTable", got)
	}
	if got := tn.String(); got != "{{block (table extension)}}" {
		t.Errorf("String(): got %q", got)
	}
}

func TestTableNodeCopy(t *testing.T) {
	orig := &TableNode{
		NodeType: NodeTable,
		Pos:      14,
		Format:   "fmt",
		Pipe:     &PipeNode{NodeType: NodePipe, Pos: 14},
		List:     &ListNode{NodeType: NodeList, Pos: 17},
		typ:      types.Typ[types.String],
	}
	cp, ok := orig.Copy().(*TableNode)
	if !ok {
		t.Fatalf("Copy(): wrong type")
	}
	if cp == orig {
		t.Fatal("Copy(): returned same pointer")
	}
	if cp.Format != orig.Format || cp.Position() != orig.Position() ||
		cp.ValueType() != orig.ValueType() {
		t.Errorf("Copy(): scalar fields mismatch")
	}
	if cp.Pipe == orig.Pipe {
		t.Error("Copy(): Pipe not deep-copied")
	}
	if cp.List == orig.List {
		t.Error("Copy(): List not deep-copied")
	}
}

// --- extNodeChildren / childrenTable -------------------------------------

func TestChildrenTable(t *testing.T) {
	if got := childrenTable(nil); got != nil {
		t.Errorf("nil receiver: got %v, want nil", got)
	}

	empty := &TableNode{NodeType: NodeTable}
	if got := childrenTable(empty); !reflect.DeepEqual(got, []Node{}) {
		t.Errorf("empty table: got %v, want []", got)
	}

	pipe := &PipeNode{NodeType: NodePipe}
	list := &ListNode{NodeType: NodeList}
	full := &TableNode{NodeType: NodeTable, Pipe: pipe, List: list}
	if got := childrenTable(full); !reflect.DeepEqual(got, []Node{pipe, list}) {
		t.Errorf("pipe+list: got %v, want [pipe list]", got)
	}
}

func TestExtNodeChildren_Allseas(t *testing.T) {
	// Non-extension node: dispatcher returns nil.
	if got := extNodeChildren(&FieldNode{NodeType: NodeField}); got != nil {
		t.Errorf("non-extension node: got %v, want nil", got)
	}
	// TableNode is dispatched to childrenTable.
	pipe := &PipeNode{NodeType: NodePipe}
	list := &ListNode{NodeType: NodeList}
	tn := &TableNode{NodeType: NodeTable, Pipe: pipe, List: list}
	if got := extNodeChildren(tn); !reflect.DeepEqual(got, []Node{pipe, list}) {
		t.Errorf("TableNode: got %v, want [pipe list]", got)
	}
}

// --- extSetEndPos --------------------------------------------------------

func TestExtSetEndPos_TableNode(t *testing.T) {
	// `{{block "fmt" .}}body{{end}}` — "table" is not present in source,
	// so TableNode end stays at `end`; pipe ends at the offset of `}}`.
	src := `{{block "fmt" .}}body{{end}}`
	tn := &TableNode{
		NodeType: NodeTable,
		Pos:      14,
		Pipe:     &PipeNode{NodeType: NodePipe, Pos: 14},
		List:     &ListNode{NodeType: NodeList, Pos: 17},
		Format:   "fmt",
	}
	extSetEndPos(tn, Pos(len(src)), &src)
	if got := tn.End(); got != Pos(len(src)) {
		t.Errorf("TableNode end: got %d, want %d", got, len(src))
	}
	if got := tn.Pipe.End(); got != 15 {
		t.Errorf("Pipe end: got %d, want 15", got)
	}
	if got := tn.List.End(); got != Pos(len(src)) {
		t.Errorf("List end: got %d, want %d", got, len(src))
	}
}

func TestExtSetEndPos_NonExtensionNoOp(t *testing.T) {
	f := &FieldNode{NodeType: NodeField, Pos: 3, Ident: []string{"X"}}
	text := "{{.X}}"
	extSetEndPos(f, Pos(len(text)), &text)
	if f.End() != 0 {
		t.Fatalf("FieldNode end mutated: got %d, want 0", f.End())
	}
}

// --- extAnalyseNode dispatcher -------------------------------------------

func TestExtAnalyseNode_DispatchesTable(t *testing.T) {
	pt := parseAllseasTemplate(t, `{{block "fmt" .}}body{{end}}`)
	tableParse := pt.Root.Nodes[0].(*parse.TableNode)

	ctx := &analysisCtx{funcs: map[string]*types.Func{}, vars: []*VariableNode{}}
	got, ok := extAnalyseNode(tableParse, nil, ctx).(*TableNode)
	if !ok {
		t.Fatalf("extAnalyseNode: wrong return type")
	}
	if got.Format != "fmt" {
		t.Errorf("Format: got %q, want %q", got.Format, "fmt")
	}
	if got.Pipe == nil || got.List == nil {
		t.Errorf("expected Pipe and List set: pipe=%v list=%v", got.Pipe, got.List)
	}
}

func TestExtAnalyseNode_PanicsOnUnknown(t *testing.T) {
	node := &parse.BranchNode{}
	want := fmt.Sprintf("unknown node type: %T", node)
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		if msg, _ := r.(string); msg != want {
			t.Fatalf("panic: got %v, want %q", r, want)
		}
	}()
	ctx := &analysisCtx{funcs: map[string]*types.Func{}, vars: []*VariableNode{}}
	extAnalyseNode(node, nil, ctx)
}

// --- analyseTable integration -------------------------------------------

type analyseTableTestCase struct {
	name       string
	src        string
	wantTables int
	wantFormat string
}

var analyseTableTestCases = []analyseTableTestCase{
	{
		name:       "simple block",
		src:        `{{block "fmt" .}}body{{end}}`,
		wantTables: 1,
		wantFormat: "fmt",
	},
	{
		name:       "block surrounded by text",
		src:        `prefix{{block "csv" .}}body{{end}}suffix`,
		wantTables: 1,
		wantFormat: "csv",
	},
	{
		name:       "template with no block",
		src:        `{{.X}}`,
		wantTables: 0,
		wantFormat: "",
	},
}

func TestAnalyseTable(t *testing.T) {
	for _, tc := range analyseTableTestCases {
		t.Run(tc.name, func(t *testing.T) {
			pt := parseAllseasTemplate(t, tc.src)
			tree := NewTree(*pt, map[string]*types.Func{}, nil, nil, nil)

			var tables []*TableNode
			Inspect(tree.Root, func(n Node) bool {
				if tn, ok := n.(*TableNode); ok {
					tables = append(tables, tn)
				}
				return true
			})
			if len(tables) != tc.wantTables {
				t.Fatalf("table count: got %d, want %d", len(tables), tc.wantTables)
			}
			if tc.wantTables == 0 {
				return
			}
			tn := tables[0]
			if tn.Format != tc.wantFormat {
				t.Errorf("Format: got %q, want %q", tn.Format, tc.wantFormat)
			}
			if tn.Pipe == nil || tn.Pipe.Parent() != tn {
				t.Errorf("Pipe not wired to table parent")
			}
			if tn.List == nil || tn.List.Parent() != tn {
				t.Errorf("List not wired to table parent")
			}
			if tn.Parent() != tree.Root {
				t.Errorf("Parent(): got %v, want root list", tn.Parent())
			}
		})
	}
}

func TestAnalyseTable_RestoresContextAfter(t *testing.T) {
	// analyseTable must pop the table's pipe vars and restore dotType so
	// the sibling action that follows still sees $x in scope.
	src := `{{$x := .}}{{block "fmt" .}}body{{end}}{{$x}}`
	pt := parseAllseasTemplate(t, src)
	tree := NewTree(*pt, map[string]*types.Func{}, nil, nil, nil)

	if len(tree.Root.Nodes) < 3 {
		t.Fatalf("expected at least 3 nodes in root, got %d", len(tree.Root.Nodes))
	}
	vis := VisibleVarsAt(tree.Root.Nodes[2])
	for _, v := range vis {
		if len(v.Ident) == 1 && v.Ident[0] == "$x" {
			return
		}
	}
	t.Fatalf("expected $x visible after {{block}}, got %v", vis)
}

// --- endPosTable integration --------------------------------------------

func TestEndPosTable_Integration(t *testing.T) {
	// `{{block "fmt" .}}body{{end}}` — see TestExtSetEndPos_TableNode
	// for position derivation.
	src := `{{block "fmt" .}}body{{end}}`
	pt := parseAllseasTemplate(t, src)
	tree := NewTree(*pt, map[string]*types.Func{}, nil, nil, nil)
	SetEndsForTree(tree, Pos(len(src)), &src)

	tn := firstTable(tree.Root)
	if tn == nil {
		t.Fatal("no TableNode found")
	}
	if got, want := tn.End(), Pos(len(src)); got != want {
		t.Errorf("TableNode end: got %d, want %d", got, want)
	}
	if got := tn.Pipe.End(); got != 15 {
		t.Errorf("Pipe end: got %d, want 15", got)
	}
	if got, want := tn.List.End(), Pos(len(src)); got != want {
		t.Errorf("List end: got %d, want %d", got, want)
	}
}
