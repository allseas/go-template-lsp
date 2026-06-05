package types

// branchChildren returns the non-nil children of a BranchNode.
func branchChildren(b *BranchNode) []Node {
	children := make([]Node, 0, 3)
	if b.Pipe != nil {
		children = append(children, b.Pipe)
	}
	if b.List != nil {
		children = append(children, b.List)
	}
	if b.ElseList != nil {
		children = append(children, b.ElseList)
	}
	return children
}

// nodeChildren returns the direct children of n for tree traversal.
// It only returns non-nil children so the caller never encounters typed nils.
func nodeChildren(n Node) []Node {
	switch node := n.(type) {
	case *ListNode:
		if node == nil {
			return nil
		}
		return node.Nodes
	case *ActionNode:
		if node == nil || node.Pipe == nil {
			return nil
		}
		return []Node{node.Pipe}
	case *PipeNode:
		if node == nil {
			return nil
		}
		children := make([]Node, 0, len(node.Decl)+len(node.Cmds))
		for _, v := range node.Decl {
			children = append(children, v)
		}
		for _, cmd := range node.Cmds {
			children = append(children, cmd)
		}
		return children
	case *CommandNode:
		if node == nil {
			return nil
		}
		return node.Args
	case *ChainNode:
		if node == nil || node.Node == nil {
			return nil
		}
		return []Node{node.Node}
	case *IfNode:
		if node == nil {
			return nil
		}
		return branchChildren(&node.BranchNode)
	case *RangeNode:
		if node == nil {
			return nil
		}
		return branchChildren(&node.BranchNode)
	case *WithNode:
		if node == nil {
			return nil
		}
		return branchChildren(&node.BranchNode)
	case *TemplateNode:
		if node == nil || node.Pipe == nil {
			return nil
		}
		return []Node{node.Pipe}
	}
	return nil
}

// NodeFind returns the deepest node in the typed tree whose start position is
// less than or equal to offset
func NodeFind(root Node, offset Pos) Node {
	if root == nil {
		return nil
	}
	best := root
	bestPos := Pos(0)

	var walk func(n Node)
	walk = func(n Node) {
		if n == nil {
			return
		}
		pos := n.Position()
		if pos <= offset && pos >= bestPos {
			bestPos = pos
			best = n
		}
		for _, child := range nodeChildren(n) {
			walk(child)
		}
	}
	walk(root)
	return best
}

// EnclosingList returns the nearest enclosing *ListNode by walking up parents.
// Returns nil if none is found.
func EnclosingList(n Node) *ListNode {
	for cur := n; cur != nil; cur = cur.Parent() {
		if l, ok := cur.(*ListNode); ok {
			return l
		}
	}
	return nil
}

// EnclosingPipe returns the nearest enclosing *PipeNode by walking up parents.
// Returns nil if none is found.
func EnclosingPipe(n Node) *PipeNode {
	for cur := n; cur != nil; cur = cur.Parent() {
		if p, ok := cur.(*PipeNode); ok {
			return p
		}
	}
	return nil
}

// EnclosingCommand returns the nearest enclosing *CommandNode.
func EnclosingCommand(n Node) *CommandNode {
	for cur := n; cur != nil; cur = cur.Parent() {
		if c, ok := cur.(*CommandNode); ok {
			return c
		}
	}
	return nil
}
