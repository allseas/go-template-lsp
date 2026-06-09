package types

// Testing Node find

type nodeFindTestCase struct {
	name     string
	root     Node
	offset   Pos
	wantNode Node
}

var nodeFindTestCases = func() []nodeFindTestCase {
	var cases []nodeFindTestCase

	cases = append(cases, nodeFindTestCase{
		name:     "nil root returns nil",
		root:     nil,
		offset:   10,
		wantNode: nil,
	})

	rootOnly := &ListNode{NodeType: NodeList, Pos: 0}
	cases = append(cases, nodeFindTestCase{
		name:     "root only",
		root:     rootOnly,
		offset:   99,
		wantNode: rootOnly,
	})

	{
		cmd := &CommandNode{NodeType: NodeCommand, Pos: 10}
		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{cmd}}
		cmd.parent = root
		cases = append(cases, nodeFindTestCase{
			name:     "offset before first child",
			root:     root,
			offset:   5,
			wantNode: root,
		})
	}

	{
		field := &FieldNode{NodeType: NodeField, Pos: 10, Ident: []string{"X"}}
		cmd := &CommandNode{NodeType: NodeCommand, Pos: 5, Args: []Node{field}}
		field.parent = cmd
		pipe := &PipeNode{NodeType: NodePipe, Pos: 5, Cmds: []*CommandNode{cmd}}
		cmd.parent = pipe
		action := &ActionNode{NodeType: NodeAction, Pos: 5, Line: 1, Pipe: pipe}
		pipe.parent = action
		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{action}}
		action.parent = root
		cases = append(cases, nodeFindTestCase{
			name:     "offset at shared position selects deepest traversed node",
			root:     root,
			offset:   5,
			wantNode: cmd,
		})
	}

	{
		field := &FieldNode{NodeType: NodeField, Pos: 10, Ident: []string{"X"}}
		cmd := &CommandNode{NodeType: NodeCommand, Pos: 5, Args: []Node{field}}
		field.parent = cmd
		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{cmd}}
		cmd.parent = root
		cases = append(cases, nodeFindTestCase{
			name:     "offset between parent and child",
			root:     root,
			offset:   7,
			wantNode: cmd,
		})
	}

	{
		field := &FieldNode{NodeType: NodeField, Pos: 10, Ident: []string{"X"}}
		cmd := &CommandNode{NodeType: NodeCommand, Pos: 5, Args: []Node{field}}
		field.parent = cmd
		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{cmd}}
		cmd.parent = root
		cases = append(cases, nodeFindTestCase{
			name:     "offset at deepest node",
			root:     root,
			offset:   10,
			wantNode: field,
		})
	}

	{
		field := &FieldNode{NodeType: NodeField, Pos: 10, Ident: []string{"X"}}
		cmd := &CommandNode{NodeType: NodeCommand, Pos: 5, Args: []Node{field}}
		field.parent = cmd
		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{cmd}}
		cmd.parent = root
		cases = append(cases, nodeFindTestCase{
			name:     "offset beyond all nodes",
			root:     root,
			offset:   100,
			wantNode: field,
		})
	}

	{
		pipe1 := &PipeNode{NodeType: NodePipe, Pos: 3}
		action1 := &ActionNode{NodeType: NodeAction, Pos: 3, Line: 1, Pipe: pipe1}
		pipe1.parent = action1

		pipe2 := &PipeNode{NodeType: NodePipe, Pos: 20}
		action2 := &ActionNode{NodeType: NodeAction, Pos: 20, Line: 1, Pipe: pipe2}
		pipe2.parent = action2

		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{action1, action2}}
		action1.parent = root
		action2.parent = root

		cases = append(cases, nodeFindTestCase{
			name:     "offset selects first sibling subtree",
			root:     root,
			offset:   10,
			wantNode: pipe1,
		})
		cases = append(cases, nodeFindTestCase{
			name:     "offset selects second sibling subtree",
			root:     root,
			offset:   20,
			wantNode: pipe2,
		})
	}

	{
		innerField := &FieldNode{NodeType: NodeField, Pos: 15, Ident: []string{"Item"}}
		innerCmd := &CommandNode{NodeType: NodeCommand, Pos: 13, Args: []Node{innerField}}
		innerField.parent = innerCmd
		innerList := &ListNode{NodeType: NodeList, Pos: 10, Nodes: []Node{innerCmd}}
		innerCmd.parent = innerList

		rangePipe := &PipeNode{NodeType: NodePipe, Pos: 5}
		rn := &RangeNode{BranchNode{NodeType: NodeRange, Pos: 3, Pipe: rangePipe, List: innerList}}
		rangePipe.parent = rn
		innerList.parent = rn

		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{rn}}
		rn.parent = root

		cases = append(cases, nodeFindTestCase{
			name:     "range body: offset inside list reaches inner field",
			root:     root,
			offset:   15,
			wantNode: innerField,
		})
		cases = append(cases, nodeFindTestCase{
			name:     "range body: offset before list body reaches range pipe",
			root:     root,
			offset:   7,
			wantNode: rangePipe,
		})
	}

	return cases
}()

