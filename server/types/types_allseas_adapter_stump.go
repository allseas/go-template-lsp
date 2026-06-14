//go:build !allseas

package types

import (
	"go/types"
	"strings"
)

// NodeType for table nodes
const (
	NodeTable = NodeText + 1000 // A table
)

// TableNode represents a table control structure in the template.
type TableNode struct {
	NodeType
	Pos
	parent Node
	Format string // The format of the table (unquoted).
	Pipe   *PipeNode
	List   *ListNode
	typ    types.Type
	isElse bool
	endPos Pos
}

func endPosTable(n *TableNode, end Pos, text *string) {
	n.endPos = end
	if l := strings.Index((*text)[n.Position():end], "table"); l != -1 {
		n.endPos = n.Position() + Pos(l) + 5
	}
	l := Pos(strings.Index((*text)[n.Position():], "}}"))
	endPosPipe(n.Pipe, n.Position()+l, text)
	endPosList(n.List, end, text)
}

// End  returns the position of the end of the table node.
func (t *TableNode) End() Pos {
	return t.endPos
}

// ValueType returns the type of the value produced by the table node, which is the result of executing the table's pipe.
func (t *TableNode) ValueType() types.Type {
	return t.typ
}

// Parent returns the parent node of the table node.
func (t *TableNode) Parent() Node {
	return t.parent
}

// IsElseList returns true if the table node is inside an else list, false otherwise.
func (t *TableNode) IsElseList() bool {
	return t.isElse
}

// Copy creates a deep copy of the table node, including its pipe and list.
func (t *TableNode) Copy() Node {
	return &TableNode{
		NodeType: NodeTable,
		Pos:      t.Pos,
		parent:   t.parent,
		Format:   t.Format,
		Pipe:     t.Pipe.CopyPipe(),
		List:     t.List.CopyList(),
		typ:      t.typ,
	}
}
func (t *TableNode) String() string              { return "{{block (table extension)}}" }
func (t *TableNode) writeTo(sb *strings.Builder) { sb.WriteString("{{block (table extension)}}") }

// childrenTable returns the direct children of a TableNode for tree traversal.
func childrenTable(t *TableNode) []Node {
	if t == nil {
		return nil
	}
	children := make([]Node, 0, 2)
	if t.Pipe != nil {
		children = append(children, t.Pipe)
	}
	if t.List != nil {
		children = append(children, t.List)
	}
	return children
}
