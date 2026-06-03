package types

import (
	"go/types"
	parse "text-template-parser"
)

type analyseTestCase struct {
	name           string
	parseTree      parse.Tree
	funcs          map[string]*types.Func
	dotType        *types.Named
	pkg            *types.Package
	resTree        Tree
	expectedErrors []TError
}

func signature(paramTypes []types.Type, resultTypes []types.Type) *types.Signature {
	return types.NewSignatureType(nil, nil, nil, tuple(paramTypes), tuple(resultTypes), false)
}

func tuple(typeList []types.Type) *types.Tuple {
	vars := make([]*types.Var, len(typeList))
	for i, typ := range typeList {
		vars[i] = types.NewVar(0, nil, "", typ)
	}
	return types.NewTuple(vars...)
}

var funcs = map[string]*types.Func{
	"FuncA": types.NewFunc(0, nil, "FuncA", signature([]types.Type{types.Typ[types.String]}, []types.Type{types.Typ[types.Int]})),                                           // string -> int
	"FuncB": types.NewFunc(0, nil, "FuncB", signature([]types.Type{types.Typ[types.Int], types.Typ[types.Int]}, []types.Type{types.Typ[types.String]})),                     // (int, int) -> string
	"FuncC": types.NewFunc(0, nil, "FuncC", signature([]types.Type{types.Typ[types.String]}, []types.Type{types.Typ[types.String], types.Universe.Lookup("error").Type()})), // (string) -> (string, error)
	"FuncD": types.NewFunc(0, nil, "FuncD", signature([]types.Type{types.Typ[types.Int]}, []types.Type{types.Typ[types.String]})),                                           // int -> string
}

func varn(name string) *parse.VariableNode {
	return &parse.VariableNode{
		NodeType: parse.NodeVariable,
		Ident:    []string{name},
	}
}

func actpipe(decls []*parse.VariableNode, coms []*parse.CommandNode) *parse.ActionNode {
	return &parse.ActionNode{
		NodeType: parse.NodeAction,
		Pipe:     pipe(decls, coms),
		Line:     1,
	}
}

func pipe(decls []*parse.VariableNode, coms []*parse.CommandNode) *parse.PipeNode {
	return &parse.PipeNode{
		NodeType: parse.NodePipe,
		Line:     1,
		Decl:     decls,
		Cmds:     coms,
	}
}

func com(args ...parse.Node) *parse.CommandNode {
	return &parse.CommandNode{
		NodeType: parse.NodeCommand,
		Args:     args,
	}
}

func coms(args ...*parse.CommandNode) []*parse.CommandNode {
	return args
}

func decls(args ...*parse.VariableNode) []*parse.VariableNode {
	return args
}

func list(args ...parse.Node) *parse.ListNode {
	return &parse.ListNode{
		NodeType: parse.NodeList,
		Nodes:    args,
	}
}

func ifN(cond *parse.PipeNode, list *parse.ListNode) *parse.IfNode {
	return &parse.IfNode{
		BranchNode: parse.BranchNode{
			NodeType: parse.NodeIf,
			Pipe:     cond,
			List:     list,
		},
	}
}

func withN(pipe *parse.PipeNode, list *parse.ListNode) *parse.WithNode {
	return &parse.WithNode{
		BranchNode: parse.BranchNode{
			NodeType: parse.NodeWith,
			Pipe:     pipe,
			List:     list,
		},
	}
}

func rangeN(pipe *parse.PipeNode, list *parse.ListNode) *parse.RangeNode {
	return &parse.RangeNode{
		BranchNode: parse.BranchNode{
			NodeType: parse.NodeRange,
			Pipe:     pipe,
			List:     list,
		},
	}
}

func num(n int) *parse.NumberNode {
	return &parse.NumberNode{
		NodeType: parse.NodeNumber,
		IsInt:    true,
		Int64:    int64(n),
	}
}

func ident(name string) *parse.IdentifierNode {
	return &parse.IdentifierNode{
		NodeType: parse.NodeIdentifier,
		Ident:    name,
	}
}

func tree(name string, root *parse.ListNode) parse.Tree {
	return parse.Tree{
		Name: name,
		Root: root,
	}
}

// Type tree helpers (mirror the parse tree helpers above)

func tvarn(name string, typ types.Type) *VariableNode {
	return &VariableNode{
		NodeType: NodeVariable,
		Ident:    []string{name},
		typ:      typ,
	}
}

func tactpipe(typ types.Type, decls []*VariableNode, coms []*CommandNode) *ActionNode {
	return &ActionNode{
		NodeType: NodeAction,
		Pipe:     tpipe(typ, decls, coms),
		Line:     1,
	}
}

func tpipe(typ types.Type, decls []*VariableNode, coms []*CommandNode) *PipeNode {
	return &PipeNode{
		NodeType: NodePipe,
		Line:     1,
		Decl:     decls,
		Cmds:     coms,
		typ:      typ,
	}
}

func tcom(typ types.Type, args ...Node) *CommandNode {
	return &CommandNode{
		NodeType: NodeCommand,
		Args:     args,
		typ:      typ,
	}
}

func tcoms(args ...*CommandNode) []*CommandNode {
	return args
}

func tdecls(args ...*VariableNode) []*VariableNode {
	return args
}

func tlist(typ types.Type, args ...Node) *ListNode {
	return &ListNode{
		NodeType: NodeList,
		Nodes:    args,
		typ:      typ,
	}
}

func tifN(cond *PipeNode, list *ListNode) *IfNode {
	return &IfNode{
		BranchNode: BranchNode{
			NodeType: NodeIf,
			Pipe:     cond,
			List:     list,
		},
	}
}

func twithN(pipe *PipeNode, list *ListNode) *WithNode {
	return &WithNode{
		BranchNode: BranchNode{
			NodeType: NodeWith,
			Pipe:     pipe,
			List:     list,
		},
	}
}

func trangeN(pipe *PipeNode, list *ListNode) *RangeNode {
	return &RangeNode{
		BranchNode: BranchNode{
			NodeType: NodeRange,
			Pipe:     pipe,
			List:     list,
		},
	}
}

func tnum(n int) *NumberNode {
	return &NumberNode{
		NodeType: NodeNumber,
		IsInt:    true,
		Int64:    int64(n),
	}
}

func tident(name string, typ types.Type) *IdentifierNode {
	return &IdentifierNode{
		NodeType: NodeIdentifier,
		Ident:    name,
		typ:      typ,
	}
}

func tfield(typ types.Type, names ...string) *FieldNode {
	return &FieldNode{
		NodeType: NodeField,
		Ident:    names,
		typ:      typ,
	}
}

func ttree(name string, root *ListNode) Tree {
	return Tree{
		Name: name,
		Root: root,
	}
}

var analyseTestCases = []analyseTestCase{
	{
		name:           "Valid function call",
		parseTree:      tree("test", list(actpipe(nil, coms(com(num(42)), com(ident("FuncB"), num(1)))))),
		resTree:        ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(tcom(types.Typ[types.Int], tnum(42)), tcom(funcs["FuncD"].Type(), tident("FuncB", funcs["FuncB"].Type()), tnum(1)))))),
		funcs:          funcs,
		dotType:        nil,
		pkg:            nil,
		expectedErrors: []TError{},
	},
}