// Enclosing tests

type enclosingListTestCase struct {
	name     string
	node     Node
	wantList *ListNode
}

var enclosingListTestCases = func() []enclosingListTestCase {
	var cases []enclosingListTestCase

	cases = append(cases, enclosingListTestCase{
		name:     "nil input",
		node:     nil,
		wantList: nil,
	})

	self := &ListNode{NodeType: NodeList}
	cases = append(cases, enclosingListTestCase{
		name:     "node is ListNode",
		node:     self,
		wantList: self,
	})

	{
		parent := &ListNode{NodeType: NodeList}
		child := &FieldNode{NodeType: NodeField, parent: parent}
		cases = append(cases, enclosingListTestCase{
			name:     "direct parent is ListNode",
			node:     child,
			wantList: parent,
		})
	}

	{
		list := &ListNode{NodeType: NodeList}
		cmd := &CommandNode{NodeType: NodeCommand, parent: list}
		field := &FieldNode{NodeType: NodeField, parent: cmd}
		cases = append(cases, enclosingListTestCase{
			name:     "ancestor ListNode two levels up",
			node:     field,
			wantList: list,
		})
	}

	{
		cmd := &CommandNode{NodeType: NodeCommand}
		field := &FieldNode{NodeType: NodeField, parent: cmd}
		cases = append(cases, enclosingListTestCase{
			name:     "no ListNode ancestor",
			node:     field,
			wantList: nil,
		})
	}

	return cases
}()

type enclosingPipeTestCase struct {
	name     string
	node     Node
	wantPipe *PipeNode
}

var enclosingPipeTestCases = func() []enclosingPipeTestCase {
	var cases []enclosingPipeTestCase

	cases = append(cases, enclosingPipeTestCase{
		name:     "nil input",
		node:     nil,
		wantPipe: nil,
	})

	self := &PipeNode{NodeType: NodePipe}
	cases = append(cases, enclosingPipeTestCase{
		name:     "node is PipeNode",
		node:     self,
		wantPipe: self,
	})

	{
		pipe := &PipeNode{NodeType: NodePipe}
		field := &FieldNode{NodeType: NodeField, parent: pipe}
		cases = append(cases, enclosingPipeTestCase{
			name:     "direct parent is PipeNode",
			node:     field,
			wantPipe: pipe,
		})
	}

	{
		pipe := &PipeNode{NodeType: NodePipe}
		cmd := &CommandNode{NodeType: NodeCommand, parent: pipe}
		field := &FieldNode{NodeType: NodeField, parent: cmd}
		cases = append(cases, enclosingPipeTestCase{
			name:     "ancestor PipeNode two levels up",
			node:     field,
			wantPipe: pipe,
		})
	}

	{
		list := &ListNode{NodeType: NodeList}
		field := &FieldNode{NodeType: NodeField, parent: list}
		cases = append(cases, enclosingPipeTestCase{
			name:     "no PipeNode ancestor",
			node:     field,
			wantPipe: nil,
		})
	}

	return cases
}()

type enclosingCommandTestCase struct {
	name    string
	node    Node
	wantCmd *CommandNode
}

var enclosingCommandTestCases = func() []enclosingCommandTestCase {
	var cases []enclosingCommandTestCase

	cases = append(cases, enclosingCommandTestCase{
		name:    "nil input",
		node:    nil,
		wantCmd: nil,
	})

	self := &CommandNode{NodeType: NodeCommand}
	cases = append(cases, enclosingCommandTestCase{
		name:    "node is CommandNode",
		node:    self,
		wantCmd: self,
	})

	{
		cmd := &CommandNode{NodeType: NodeCommand}
		field := &FieldNode{NodeType: NodeField, parent: cmd}
		cases = append(cases, enclosingCommandTestCase{
			name:    "direct parent is CommandNode",
			node:    field,
			wantCmd: cmd,
		})
	}

	{
		cmd := &CommandNode{NodeType: NodeCommand}
		chain := &ChainNode{NodeType: NodeChain, parent: cmd}
		field := &FieldNode{NodeType: NodeField, parent: chain}
		cases = append(cases, enclosingCommandTestCase{
			name:    "ancestor CommandNode two levels up",
			node:    field,
			wantCmd: cmd,
		})
	}

	{
		pipe := &PipeNode{NodeType: NodePipe}
		field := &FieldNode{NodeType: NodeField, parent: pipe}
		cases = append(cases, enclosingCommandTestCase{
			name:    "no CommandNode ancestor",
			node:    field,
			wantCmd: nil,
		})
	}

	return cases
}()
