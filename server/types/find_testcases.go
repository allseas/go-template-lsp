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

// Inspect tests
// ai
type inspectTestCase struct {
	name        string
	root        Node
	stopAt      func(Node) bool // visitor returns false (stops recursion) when this returns true
	wantVisited []Node          // nodes in expected DFS visitation order
}

func neverStop(_ Node) bool { return false }

var inspectTestCases = func() []inspectTestCase {
	var cases []inspectTestCase

	cases = append(cases, inspectTestCase{
		name:        "nil root visits nothing",
		root:        nil,
		stopAt:      neverStop,
		wantVisited: nil,
	})

	{
		root := &ListNode{NodeType: NodeList, Pos: 0}
		cases = append(cases, inspectTestCase{
			name:        "root only",
			root:        root,
			stopAt:      neverStop,
			wantVisited: []Node{root},
		})
	}

	{
		// list -> action -> pipe -> command -> field
		field := &FieldNode{NodeType: NodeField, Pos: 10, Ident: []string{"X"}}
		cmd := &CommandNode{NodeType: NodeCommand, Pos: 5, Args: []Node{field}}
		field.parent = cmd
		pipe := &PipeNode{NodeType: NodePipe, Pos: 5, Cmds: []*CommandNode{cmd}}
		cmd.parent = pipe
		action := &ActionNode{NodeType: NodeAction, Pos: 5, Line: 1, Pipe: pipe}
		pipe.parent = action
		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{action}}
		action.parent = root
		cases = append(cases, inspectTestCase{
			name:        "single linear chain visited depth-first",
			root:        root,
			stopAt:      neverStop,
			wantVisited: []Node{root, action, pipe, cmd, field},
		})
	}

	{
		// list with two action siblings, each with a single command
		cmd1 := &CommandNode{NodeType: NodeCommand, Pos: 3}
		pipe1 := &PipeNode{NodeType: NodePipe, Pos: 3, Cmds: []*CommandNode{cmd1}}
		cmd1.parent = pipe1
		action1 := &ActionNode{NodeType: NodeAction, Pos: 3, Line: 1, Pipe: pipe1}
		pipe1.parent = action1

		cmd2 := &CommandNode{NodeType: NodeCommand, Pos: 20}
		pipe2 := &PipeNode{NodeType: NodePipe, Pos: 20, Cmds: []*CommandNode{cmd2}}
		cmd2.parent = pipe2
		action2 := &ActionNode{NodeType: NodeAction, Pos: 20, Line: 1, Pipe: pipe2}
		pipe2.parent = action2

		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{action1, action2}}
		action1.parent = root
		action2.parent = root

		cases = append(cases, inspectTestCase{
			name:        "siblings visited in order, depth-first",
			root:        root,
			stopAt:      neverStop,
			wantVisited: []Node{root, action1, pipe1, cmd1, action2, pipe2, cmd2},
		})
	}

	{
		// visitor returns false at the pipe: pipe is visited but its
		// children (the command) are not.
		cmd := &CommandNode{NodeType: NodeCommand, Pos: 5}
		pipe := &PipeNode{NodeType: NodePipe, Pos: 5, Cmds: []*CommandNode{cmd}}
		cmd.parent = pipe
		action := &ActionNode{NodeType: NodeAction, Pos: 5, Line: 1, Pipe: pipe}
		pipe.parent = action
		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{action}}
		action.parent = root
		cases = append(cases, inspectTestCase{
			name: "visitor returning false skips children",
			root: root,
			stopAt: func(n Node) bool {
				_, ok := n.(*PipeNode)
				return ok
			},
			wantVisited: []Node{root, action, pipe},
		})
	}

	{
		// visitor returns false at the root: nothing else is visited.
		cmd := &CommandNode{NodeType: NodeCommand, Pos: 5}
		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{cmd}}
		cmd.parent = root
		cases = append(cases, inspectTestCase{
			name: "visitor returning false at root skips entire subtree",
			root: root,
			stopAt: func(n Node) bool {
				_, ok := n.(*ListNode)
				return ok
			},
			wantVisited: []Node{root},
		})
	}

	{
		// range body: range -> pipe + list -> command -> field.
		// branchChildren ordering is pipe, list, elseList.
		field := &FieldNode{NodeType: NodeField, Pos: 15, Ident: []string{"Item"}}
		innerCmd := &CommandNode{NodeType: NodeCommand, Pos: 13, Args: []Node{field}}
		field.parent = innerCmd
		innerList := &ListNode{NodeType: NodeList, Pos: 10, Nodes: []Node{innerCmd}}
		innerCmd.parent = innerList

		rangePipe := &PipeNode{NodeType: NodePipe, Pos: 5}
		rn := &RangeNode{BranchNode{NodeType: NodeRange, Pos: 3, Pipe: rangePipe, List: innerList}}
		rangePipe.parent = rn
		innerList.parent = rn

		root := &ListNode{NodeType: NodeList, Pos: 0, Nodes: []Node{rn}}
		rn.parent = root

		cases = append(cases, inspectTestCase{
			name:        "range branch visits pipe then list children",
			root:        root,
			stopAt:      neverStop,
			wantVisited: []Node{root, rn, rangePipe, innerList, innerCmd, field},
		})
	}

	return cases
}()

