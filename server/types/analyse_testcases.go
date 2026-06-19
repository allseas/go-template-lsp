package types

import (
	"fmt"
	"go/types"
	parse "text-template-parser"
)

// anyType is the empty interface type (i.e. `any` / `interface{}`), reused
// across test cases as the expected type for unresolved nodes.
var anyType = types.NewInterfaceType(nil, nil).Complete()

type analyseNodePanicTestCase struct {
	name      string
	node      parse.Node
	wantPanic string
}

var analyseNodePanicTestCases = []analyseNodePanicTestCase{
	{
		name:      "unknown node type panics",
		node:      &parse.BranchNode{},
		wantPanic: fmt.Sprintf("unknown node type: %T", &parse.BranchNode{}),
	},
}

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

var mockPkg = types.NewPackage("example.com/mock", "mock")

var mockInnerType = func() *types.Named {
	fields := []*types.Var{
		types.NewVar(0, mockPkg, "Name", types.Typ[types.String]),
		types.NewVar(0, mockPkg, "Age", types.Typ[types.Int]),
	}
	structType := types.NewStruct(fields, nil)
	typeName := types.NewTypeName(0, mockPkg, "Inner", nil)
	return types.NewNamed(typeName, structType, nil)
}()

var mockDotType = func() *types.Named {
	fields := []*types.Var{
		types.NewVar(0, mockPkg, "X", types.Typ[types.String]),
		types.NewVar(0, mockPkg, "Y", types.Typ[types.Int]),
		types.NewVar(0, mockPkg, "Inner", mockInnerType),
		types.NewVar(0, mockPkg, "Items", types.NewSlice(types.Typ[types.String])),
	}
	structType := types.NewStruct(fields, nil)
	typeName := types.NewTypeName(0, mockPkg, "MockDot", nil)
	return types.NewNamed(typeName, structType, nil)
}()

// mockDotGreet is the *types.Func for MockDot.Greet() string, attached to
// mockDotType in init below. Pulled out so tests can reference its signature.
var mockDotGreet = types.NewFunc(0, mockPkg, "Greet",
	types.NewSignatureType(
		types.NewVar(0, mockPkg, "", mockDotType),
		nil, nil,
		types.NewTuple(),
		types.NewTuple(types.NewVar(0, mockPkg, "", types.Typ[types.String])),
		false,
	),
)

func init() {
	mockDotType.AddMethod(mockDotGreet)
}

// mockSeqType models iter.Seq[string] -- func(yield func(string) bool).
var mockSeqType = types.NewSignatureType(
	nil, nil, nil,
	types.NewTuple(types.NewVar(0, mockPkg, "yield",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(0, mockPkg, "", types.Typ[types.String])),
			types.NewTuple(types.NewVar(0, mockPkg, "", types.Typ[types.Bool])),
			false,
		),
	)),
	types.NewTuple(),
	false,
)

// mockSeq2Type models iter.Seq2[int, string] -- func(yield func(int, string) bool).
var mockSeq2Type = types.NewSignatureType(
	nil, nil, nil,
	types.NewTuple(types.NewVar(0, mockPkg, "yield",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(0, mockPkg, "", types.Typ[types.Int]),
				types.NewVar(0, mockPkg, "", types.Typ[types.String]),
			),
			types.NewTuple(types.NewVar(0, mockPkg, "", types.Typ[types.Bool])),
			false,
		),
	)),
	types.NewTuple(),
	false,
)

// mockSeqDotType is a dot type exposing iter.Seq and iter.Seq2 valued fields
// so range-over-seq behaviour can be exercised without touching mockDotType.
var mockSeqDotType = func() *types.Named {
	fields := []*types.Var{
		types.NewVar(0, mockPkg, "Seq", mockSeqType),
		types.NewVar(0, mockPkg, "Seq2", mockSeq2Type),
	}
	structType := types.NewStruct(fields, nil)
	typeName := types.NewTypeName(0, mockPkg, "MockSeqDot", nil)
	return types.NewNamed(typeName, structType, nil)
}()

