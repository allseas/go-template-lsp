package types

import (
	"go/types"
	"strings"
	parse "text-template-parser"
)

const (
	NodeTable = NodeText + 1000 // A table
)

type TableNode struct {
	NodeType
	Pos
	parent Node
	Format string // The format of the table (unquoted).
	Pipe   *PipeNode
	List   *ListNode
	typ    types.Type
	isElse bool
}

func (t *TableNode) ValueType() types.Type {
	return t.typ
}

func (t *TableNode) Parent() Node {
	return t.parent
}

func (t *TableNode) IsElseList() bool {
	return t.isElse
}

func (t *TableNode) Copy() Node {
	return &TableNode{NodeType: NodeTable, Pos: t.Pos, parent: t.parent, Format: t.Format, Pipe: t.Pipe.CopyPipe(), List: t.List.CopyList(), typ: t.typ}
}
func (t *TableNode) String() string              { return "{{block (table extension)}}" }
func (t *TableNode) writeTo(sb *strings.Builder) { sb.WriteString("{{block (table extension)}}") }

func analyseTable(n *parse.TableNode, parent Node, ctx *analysisCtx) Node {
	table := &TableNode{NodeType: NodeTable, parent: parent, Format: n.Format, Pos: Pos(n.Pos)}
	keepVars := len(ctx.vars)
	pipe := analysePipe(n.Pipe, table, ctx)
	list := analyseList(n.List, table, ctx)
	ctx.vars = ctx.vars[:keepVars]
	table.Pipe = pipe
	table.List = list
	return table
}