// VisibleVarsAt tests

type visibleVarsAtTestCase struct {
	name     string
	cur      Node
	wantVars []*VariableNode
}

var visibleVarsAtTestCases = func() []visibleVarsAtTestCase {
	var cases []visibleVarsAtTestCase

	cases = append(cases, visibleVarsAtTestCase{
		name:     "nil cur returns nil",
		cur:      nil,
		wantVars: nil,
	})

	{
		// Node with no enclosing list at all.
		cmd := &CommandNode{NodeType: NodeCommand}
		field := &FieldNode{NodeType: NodeField, parent: cmd}
		cases = append(cases, visibleVarsAtTestCase{
			name:     "no enclosing list returns nil",
			cur:      field,
			wantVars: nil,
		})
	}

	{
		// Enclosing list has no inherited vars and no preceding sibling
		// actions; only sibling is the one containing cur.
		field := &FieldNode{NodeType: NodeField}
		cmd := &CommandNode{NodeType: NodeCommand, Args: []Node{field}}
		field.parent = cmd
		pipe := &PipeNode{NodeType: NodePipe, Cmds: []*CommandNode{cmd}}
		cmd.parent = pipe
		action := &ActionNode{NodeType: NodeAction, Pipe: pipe}
		pipe.parent = action
		list := &ListNode{NodeType: NodeList, Nodes: []Node{action}}
		action.parent = list
		cases = append(cases, visibleVarsAtTestCase{
			name:     "empty list, no preceding siblings",
			cur:      field,
			wantVars: []*VariableNode{},
		})
	}

	{
		// List has inherited vars (e.g. range pipe decls) and no
		// preceding sibling actions.
		dollar := &VariableNode{NodeType: NodeVariable, Ident: []string{"$"}}
		v := &VariableNode{NodeType: NodeVariable, Ident: []string{"$x"}}

		field := &FieldNode{NodeType: NodeField}
		cmd := &CommandNode{NodeType: NodeCommand, Args: []Node{field}}
		field.parent = cmd
		pipe := &PipeNode{NodeType: NodePipe, Cmds: []*CommandNode{cmd}}
		cmd.parent = pipe
		action := &ActionNode{NodeType: NodeAction, Pipe: pipe}
		pipe.parent = action
		list := &ListNode{
			NodeType: NodeList,
			Nodes:    []Node{action},
			vars:     []*VariableNode{dollar, v},
		}
		action.parent = list
		cases = append(cases, visibleVarsAtTestCase{
			name:     "inherited list vars are returned",
			cur:      field,
			wantVars: []*VariableNode{dollar, v},
		})
	}

	{
		// A preceding sibling action declares a variable; it must be
		// visible at cur. The action containing cur is the stop sibling
		// and its own decls are not included.
		declared := &VariableNode{NodeType: NodeVariable, Ident: []string{"$a"}}
		decl1Pipe := &PipeNode{
			NodeType: NodePipe,
			Decl:     []*VariableNode{declared},
		}
		action1 := &ActionNode{NodeType: NodeAction, Pipe: decl1Pipe}
		decl1Pipe.parent = action1

		// Not visible: cur's own action declares another var.
		selfDecl := &VariableNode{NodeType: NodeVariable, Ident: []string{"$b"}}
		field := &FieldNode{NodeType: NodeField}
		cmd := &CommandNode{NodeType: NodeCommand, Args: []Node{field}}
		field.parent = cmd
		curPipe := &PipeNode{
			NodeType: NodePipe,
			Decl:     []*VariableNode{selfDecl},
			Cmds:     []*CommandNode{cmd},
		}
		cmd.parent = curPipe
		action2 := &ActionNode{NodeType: NodeAction, Pipe: curPipe}
		curPipe.parent = action2

		list := &ListNode{NodeType: NodeList, Nodes: []Node{action1, action2}}
		action1.parent = list
		action2.parent = list

		cases = append(cases, visibleVarsAtTestCase{
			name:     "preceding sibling decl is visible, own decl excluded",
			cur:      field,
			wantVars: []*VariableNode{declared},
		})
	}

	{
		// Preceding sibling action has IsAssign=true: those decls must
		// NOT be added (they are assignments to existing variables).
		assigned := &VariableNode{NodeType: NodeVariable, Ident: []string{"$x"}}
		assignPipe := &PipeNode{
			NodeType: NodePipe,
			IsAssign: true,
			Decl:     []*VariableNode{assigned},
		}
		action1 := &ActionNode{NodeType: NodeAction, Pipe: assignPipe}
		assignPipe.parent = action1

		field := &FieldNode{NodeType: NodeField}
		cmd := &CommandNode{NodeType: NodeCommand, Args: []Node{field}}
		field.parent = cmd
		curPipe := &PipeNode{NodeType: NodePipe, Cmds: []*CommandNode{cmd}}
		cmd.parent = curPipe
		action2 := &ActionNode{NodeType: NodeAction, Pipe: curPipe}
		curPipe.parent = action2

		list := &ListNode{NodeType: NodeList, Nodes: []Node{action1, action2}}
		action1.parent = list
		action2.parent = list

		cases = append(cases, visibleVarsAtTestCase{
			name:     "preceding assignment is not contributed as decl",
			cur:      field,
			wantVars: []*VariableNode{},
		})
	}

	{
		// Sibling action after the stop sibling must be ignored.
		earlyDecl := &VariableNode{NodeType: NodeVariable, Ident: []string{"$a"}}
		earlyPipe := &PipeNode{NodeType: NodePipe, Decl: []*VariableNode{earlyDecl}}
		earlyAction := &ActionNode{NodeType: NodeAction, Pipe: earlyPipe}
		earlyPipe.parent = earlyAction

		field := &FieldNode{NodeType: NodeField}
		cmd := &CommandNode{NodeType: NodeCommand, Args: []Node{field}}
		field.parent = cmd
		curPipe := &PipeNode{NodeType: NodePipe, Cmds: []*CommandNode{cmd}}
		cmd.parent = curPipe
		curAction := &ActionNode{NodeType: NodeAction, Pipe: curPipe}
		curPipe.parent = curAction

		laterDecl := &VariableNode{NodeType: NodeVariable, Ident: []string{"$c"}}
		laterPipe := &PipeNode{NodeType: NodePipe, Decl: []*VariableNode{laterDecl}}
		laterAction := &ActionNode{NodeType: NodeAction, Pipe: laterPipe}
		laterPipe.parent = laterAction

		list := &ListNode{NodeType: NodeList, Nodes: []Node{earlyAction, curAction, laterAction}}
		earlyAction.parent = list
		curAction.parent = list
		laterAction.parent = list

		cases = append(cases, visibleVarsAtTestCase{
			name:     "later sibling decls are not visible",
			cur:      field,
			wantVars: []*VariableNode{earlyDecl},
		})
	}

	{
		// Multiple preceding decls and inherited vars combine in order:
		// inherited first, then sibling decls in lexical order.
		inherited := &VariableNode{NodeType: NodeVariable, Ident: []string{"$"}}

		decl1 := &VariableNode{NodeType: NodeVariable, Ident: []string{"$a"}}
		pipe1 := &PipeNode{NodeType: NodePipe, Decl: []*VariableNode{decl1}}
		action1 := &ActionNode{NodeType: NodeAction, Pipe: pipe1}
		pipe1.parent = action1

		decl2 := &VariableNode{NodeType: NodeVariable, Ident: []string{"$b"}}
		pipe2 := &PipeNode{NodeType: NodePipe, Decl: []*VariableNode{decl2}}
		action2 := &ActionNode{NodeType: NodeAction, Pipe: pipe2}
		pipe2.parent = action2

		field := &FieldNode{NodeType: NodeField}
		cmd := &CommandNode{NodeType: NodeCommand, Args: []Node{field}}
		field.parent = cmd
		curPipe := &PipeNode{NodeType: NodePipe, Cmds: []*CommandNode{cmd}}
		cmd.parent = curPipe
		curAction := &ActionNode{NodeType: NodeAction, Pipe: curPipe}
		curPipe.parent = curAction

		list := &ListNode{
			NodeType: NodeList,
			Nodes:    []Node{action1, action2, curAction},
			vars:     []*VariableNode{inherited},
		}
		action1.parent = list
		action2.parent = list
		curAction.parent = list

		cases = append(cases, visibleVarsAtTestCase{
			name:     "inherited then preceding sibling decls in order",
			cur:      field,
			wantVars: []*VariableNode{inherited, decl1, decl2},
		})
	}

	{
		// Non-action sibling (e.g. a TextNode) before cur is silently
		// skipped, doesn't terminate the walk.
		text := &TextNode{NodeType: NodeText, Text: []byte("hi")}
		decl := &VariableNode{NodeType: NodeVariable, Ident: []string{"$a"}}
		pipe1 := &PipeNode{NodeType: NodePipe, Decl: []*VariableNode{decl}}
		action1 := &ActionNode{NodeType: NodeAction, Pipe: pipe1}
		pipe1.parent = action1

		field := &FieldNode{NodeType: NodeField}
		cmd := &CommandNode{NodeType: NodeCommand, Args: []Node{field}}
		field.parent = cmd
		curPipe := &PipeNode{NodeType: NodePipe, Cmds: []*CommandNode{cmd}}
		cmd.parent = curPipe
		curAction := &ActionNode{NodeType: NodeAction, Pipe: curPipe}
		curPipe.parent = curAction

		list := &ListNode{NodeType: NodeList, Nodes: []Node{text, action1, curAction}}
		text.parent = list
		action1.parent = list
		curAction.parent = list

		cases = append(cases, visibleVarsAtTestCase{
			name:     "non-action siblings are skipped without terminating walk",
			cur:      field,
			wantVars: []*VariableNode{decl},
		})
	}

	return cases
}()