var funcs = map[string]*types.Func{
	"FuncA": types.NewFunc(
		0,
		nil,
		"FuncA",
		signature([]types.Type{types.Typ[types.String]}, []types.Type{types.Typ[types.Int]}),
	), // string -> int
	"FuncB": types.NewFunc(
		0,
		nil,
		"FuncB",
		signature(
			[]types.Type{types.Typ[types.Int], types.Typ[types.Int]},
			[]types.Type{types.Typ[types.String]},
		),
	), // (int, int) -> string
	"FuncC": types.NewFunc(
		0,
		nil,
		"FuncC",
		signature(
			[]types.Type{types.Typ[types.String]},
			[]types.Type{types.Typ[types.String], types.Universe.Lookup("error").Type()},
		),
	), // (string) -> (string, error)
	"FuncD": types.NewFunc(
		0,
		nil,
		"FuncD",
		signature([]types.Type{types.Typ[types.Int]}, []types.Type{types.Typ[types.String]}),
	), // int -> string
	"GetInner": types.NewFunc(
		0,
		nil,
		"GetInner",
		signature([]types.Type{mockDotType}, []types.Type{mockInnerType}),
	), // MockDot -> Inner
	"VoidFn": types.NewFunc(
		0,
		nil,
		"VoidFn",
		signature([]types.Type{types.Typ[types.String]}, []types.Type{}),
	), // string -> ()
	"FuncAnyParam": types.NewFunc(
		0,
		nil,
		"FuncAnyParam",
		signature(
			[]types.Type{types.NewInterfaceType(nil, nil).Complete()},
			[]types.Type{types.Typ[types.String]},
		),
	), // any -> string
	"FuncPrintf": types.NewFunc(
		0,
		nil,
		"FuncPrintf",
		types.NewSignatureType(nil, nil, nil,
			types.NewTuple(
				types.NewVar(0, nil, "format", types.Typ[types.String]),
				types.NewVar(
					0,
					nil,
					"args",
					types.NewSlice(types.NewInterfaceType(nil, nil).Complete()),
				),
			),
			types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])),
			true,
		),
	), // (string, ...any) -> string
}

func varn(name string) *parse.VariableNode {
	return &parse.VariableNode{
		NodeType: parse.NodeVariable,
		Ident:    []string{name},
	}
}

