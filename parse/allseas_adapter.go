package parse

import "strings"

const (
	lineSpaceChars    = " \t"      // These are the space characters defined by Go itself.
	trimLineMarker    = '#'        // Attached to left/right delimiter, trims trailing spaces and tabs from preceding/following text.
	trimLineMarkerLen = Pos(1 + 1) // marker plus space before or after
)

const (
	NodeTable = NodeText + 1000 // A table
)

const itemTable = itemError + 1000

type TableNode struct {
	NodeType
	Pos
	tr     *Tree
	Line   int       // The line number in the input. Deprecated: Kept for compatibility.
	Format string    // The format of the table (unquoted).
	Pipe   *PipeNode // The command to evaluate as dot for the table.
	List   *ListNode // What to execute if the value is non-empty.
}

func (t *Tree) newTable(pos Pos, line int, format string, pipe *PipeNode, list *ListNode) *TableNode {
	return &TableNode{tr: t, NodeType: NodeTable, Pos: pos, Line: line, Format: format, Pipe: pipe, List: list}
}
func (t *TableNode) Copy() Node {
	return t.tr.newTable(t.Pos, t.Line, t.Format, t.Pipe.CopyPipe(), t.List.CopyList())
}
func (t *TableNode) String() string              { return "{{block (table extension)}}" }
func (t *TableNode) tree() *Tree                 { return t.tr }
func (t *TableNode) writeTo(sb *strings.Builder) { sb.WriteString("{{block (table extension)}}") }

// rightTrimLength returns the length of the spaces at the end of the string.
func rightTrimCharsLength(s string, trimChars string) Pos {
	return Pos(len(s) - len(strings.TrimRight(s, trimChars)))
}

func (t *Tree) parseTable(context string) (pos Pos, line int, format string, pipe *PipeNode, list *ListNode) {
	defer t.popVars(len(t.vars))
	token := t.nextNonSpace()
	format = t.parseTemplateName(token, context)
	pipe = t.pipeline(context, itemRightDelim)

	var next Node
	list, next = t.itemList()

	if next.Type() == nodeEnd && t.Mode&ParsePartial == 0 {
		return pipe.Position(), pipe.Line, format, pipe, list
	}
	if t.Mode&ParsePartial != 0 {
		err := t.recordError(t.peek().pos, "expected end; found %v", next)
		n := &UndefinedNode{NodeType: NodeUndefined, Pos: t.peek().pos, Err: err}
		list.append(n)
		return pipe.Position(), pipe.Line, format, pipe, list
	}
	t.errorf("expected end; found %s", next)
	return 0, 0, "", nil, nil
}

// Table:
//
//	{{table stringValue pipeline}}
//
// Table keyword is past.
// The format must be something that can evaluate to a string.
// The pipeline is mandatory.
func (t *Tree) tableControl() Node {

	return t.newTable(t.parseTable("table"))
}

func hasLeftLineTrimMarker(s string) bool {
	return len(s) >= 2 && s[0] == trimLineMarker && isSpace(rune(s[1]))
}
