package types

import "strings"

// SetEndsForTree traverses the syntax tree and sets the end positions for all nodes based on their start positions and the provided text.
func SetEndsForTree(t Tree, end Pos, text *string) {
	setEndPos(t.Root, end, text)
}

func setEndPos(node Node, end Pos, text *string) {
	switch n := node.(type) {
	case *ListNode:
		endPosList(n, end, text)
	case *TextNode:
		n.endPos = end
		if l := strings.LastIndex((*text)[n.Position():end], "{{"); l != -1 {
			n.endPos = n.Position() + Pos(l)
		}
	case *ActionNode:
		endPosAction(n, end, text)
	case *CommandNode:
		endPosCommand(n, end, text)
	case *FieldNode:
		l := 0
		for _, id := range n.Ident {
			l += 1 + len(id) // "." + segment
		}
		n.endPos = n.Position() + Pos(l)
	case *PipeNode:
		endPosPipe(n, end, text)
	case *VariableNode:
		l := 0
		for _, ident := range n.Ident {
			l += len(ident)
		}
		n.endPos = n.Position() + Pos(l)
	case *IdentifierNode:
		n.endPos = n.Position() + Pos(len(n.Ident))
	case *ChainNode:
		endPosChain(n, text)
	case *DotNode:
		n.endPos = n.Position() + 1
	case *NilNode:
		n.endPos = n.Position() + 3
	case *BoolNode:
		if n.True {
			n.endPos = n.Position() + 4
		} else {
			n.endPos = n.Position() + 5
		}
	case *NumberNode:
		n.endPos = n.Position() + Pos(len(n.Text))
	case *StringNode:
		n.endPos = n.Position() + Pos(len(n.Text)) + 2
	case *CommentNode:
		n.endPos = end
		if l := strings.LastIndex((*text)[n.Position():end], "}}"); l != -1 {
			n.endPos = n.Position() + Pos(l) + 2
		}
	case *IfNode:
		endPosIf(n, end, text)
	case *RangeNode:
		endPosRange(n, end, text)
	case *WithNode:
		endPosWith(n, end, text)
	case *TemplateNode:
		n.endPos = end
	case *BreakNode:
		n.endPos = end
		// set end pos to end of }}, if it exists
		if l := strings.Index((*text)[n.Position():end], "}}"); l != -1 {
			n.endPos = n.Position() + Pos(l) + 2
		}
	case *ContinueNode:
		n.endPos = end
		// set end pos to end of }}, if it exists
		if l := strings.Index((*text)[n.Position():end], "}}"); l != -1 {
			n.endPos = n.Position() + Pos(l) + 2
		}
	case *TableNode:
		endPosTable(n, end, text)
	}
}

func endPosWith(n *WithNode, end Pos, text *string) {
	n.endPos = end
	if l := Pos(strings.Index((*text)[n.Position():end], "with")); l != -1 {
		n.endPos = n.Position() + l + 4
	}
	l := Pos(strings.Index((*text)[n.Position():], "}}"))

	setEndPos(n.Pipe, n.Position()+l, text)
	if n.ElseList != nil {
		l = n.ElseList.Pos
	} else {
		l = end
	}
	setEndPos(n.List, l, text)
	if n.ElseList != nil {
		setEndPos(n.ElseList, end, text)
	}
}

func endPosRange(n *RangeNode, end Pos, text *string) {
	n.endPos = end
	if l := strings.Index((*text)[n.Position():end], "range"); l != -1 {
		n.endPos = n.Position() + Pos(l) + 5
	}
	l := Pos(strings.Index((*text)[n.Position():], "}}"))

	setEndPos(n.Pipe, n.Position()+l, text)
	if n.ElseList != nil {
		l = n.ElseList.Pos
	} else {
		l = end
	}
	setEndPos(n.List, l, text)
	if n.ElseList != nil {
		setEndPos(n.ElseList, end, text)
	}
}

func endPosIf(n *IfNode, end Pos, text *string) {
	n.endPos = end
	if l := strings.Index((*text)[n.Position():end], "if"); l != -1 {
		n.endPos = n.Position() + Pos(l) + 2
	}
	l := Pos(strings.Index((*text)[n.Position():], "}}"))

	setEndPos(n.Pipe, n.Position()+l, text)
	if n.ElseList != nil {
		l = n.ElseList.Pos
	} else {
		l = end
	}
	setEndPos(n.List, l, text)
	if n.ElseList != nil {
		setEndPos(n.ElseList, end, text)
	}
}

func endPosChain(n *ChainNode, text *string) {
	l := 0
	for _, field := range n.Field {
		l += len(field)
	}
	n.endPos = n.Position() + Pos(l)
	setEndPos(n.Node, n.Pos, text)
}

func endPosPipe(n *PipeNode, end Pos, text *string) {
	n.endPos = end
	for i := range len(n.Decl) - 1 {
		setEndPos(n.Decl[i], n.Decl[i+1].Position()-1, text)
	}
	if len(n.Decl) > 0 {
		setEndPos(n.Decl[len(n.Decl)-1], n.Cmds[0].Position()-1, text) // TODO: Look in text
	}
	for i := range len(n.Cmds) - 1 {
		setEndPos(n.Cmds[i], n.Cmds[i+1].Position()-1, text)
	}
	if len(n.Cmds) > 0 {
		setEndPos(n.Cmds[len(n.Cmds)-1], end, text)
	}
}

func endPosCommand(n *CommandNode, end Pos, text *string) {
	n.endPos = end
	for i := range len(n.Args) - 1 {
		setEndPos(n.Args[i], n.Args[i+1].Position()-1, text)
	}
	if len(n.Args) > 0 {
		setEndPos(n.Args[len(n.Args)-1], end, text)
	}
}

func endPosAction(n *ActionNode, end Pos, text *string) {
	n.endPos = end
	if l := strings.LastIndex((*text)[n.Position():end], "-}}"); l != -1 {
		n.endPos = n.Position() + Pos(l) + 3
		end = n.Position() + Pos(l)
	} else if l := strings.LastIndex((*text)[n.Position():end], "}}"); l != -1 {
		n.endPos = n.Position() + Pos(l) + 2
		end = n.Position() + Pos(l)
	}
	setEndPos(n.Pipe, end, text)
}

func endPosList(n *ListNode, end Pos, text *string) {
	n.endPos = end
	for i := range len(n.Nodes) - 1 {
		setEndPos(n.Nodes[i], n.Nodes[i+1].Position()-1, text)
	}
	if len(n.Nodes) > 0 {
		setEndPos(n.Nodes[len(n.Nodes)-1], end, text)
	}
}