// varnf builds a $var.Field1.Field2 ... style VariableNode.
func varnf(name string, fields ...string) *parse.VariableNode {
	idents := append([]string{name}, fields...)
	return &parse.VariableNode{
		NodeType: parse.NodeVariable,
		Ident:    idents,
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

func field(names ...string) *parse.FieldNode {
	return &parse.FieldNode{
		NodeType: parse.NodeField,
		Ident:    names,
	}
}

func list(args ...parse.Node) *parse.ListNode {
	return &parse.ListNode{
		NodeType: parse.NodeList,
		Nodes:    args,
	}
}

func ifN(cond *parse.PipeNode, list, elseList *parse.ListNode) *parse.IfNode {
	return &parse.IfNode{
		BranchNode: parse.BranchNode{
			NodeType: parse.NodeIf,
			Pipe:     cond,
			List:     list,
			ElseList: elseList,
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

func chain(base parse.Node, fields ...string) *parse.ChainNode {
	return &parse.ChainNode{
		NodeType: parse.NodeChain,
		Node:     base,
		Field:    fields,
	}
}

func text(s string) *parse.TextNode {
	return &parse.TextNode{
		NodeType: parse.NodeText,
		Text:     []byte(s),
	}
}

func boolN(b bool) *parse.BoolNode {
	return &parse.BoolNode{
		NodeType: parse.NodeBool,
		True:     b,
	}
}

func nilN() *parse.NilNode {
	return &parse.NilNode{
		NodeType: parse.NodeNil,
	}
}

func str(s string) *parse.StringNode {
	return &parse.StringNode{
		NodeType: parse.NodeString,
		Quoted:   "\"" + s + "\"",
		Text:     s,
	}
}

func comment(s string) *parse.CommentNode {
	return &parse.CommentNode{
		NodeType: parse.NodeComment,
		Text:     s,
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

// tvarnf builds an expected typed VariableNode with chained field idents.
func tvarnf(typ types.Type, name string, fields ...string) *VariableNode {
	idents := append([]string{name}, fields...)
	return &VariableNode{
		NodeType: NodeVariable,
		Ident:    idents,
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

func tifN(cond *PipeNode, list, elseList *ListNode) *IfNode {
	return &IfNode{
		BranchNode: BranchNode{
			NodeType: NodeIf,
			Pipe:     cond,
			List:     list,
			ElseList: elseList,
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

func tchain(typ types.Type, base Node, fields ...string) *ChainNode {
	return &ChainNode{
		NodeType: NodeChain,
		Node:     base,
		Field:    fields,
		typ:      typ,
	}
}

func ttext(s string) *TextNode {
	return &TextNode{
		NodeType: NodeText,
		Text:     []byte(s),
	}
}

func tboolN(b bool) *BoolNode {
	return &BoolNode{
		NodeType: NodeBool,
		True:     b,
	}
}

func tnilN() *NilNode {
	return &NilNode{
		NodeType: NodeNil,
	}
}

func tstr(s string) *StringNode {
	return &StringNode{
		NodeType: NodeString,
		Quoted:   "\"" + s + "\"",
		Text:     s,
	}
}

func tcomment(s string) *CommentNode {
	return &CommentNode{
		NodeType: NodeComment,
		Text:     s,
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
		name: "Valid function call",
		parseTree: tree(
			"test",
			list(actpipe(nil, coms(com(num(42)), com(ident("FuncB"), num(1))))),
		),
		resTree: ttree(
			"test",
			tlist(
				nil,
				tactpipe(
					types.Typ[types.String],
					nil,
					tcoms(
						tcom(types.Typ[types.Int], tnum(42)),
						tcom(
							types.Typ[types.String],
							tident("FuncB", funcs["FuncB"].Type()),
							tnum(1),
						),
					),
				),
			),
		),
		funcs:          funcs,
		dotType:        nil,
		pkg:            nil,
		expectedErrors: []TError{},
	},
	{
		name: "dot context switch",
		parseTree: tree("test", list(withN(
			pipe(nil, coms(com(field("X")))),
			list(actpipe(nil, coms(com(&parse.DotNode{})))),
		))),
		resTree: ttree("test", tlist(mockDotType, twithN(
			tpipe(
				types.Typ[types.String],
				nil,
				tcoms(tcom(types.Typ[types.String], tfield(types.Typ[types.String], "X"))),
			),
			tlist(
				types.Typ[types.String],
				tactpipe(
					types.Typ[types.String],
					nil,
					tcoms(tcom(types.Typ[types.String], &DotNode{typ: types.Typ[types.String]})),
				),
			),
		))),
		funcs:   funcs,
		dotType: mockDotType,
		pkg:     mockPkg,
	},
	{
		// {{ 42 | FuncD | FuncA }}  -- int -> (FuncD) -> string -> (FuncA) -> int
		name: "pipeline function chain",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(num(42)),
			com(ident("FuncD")),
			com(ident("FuncA")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.Int], nil, tcoms(
			tcom(types.Typ[types.Int], tnum(42)),
			tcom(types.Typ[types.String], tident("FuncD", funcs["FuncD"].Type())),
			tcom(types.Typ[types.Int], tident("FuncA", funcs["FuncA"].Type())),
		)))),
		funcs:          funcs,
		dotType:        nil,
		pkg:            nil,
		expectedErrors: []TError{},
	},
	{
		// {{ .Inner.Name }}  -- chained field access
		name:      "chained field access",
		parseTree: tree("test", list(actpipe(nil, coms(com(field("Inner", "Name")))))),
		resTree: ttree("test", tlist(mockDotType, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(types.Typ[types.String], tfield(types.Typ[types.String], "Inner", "Name")),
		)))),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ $x := .X }}{{ $x }}  -- variable declaration then reference
		name: "variable declaration and reference",
		parseTree: tree("test", list(
			actpipe(decls(varn("$x")), coms(com(field("X")))),
			actpipe(nil, coms(com(varn("$x")))),
		)),
		resTree: ttree("test", tlist(
			mockDotType,
			tactpipe(types.Typ[types.String],
				tdecls(tvarn("$x", types.Typ[types.String])),
				tcoms(tcom(types.Typ[types.String], tfield(types.Typ[types.String], "X"))),
			),
			tactpipe(
				types.Typ[types.String],
				nil,
				tcoms(tcom(types.Typ[types.String], tvarn("$x", types.Typ[types.String]))),
			),
		)),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ range $i, $v := .Items }}{{ $v }}{{ end }}  -- range with index + value
		name: "range with index and value vars",
		parseTree: tree("test", list(rangeN(
			pipe(decls(varn("$i"), varn("$v")), coms(com(field("Items")))),
			list(actpipe(nil, coms(com(varn("$v"))))),
		))),
		resTree: ttree("test", tlist(mockDotType, trangeN(
			tpipe(
				types.NewSlice(types.Typ[types.String]),
				tdecls(tvarn("$i", types.Typ[types.Int]), tvarn("$v", types.Typ[types.String])),
				tcoms(
					tcom(
						types.NewSlice(types.Typ[types.String]),
						tfield(types.NewSlice(types.Typ[types.String]), "Items"),
					),
				),
			),
			tlist(
				types.Typ[types.String],
				tactpipe(
					types.Typ[types.String],
					nil,
					tcoms(tcom(types.Typ[types.String], tvarn("$v", types.Typ[types.String]))),
				),
			),
		))),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ with .Inner }}{{ .Name }}{{ end }}  -- scope change via with, field on new dot
		name: "with scope change and field access",
		parseTree: tree("test", list(withN(
			pipe(nil, coms(com(field("Inner")))),
			list(actpipe(nil, coms(com(field("Name"))))),
		))),
		resTree: ttree("test", tlist(mockDotType, twithN(
			tpipe(mockInnerType, nil, tcoms(tcom(mockInnerType, tfield(mockInnerType, "Inner")))),
			tlist(
				mockInnerType,
				tactpipe(
					types.Typ[types.String],
					nil,
					tcoms(tcom(types.Typ[types.String], tfield(types.Typ[types.String], "Name"))),
				),
			),
		))),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ $a := .X }}{{ $b := .Y }}{{ $a }}{{ $b }}  -- two consecutive declarations & uses
		name: "multiple variable declarations and uses",
		parseTree: tree("test2", list(
			actpipe(decls(varn("$a")), coms(com(field("X")))),
			actpipe(decls(varn("$b")), coms(com(field("Y")))),
			actpipe(nil, coms(com(varn("$a")))),
			actpipe(nil, coms(com(varn("$b")))),
		)),
		resTree: ttree("test2", tlist(
			mockDotType,
			tactpipe(
				types.Typ[types.String],
				tdecls(tvarn("$a", types.Typ[types.String])),
				tcoms(tcom(types.Typ[types.String], tfield(types.Typ[types.String], "X"))),
			),
			tactpipe(
				types.Typ[types.Int],
				tdecls(tvarn("$b", types.Typ[types.Int])),
				tcoms(tcom(types.Typ[types.Int], tfield(types.Typ[types.Int], "Y"))),
			),
			tactpipe(
				types.Typ[types.String],
				nil,
				tcoms(tcom(types.Typ[types.String], tvarn("$a", types.Typ[types.String]))),
			),
			tactpipe(
				types.Typ[types.Int],
				nil,
				tcoms(tcom(types.Typ[types.Int], tvarn("$b", types.Typ[types.Int]))),
			),
		)),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ $d := . }}{{ $d.Inner.Name }}  -- field access through a variable
		name: "field access on variable",
		parseTree: tree("test", list(
			actpipe(decls(varn("$d")), coms(com(&parse.DotNode{}))),
			actpipe(nil, coms(com(varnf("$d", "Inner", "Name")))),
		)),
		resTree: ttree("test", tlist(
			mockDotType,
			tactpipe(
				mockDotType,
				tdecls(tvarn("$d", mockDotType)),
				tcoms(tcom(mockDotType, &DotNode{typ: mockDotType})),
			),
			tactpipe(
				types.Typ[types.String],
				nil,
				tcoms(
					tcom(
						types.Typ[types.String],
						tvarnf(types.Typ[types.String], "$d", "Inner", "Name"),
					),
				),
			),
		)),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ 2 | FuncB 1 }}  -- curried function in pipe: FuncB partially applied with 1, pipe supplies 2
		name: "curried function in pipe",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(num(2)),
			com(ident("FuncB"), num(1)),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(types.Typ[types.Int], tnum(2)),
			tcom(types.Typ[types.String], tident("FuncB", funcs["FuncB"].Type()), tnum(1)),
		)))),
		funcs:          funcs,
		dotType:        nil,
		pkg:            nil,
		expectedErrors: []TError{},
	},
	{
		// {{ 5 | FuncB 1 | FuncA }}  -- curried then fully-applied function
		name: "curried then full function in pipe",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(num(5)),
			com(ident("FuncB"), num(1)),
			com(ident("FuncA")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.Int], nil, tcoms(
			tcom(types.Typ[types.Int], tnum(5)),
			tcom(types.Typ[types.String], tident("FuncB", funcs["FuncB"].Type()), tnum(1)),
			tcom(types.Typ[types.Int], tident("FuncA", funcs["FuncA"].Type())),
		)))),
		funcs:          funcs,
		dotType:        nil,
		pkg:            nil,
		expectedErrors: []TError{},
	},
	{
		// {{ .Greet }}  -- method call on the dot context
		name:      "method call on dot",
		parseTree: tree("test", list(actpipe(nil, coms(com(field("Greet")))))),
		resTree: ttree("test", tlist(mockDotType, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(types.Typ[types.String], tfield(types.Typ[types.String], "Greet")),
		)))),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ with .Inner }}{{ $n := .Name }}{{ $n }}{{ end }}  -- declaration inside a with scope
		name: "variable declaration inside with scope",
		parseTree: tree("test", list(withN(
			pipe(nil, coms(com(field("Inner")))),
			list(
				actpipe(decls(varn("$n")), coms(com(field("Name")))),
				actpipe(nil, coms(com(varn("$n")))),
			),
		))),
		resTree: ttree("test", tlist(mockDotType, twithN(
			tpipe(mockInnerType, nil, tcoms(tcom(mockInnerType, tfield(mockInnerType, "Inner")))),
			tlist(
				mockInnerType,
				tactpipe(
					types.Typ[types.String],
					tdecls(tvarn("$n", types.Typ[types.String])),
					tcoms(tcom(types.Typ[types.String], tfield(types.Typ[types.String], "Name"))),
				),
				tactpipe(
					types.Typ[types.String],
					nil,
					tcoms(tcom(types.Typ[types.String], tvarn("$n", types.Typ[types.String]))),
				),
			),
		))),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ (. | GetInner).Name }}  -- chain over a parenthesised pipe.
		// GetInner :: MockDot -> Inner; .Name then selects string on Inner.
		name: "chain over pipe expression",
		parseTree: tree("test", list(actpipe(nil, coms(com(
			chain(
				pipe(nil, coms(
					com(&parse.DotNode{}),
					com(ident("GetInner")),
				)),
				"Name",
			),
		))))),
		resTree: ttree(
			"test",
			tlist(
				mockDotType,
				tactpipe(types.Typ[types.String], nil, tcoms(tcom(types.Typ[types.String],
					tchain(types.Typ[types.String],
						tpipe(mockInnerType, nil, tcoms(
							tcom(mockDotType, &DotNode{typ: mockDotType}),
							tcom(
								mockInnerType,
								tident("GetInner", funcs["GetInner"].Type()),
							),
						)),
						"Name",
					),
				))),
			),
		),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ if .X }}{{ . }}{{ else }}{{ . }}{{ end }}
		// dot inside both the if-list and the else-list should remain the
		// outer MockDot type (if does not introduce a new scope).
		name: "if/else preserves dot in both branches",
		parseTree: tree("test", list(ifN(
			pipe(nil, coms(com(field("X")))),
			list(actpipe(nil, coms(com(&parse.DotNode{})))),
			list(actpipe(nil, coms(com(&parse.DotNode{})))),
		))),
		resTree: ttree("test", tlist(mockDotType, tifN(
			tpipe(
				types.Typ[types.String],
				nil,
				tcoms(tcom(types.Typ[types.String], tfield(types.Typ[types.String], "X"))),
			),
			tlist(
				mockDotType,
				tactpipe(
					mockDotType,
					nil,
					tcoms(tcom(mockDotType, &DotNode{typ: mockDotType})),
				),
			),
			tlist(
				mockDotType,
				tactpipe(
					mockDotType,
					nil,
					tcoms(tcom(mockDotType, &DotNode{typ: mockDotType})),
				),
			),
		))),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// "hello"  -- a bare text node in the root list.
		name:      "text node",
		parseTree: tree("test", list(text("hello"))),
		resTree:   ttree("test", tlist(nil, ttext("hello"))),
		funcs:     funcs,
		// TextNode itself has no type; ListNode dot type stays nil with no dotType.
		expectedErrors: []TError{},
	},
	{
		// {{ true }}  -- boolean literal as an action.
		name:      "bool node literal",
		parseTree: tree("test", list(actpipe(nil, coms(com(boolN(true)))))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.Bool], nil, tcoms(
			tcom(types.Typ[types.Bool], tboolN(true)),
		)))),
		funcs:          funcs,
		expectedErrors: []TError{},
	},
	{
		// {{ nil }}  -- nil literal as an action; typed as untyped nil.
		name:      "nil node literal",
		parseTree: tree("test", list(actpipe(nil, coms(com(nilN()))))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.UntypedNil], nil, tcoms(
			tcom(types.Typ[types.UntypedNil], tnilN()),
		)))),
		funcs:          funcs,
		expectedErrors: []TError{},
	},
	{
		// {{ range .X }}{{ end }}  -- .X is string, not rangeable.
		// Expect a single ErrorTypeInvalidRange diagnostic on the pipe.
		name: "range over non-iterable type",
		parseTree: tree("test", list(rangeN(
			pipe(nil, coms(com(field("X")))),
			list(),
		))),
		resTree: ttree("test", tlist(mockDotType, trangeN(
			tpipe(
				types.Typ[types.String],
				nil,
				tcoms(tcom(types.Typ[types.String], tfield(types.Typ[types.String], "X"))),
			),
			tlist(nil),
		))),
		funcs:   funcs,
		dotType: mockDotType,
		pkg:     mockPkg,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidRange},
		},
	},
	{
		// {{ range 5 }}{{ . }}{{ end }}  -- range over an integer: dot becomes int64.
		name: "range over int",
		parseTree: tree("test", list(rangeN(
			pipe(nil, coms(com(num(5)))),
			list(actpipe(nil, coms(com(&parse.DotNode{})))),
		))),
		resTree: ttree("test", tlist(nil, trangeN(
			tpipe(
				types.Typ[types.Int],
				nil,
				tcoms(tcom(types.Typ[types.Int], tnum(5))),
			),
			tlist(
				types.Typ[types.Int],
				tactpipe(
					types.Typ[types.Int],
					nil,
					tcoms(tcom(types.Typ[types.Int], &DotNode{typ: types.Typ[types.Int]})),
				),
			),
		))),
		funcs:          funcs,
		expectedErrors: []TError{},
	},
	{
		// {{ "hi" }}  -- string literal as an action.
		name:      "string node literal",
		parseTree: tree("test", list(actpipe(nil, coms(com(str("hi")))))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(types.Typ[types.String], tstr("hi")),
		)))),
		funcs:          funcs,
		expectedErrors: []TError{},
	},
	{
		// {{/* a comment */}}  -- comment node in the root list; carries no type.
		name:           "comment node",
		parseTree:      tree("test", list(comment("/* a comment */"))),
		resTree:        ttree("test", tlist(nil, tcomment("/* a comment */"))),
		funcs:          funcs,
		expectedErrors: []TError{},
	},
	{
		// {{ FuncA 42 }}  -- FuncA :: string -> int, called with an int.
		// Expect a single ErrorTypeInvalidCommand for the bad argument type.
		// The command's result type is still FuncA's return type (int).
		name: "wrong argument type (single param)",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(ident("FuncA"), num(42)),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.Int], nil, tcoms(
			tcom(
				types.Typ[types.Int],
				tident("FuncA", funcs["FuncA"].Type()),
				tnum(42),
			),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidCommand},
		},
	},
	{
		// {{ FuncB 1 "hi" }}  -- FuncB :: (int, int) -> string, second arg
		// is a string instead of an int. Expect one ErrorTypeInvalidCommand.
		name: "wrong argument type (second of two params)",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(ident("FuncB"), num(1), str("hi")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(
				types.Typ[types.String],
				tident("FuncB", funcs["FuncB"].Type()),
				tnum(1),
				tstr("hi"),
			),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidCommand},
		},
	},
	{
		// {{ FuncB "hi" 2 }}  -- FuncB :: (int, int) -> string, first arg
		// is a string instead of an int. Expect one ErrorTypeInvalidCommand.
		name: "wrong argument type (first of two params)",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(ident("FuncB"), str("hi"), num(2)),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(
				types.Typ[types.String],
				tident("FuncB", funcs["FuncB"].Type()),
				tstr("hi"),
				tnum(2),
			),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidCommand},
		},
	},
	{
		// {{ FuncB "x" "y" }}  -- both args wrong type. Expect two
		// ErrorTypeInvalidCommand entries (one per mismatched argument).
		name: "wrong argument type (both params wrong)",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(ident("FuncB"), str("x"), str("y")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(
				types.Typ[types.String],
				tident("FuncB", funcs["FuncB"].Type()),
				tstr("x"),
				tstr("y"),
			),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidCommand},
			{typ: ErrorTypeInvalidCommand},
		},
	},
	{
		// {{ FuncA "x" "y" }}  -- FuncA expects 1 arg, given 2.
		// Expect a single ErrorArgumentNumberMismatch. The provided first
		// argument type matches so no other errors are reported.
		name: "too many arguments (single-param func)",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(ident("FuncA"), str("x"), str("y")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.Int], nil, tcoms(
			tcom(
				types.Typ[types.Int],
				tident("FuncA", funcs["FuncA"].Type()),
				tstr("x"),
				tstr("y"),
			),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorArgumentNumberMismatch},
		},
	},
	{
		// {{ FuncB 1 2 3 }}  -- FuncB expects 2 args, given 3.
		// Expect a single ErrorArgumentNumberMismatch.
		name: "too many arguments (multi-param func)",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(ident("FuncB"), num(1), num(2), num(3)),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(
				types.Typ[types.String],
				tident("FuncB", funcs["FuncB"].Type()),
				tnum(1),
				tnum(2),
				tnum(3),
			),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorArgumentNumberMismatch},
		},
	},
	{
		// {{ "hello" | FuncD }}  -- FuncD :: int -> string, but the
		// pipelined value is a string. Expect one ErrorTypeInvalidCommand.
		name: "wrong argument type via pipeline",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(str("hello")),
			com(ident("FuncD")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(types.Typ[types.String], tstr("hello")),
			tcom(types.Typ[types.String], tident("FuncD", funcs["FuncD"].Type())),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidCommand},
		},
	},
	{
		// {{ "x" | FuncB 1 }}  -- FuncB :: (int, int) -> string. The
		// literal arg supplies the first int; the pipeline supplies a
		// string for the second. Expect one ErrorTypeInvalidCommand on
		// the second argument.
		name: "wrong argument type via pipeline (multi-param func)",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(str("x")),
			com(ident("FuncB"), num(1)),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(types.Typ[types.String], tstr("x")),
			tcom(
				types.Typ[types.String],
				tident("FuncB", funcs["FuncB"].Type()),
				tnum(1),
			),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidCommand},
		},
	},
	{
		// {{ 1 | FuncB 1 2 }}  -- FuncB expects 2 args; literal supplies
		// 2 and the pipeline adds a 3rd, yielding 3 args total. Expect a
		// single ErrorArgumentNumberMismatch.
		name: "too many arguments via pipeline",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(num(1)),
			com(ident("FuncB"), num(1), num(2)),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(types.Typ[types.Int], tnum(1)),
			tcom(
				types.Typ[types.String],
				tident("FuncB", funcs["FuncB"].Type()),
				tnum(1),
				tnum(2),
			),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorArgumentNumberMismatch},
		},
	},
	{
		// {{ range .Seq }}{{ . }}{{ end }}
		// .Seq is iter.Seq[string]; ranging yields a string-typed dot
		// inside the body. No declared vars.
		name: "range over iter.Seq",
		parseTree: tree("test", list(rangeN(
			pipe(nil, coms(com(field("Seq")))),
			list(actpipe(nil, coms(com(&parse.DotNode{})))),
		))),
		resTree: ttree("test", tlist(mockSeqDotType, trangeN(
			tpipe(
				mockSeqType,
				nil,
				tcoms(tcom(mockSeqType, tfield(mockSeqType, "Seq"))),
			),
			tlist(
				types.Typ[types.String],
				tactpipe(
					types.Typ[types.String],
					nil,
					tcoms(tcom(types.Typ[types.String], &DotNode{typ: types.Typ[types.String]})),
				),
			),
		))),
		funcs:          funcs,
		dotType:        mockSeqDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ range $v := .Seq }}{{ $v }}{{ end }}
		// Single declared var binds the iter.Seq value type (string).
		name: "range over iter.Seq with value var",
		parseTree: tree("test", list(rangeN(
			pipe(decls(varn("$v")), coms(com(field("Seq")))),
			list(actpipe(nil, coms(com(varn("$v"))))),
		))),
		resTree: ttree("test", tlist(mockSeqDotType, trangeN(
			tpipe(
				mockSeqType,
				tdecls(tvarn("$v", types.Typ[types.String])),
				tcoms(tcom(mockSeqType, tfield(mockSeqType, "Seq"))),
			),
			tlist(
				types.Typ[types.String],
				tactpipe(
					types.Typ[types.String],
					nil,
					tcoms(tcom(types.Typ[types.String], tvarn("$v", types.Typ[types.String]))),
				),
			),
		))),
		funcs:          funcs,
		dotType:        mockSeqDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{ // {{ range $k, $v := .Seq2 }}{{ $v }}{{ end }}
		// iter.Seq2[int, string]: $k -> int, $v -> string; dot in body
		// becomes the value type (string).
		name: "range over iter.Seq2 with key and value vars",
		parseTree: tree("test", list(rangeN(
			pipe(decls(varn("$k"), varn("$v")), coms(com(field("Seq2")))),
			list(actpipe(nil, coms(com(varn("$v"))))),
		))),
		resTree: ttree("test", tlist(mockSeqDotType, trangeN(
			tpipe(
				mockSeq2Type,
				tdecls(
					tvarn("$k", types.Typ[types.Int]),
					tvarn("$v", types.Typ[types.String]),
				),
				tcoms(tcom(mockSeq2Type, tfield(mockSeq2Type, "Seq2"))),
			),
			tlist(
				types.Typ[types.String],
				tactpipe(
					types.Typ[types.String],
					nil,
					tcoms(tcom(types.Typ[types.String], tvarn("$v", types.Typ[types.String]))),
				),
			),
		))),
		funcs:          funcs,
		dotType:        mockSeqDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ VoidFn "x" }} -- VoidFn :: string -> (); should not panic;
		// command and pipe type are nil.
		name: "void function produces nil type",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(ident("VoidFn"), str("x")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(nil, nil, tcoms(
			tcom(nil, tident("VoidFn", funcs["VoidFn"].Type()), tstr("x")),
		)))),
		funcs:          funcs,
		dotType:        nil,
		pkg:            nil,
		expectedErrors: []TError{},
	},
	{
		// {{ .Columns }} with no dotType -- field access on unknown dot;
		// should produce any type with no error (type is simply unknown).
		name: "field access with nil dot type produces any",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(field("Columns")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(anyType, nil, tcoms(
			tcom(anyType, tfield(anyType, "Columns")),
		)))),
		funcs:          funcs,
		dotType:        nil,
		pkg:            nil,
		expectedErrors: []TError{},
	},
	{
		// {{ .BadField }} on mockDotType -- field that doesn't exist;
		// should produce any type and one ErrorTypeInvalidField.
		name: "unknown field on known type produces any",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(field("BadField")),
		)))),
		resTree: ttree("test", tlist(mockDotType, tactpipe(anyType, nil, tcoms(
			tcom(anyType, tfield(anyType, "BadField")),
		)))),
		funcs:   funcs,
		dotType: mockDotType,
		pkg:     mockPkg,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidField},
		},
	},
	{
		// {{ FuncPrintf }} -- variadic func (string, ...any) -> string called
		// with zero args; the required format arg is missing.
		// Should not panic, and should emit ErrorArgumentNumberMismatch.
		name: "variadic function missing required arg does not panic",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(ident("FuncPrintf")),
		)))),
		resTree: ttree("test", tlist(nil, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(types.Typ[types.String], tident("FuncPrintf", funcs["FuncPrintf"].Type())),
		)))),
		funcs:   funcs,
		dotType: nil,
		pkg:     nil,
		expectedErrors: []TError{
			{typ: ErrorArgumentNumberMismatch},
		},
	},
	{
		// {{ .Items | FuncAnyParam }} -- .Items is []string; FuncAnyParam takes
		// any. Concrete type passed to any param: no warning.
		name: "concrete arg to any param produces no warning",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(field("Items")),
			com(ident("FuncAnyParam")),
		)))),
		resTree: ttree("test", tlist(mockDotType, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(
				types.NewSlice(types.Typ[types.String]),
				tfield(types.NewSlice(types.Typ[types.String]), "Items"),
			),
			tcom(types.Typ[types.String], tident("FuncAnyParam", funcs["FuncAnyParam"].Type())),
		)))),
		funcs:          funcs,
		dotType:        mockDotType,
		pkg:            mockPkg,
		expectedErrors: []TError{},
	},
	{
		// {{ .BadField | FuncD }} -- .BadField is unknown (any); FuncD takes int.
		// Passing any to a concrete param: ErrorTypeInvalidField (field) +
		// ErrorUnknownType (any->int warning).
		name: "any arg to concrete param produces warning",
		parseTree: tree("test", list(actpipe(nil, coms(
			com(field("BadField")),
			com(ident("FuncD")),
		)))),
		resTree: ttree("test", tlist(mockDotType, tactpipe(types.Typ[types.String], nil, tcoms(
			tcom(anyType, tfield(anyType, "BadField")),
			tcom(types.Typ[types.String], tident("FuncD", funcs["FuncD"].Type())),
		)))),
		funcs:   funcs,
		dotType: mockDotType,
		pkg:     mockPkg,
		expectedErrors: []TError{
			{typ: ErrorTypeInvalidField},
			{typ: ErrorUnknownType},
		},
	},
}
