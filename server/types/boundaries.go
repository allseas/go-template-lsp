package types

import "strings"

func SetEndsForTree(t Tree, end Pos, text *string) {
	setEndPos(t.Root, end, text)
}

func setEndPos(node Node, end Pos, text *string) {
	switch n := node.(type) {
	case *ListNode:
		n.endPos = end
		for i := range len(n.Nodes) - 1 {
			setEndPos(n.Nodes[i], n.Nodes[i+1].Position()-1, text)
		}
		if len(n.Nodes) > 0 {
			setEndPos(n.Nodes[len(n.Nodes)-1], end, text)
		}
	case *TextNode:
		n.endPos = end
		if l := strings.LastIndex((*text)[n.Position():end], "{{"); l != -1 {
			n.endPos = n.Position() + Pos(l)
		}
	case *ActionNode:
		n.endPos = end
		if l := strings.LastIndex((*text)[n.Position():end], "-}}"); l != -1 {
			n.endPos = n.Position() + Pos(l) + 3
			end = n.Position() + Pos(l)
		} else if l := strings.LastIndex((*text)[n.Position():end], "}}"); l != -1 {
			n.endPos = n.Position() + Pos(l) + 2
			end = n.Position() + Pos(l)
		}
		setEndPos(n.Pipe, end, text)
	case *CommandNode:
		n.endPos = end
		for i := range len(n.Args) - 1 {
			setEndPos(n.Args[i], n.Args[i+1].Position()-1, text)
		}
		if len(n.Args) > 0 {
			setEndPos(n.Args[len(n.Args)-1], end, text)
		}
	case *FieldNode:
		n.endPos = n.Position() + Pos(len(n.Ident))
	case *PipeNode:
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
	case *VariableNode:
		l := 0
		for _, ident := range n.Ident {
			l += len(ident)
		}
		n.endPos = n.Position() + Pos(l)
	case *IdentifierNode:
		n.endPos = n.Position() + Pos(len(n.Ident))
	case *ChainNode:
		l := 0
		for _, field := range n.Field {
			l += len(field)
		}
		n.endPos = n.Position() + Pos(l)
		setEndPos(n.Node, n.Pos, text)
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
	case *RangeNode:
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
	case *WithNode:
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
	case *TemplateNode:
		n.endPos = end
	case *BreakNode:
		n.endPos = end
	case *ContinueNode:
		n.endPos = end
	}
}
