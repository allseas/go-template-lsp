package types

import (
	"fmt"
	"go/token"
	"go/types"

	parse "text-template-parser"
)

// TODO: check license
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Tree represents a parsed template with type information.
// It wraps the parse tree and enriches nodes with type annotations.
type Tree struct {
	Name       string                 // name of the template represented by the tree.
	ParseName  string                 // name of the top-level template during parsing, for error messages.
	Root       *ListNode              // top-level root of the tree.
	Errors     []error                // errors collected during partial parsing; only populated when Mode&ParsePartial != 0.
	End        Pos                    // position of the end of the template text; only set after parsing.
	funcs      map[string]*types.Func // available functions with their signatures
	DotType    *types.Named           // optional: type of dot context (from gotype hint)
	Pkg        *types.Package         // optional: package containing DotType
	TypeErrors []TError               // scary
	Fset       *token.FileSet         // FileSet for resolving token positions to file locations
}

// ErrorType categorizes the type of an error for customization of inspections.
type ErrorType int

const (
	// ErrorTypeInvalidField Field or method lookup failed
	ErrorTypeInvalidField = iota
	// ErrorTypeInvalidFunction Function call failed (undefined function, wrong number of args, etc)
	ErrorTypeInvalidFunction
	// ErrorTypeInvalidCommand Command execution failed (type mismatch, etc)
	ErrorTypeInvalidCommand
	// ErrorTypeInvalidRange Range over non-rangeable type
	ErrorTypeInvalidRange
	// ErrorTypeInvalidIf If condition is not boolean
	ErrorTypeInvalidIf
	// ErrorTypeInvalidWith With dot is not a struct/interface
	ErrorTypeInvalidWith
	// ErrorUndeclaredVariable Variable used without declaration
	ErrorUndeclaredVariable
	// ErrorDoubleDeclaredVariable Variable declared more than once in the same scope
	ErrorDoubleDeclaredVariable
	// ErrorTypeInvalidTemplateArg Template called with an argument of the wrong type
	ErrorTypeInvalidTemplateArg
	// Add more error types as needed
)

// TError represents a type error found during analysis, with context about the node and error type for categorization.
type TError struct {
	Node Node
	Err  string
	typ  ErrorType // for categorization
}

// ErrType returns the category of this type error.
func (e TError) ErrType() ErrorType { return e.typ }

// NewTree creates a typed tree from a parse tree, optionally with type information.
// templateInputTypes maps template names to their expected input types (from gotype hints
// on {{define}} blocks). Pass nil if template argument type checking is not needed.
func NewTree(
	parseTree parse.Tree,
	funcs map[string]*types.Func,
	dotType types.Type,
	pkg *types.Package,
	templateInputTypes map[string]types.Type,
) Tree {
	typeTree := Tree{
		Name:      parseTree.Name,
		ParseName: parseTree.ParseName,
		Errors:    parseTree.Errors,
		End:       Pos(parseTree.End),
		funcs:     funcs,
		Pkg:       pkg,
	}

	if parseTree.Root != nil {
		typeTree.Root = analyseList(parseTree.Root, nil, &analysisCtx{
			funcs:              funcs,
			dotType:            dotType,
			tree:               &typeTree,
			templateInputTypes: templateInputTypes,
			vars: []*VariableNode{
				{
					Pos:      0,
					NodeType: NodeVariable,
					Ident:    []string{"$"},
					typ:      dotType,
				},
			},
		})
	}

	return typeTree
}

// NewTreeWithType creates a typed tree with Go type information for the dot context.
// This enables hover definitions, type checking, and better completions.
//
// After creating the tree, you should call ResolveTypes() to populate type information
// on nodes that depend on context (VariableNode, FieldNode, CommandNode, etc).
func NewTreeWithType(
	parseTree parse.Tree,
	funcs map[string]*types.Func,
	dotType *types.Named,
	pkg *types.Package,
	templateInputTypes map[string]types.Type,
) Tree {
	typeTree := NewTree(parseTree, funcs, dotType, pkg, templateInputTypes)
	return typeTree
}

// analyseList converts a parse ListNode to a typed ListNode.
// ctx contains type information that flows through the analysis.
func analyseList(listNode *parse.ListNode, parent Node, ctx *analysisCtx) *ListNode {
	if listNode == nil {
		return nil
	}
	keepVars := len(ctx.vars)

	typeList := &ListNode{
		NodeType: NodeList,
		Pos:      Pos(listNode.Position()),
		Nodes:    make([]Node, len(listNode.Nodes)),
		parent:   parent,
		vars:     make([]*VariableNode, keepVars),
		typ:      ctx.dotType,
	}
	copy(typeList.vars, ctx.vars) // Preserve current variables in scope

	for i, node := range listNode.Nodes {
		typeList.Nodes[i] = analyseNode(node, typeList, ctx)
	}

	ctx.vars = ctx.vars[:keepVars] // Pop any variables declared in this list

	return typeList
}

// analyseNode converts a parse Node to a typed Node.
func analyseNode(node parse.Node, parent Node, ctx *analysisCtx) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *parse.ListNode:
		return analyseList(n, parent, ctx)

	case *parse.TextNode:
		return analyseText(n, parent)

	case *parse.ActionNode:

		return analyseAction(n, parent, ctx)

	case *parse.CommandNode:
		return analyseCommand(n, parent, ctx)

	case *parse.FieldNode:
		return analyseField(n, parent, ctx)

	case *parse.VariableNode:
		return analyseVariable(n, parent, ctx)

	case *parse.IdentifierNode:
		return analyseIdentifier(n, parent, ctx)

	case *parse.ChainNode:
		return analyseChain(n, parent, ctx)

	case *parse.DotNode:
		return analyseDot(n, parent, ctx)

	case *parse.NilNode:
		return analyseNil(n, parent, ctx)

	case *parse.BoolNode:
		return analyseBool(n, parent, ctx)

	case *parse.NumberNode:
		return analyseNumber(n, parent, ctx)

	case *parse.StringNode:
		return analyseString(n, parent, ctx)

	case *parse.CommentNode:
		return analyseComment(n, parent, ctx)

	case *parse.IfNode:
		return analyseIf(n, parent, ctx)

	case *parse.RangeNode:
		return analyseRange(n, parent, ctx)

	case *parse.WithNode:
		return analyseWith(n, parent, ctx)

	case *parse.TemplateNode:
		return analyseTemplate(n, parent, ctx)

	case *parse.BreakNode:
		return analyseBreak(n, parent, ctx)

	case *parse.ContinueNode:
		return analyseContinue(n, parent, ctx)
	case *parse.PipeNode:
		return analysePipe(n, parent, ctx)
	case *parse.UndefinedNode:
		return analyseUndefined(n, parent)
	default:
		panic(fmt.Sprintf("unknown node type: %T", node))
	}
}

func analyseUndefined(n *parse.UndefinedNode, parent Node) Node {
	return &UndefinedNode{
		NodeType: NodeUndefined,
		Pos:      Pos(n.Position()),
		parent:   parent,
		Err:      n.Err,
		str:      n.String(),
	}
}

func analyseContinue(n *parse.ContinueNode, parent Node, _ *analysisCtx) Node {
	return &ContinueNode{
		NodeType: NodeContinue,
		Pos:      Pos(n.Position()),
		Line:     n.Line,
		parent:   parent,
	}
}

func analyseBreak(n *parse.BreakNode, parent Node, _ *analysisCtx) Node {
	return &BreakNode{
		NodeType: NodeBreak,
		Pos:      Pos(n.Position()),
		Line:     n.Line,
		parent:   parent,
	}
}

func analyseTemplate(n *parse.TemplateNode, parent Node, ctx *analysisCtx) Node {
	t := &TemplateNode{
		NodeType: NodeTemplate,
		Pos:      Pos(n.Position()),
		Line:     n.Line,
		Name:     n.Name,
		parent:   parent,
	}
	t.Pipe = analysePipe(n.Pipe, t, ctx)

	// Type-check the argument against the template's declared input type (if known).
	if t.Pipe != nil && ctx.templateInputTypes != nil {
		if expectedType, ok := ctx.templateInputTypes[n.Name]; ok && expectedType != nil {
			argType := t.Pipe.ValueType()
			if argType != nil && argType.String() != expectedType.String() {
				ctx.errorf(
					t,
					ErrorTypeInvalidTemplateArg,
					"template %q expects argument of type %s, but got %s",
					n.Name,
					expectedType.String(),
					argType.String(),
				)
			}
		}
	}

	return t
}

func analyseWith(n *parse.WithNode, parent Node, ctx *analysisCtx) Node {
	w := &WithNode{
		BranchNode{
			NodeType: NodeWith,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
			parent:   parent,
		},
	}
	keepDot := ctx.dotType
	keepVars := len(ctx.vars)
	w.Pipe = analysePipe(n.Pipe, w, ctx)
	ctx.dotType = w.Pipe.typ
	w.List = analyseList(n.List, w, ctx)
	ctx.dotType = keepDot
	ctx.vars = ctx.vars[:keepVars]
	w.ElseList = analyseList(n.ElseList, w, ctx)
	return w
}

func getRangeableType(typ types.Type) types.Type {
	if typ == nil {
		return nil
	}
	switch t := typ.Underlying().(type) {
	case *types.Pointer:
		return getRangeableType(types.Unalias(t.Elem()))
	case *types.Array:
		return t.Elem()
	case *types.Slice:
		return t.Elem()
	case *types.Map:
		return t.Elem()
	case *types.Chan:
		return t.Elem()
	case *types.Basic:
		if t.Info()&types.IsInteger != 0 {
			return t
		}
		return nil
	default:
		// TODO: handle Seq
		return nil
	}
}

func (ctx *analysisCtx) errorf(node Node, typ ErrorType, format string, args ...any) {
	ctx.tree.TypeErrors = append(
		ctx.tree.TypeErrors,
		TError{
			Node: node,
			Err:  fmt.Sprintf(format, args...),
			typ:  typ, // TODO: set appropriate error type based on context
		},
	)
}

func analyseRange(n *parse.RangeNode, parent Node, ctx *analysisCtx) Node {
	r := &RangeNode{
		BranchNode{
			NodeType: NodeRange,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
			parent:   parent,
		},
	}
	keepDot := ctx.dotType
	keepVars := len(ctx.vars)
	r.Pipe = analysePipe(n.Pipe, r, ctx)
	typ := getRangeableType(r.Pipe.typ)
	if r.Pipe.typ == nil {
		ctx.errorf(r.Pipe, ErrorTypeInvalidRange, "cannot range over untyped value")
	} else if typ == nil {
		ctx.errorf(r.Pipe, ErrorTypeInvalidRange, "cannot range over type %v", r.Pipe.typ)
		ctx.dotType = nil
	} else {
		ctx.dotType = typ
		// override the range var if it was set
		if len(r.Pipe.Decl) == 1 {
			r.Pipe.Decl[0].typ = typ
		} else if len(r.Pipe.Decl) == 2 {
			r.Pipe.Decl[1].typ = typ
		}
	}
	r.List = analyseList(n.List, r, ctx)
	ctx.dotType = keepDot
	ctx.vars = ctx.vars[:keepVars]
	r.ElseList = analyseList(n.ElseList, r, ctx)
	return r
}

func analyseIf(n *parse.IfNode, parent Node, ctx *analysisCtx) Node {
	i := &IfNode{
		BranchNode{
			NodeType: NodeIf,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
			parent:   parent,
		},
	}
	keepVars := len(ctx.vars)
	i.Pipe = analysePipe(n.Pipe, i, ctx)
	i.List = analyseList(n.List, i, ctx)
	i.ElseList = analyseList(n.ElseList, i, ctx)

	ctx.vars = ctx.vars[:keepVars] // Pop any variables declared in this if block
	return i
}

func analyseComment(n *parse.CommentNode, parent Node, _ *analysisCtx) Node {
	return &CommentNode{
		NodeType: NodeComment,
		Pos:      Pos(n.Position()),
		Text:     n.Text,
		parent:   parent,
	}
}

func analyseString(n *parse.StringNode, parent Node, _ *analysisCtx) Node {
	return &StringNode{
		NodeType: NodeString,
		Pos:      Pos(n.Position()),
		Quoted:   n.Quoted,
		Text:     n.Text,
		parent:   parent,
	}
}

func analyseNumber(n *parse.NumberNode, parent Node, _ *analysisCtx) Node {
	return &NumberNode{
		NodeType:   NodeNumber,
		Pos:        Pos(n.Position()),
		IsInt:      n.IsInt,
		IsUint:     n.IsUint,
		IsFloat:    n.IsFloat,
		IsComplex:  n.IsComplex,
		Int64:      n.Int64,
		Uint64:     n.Uint64,
		Float64:    n.Float64,
		Complex128: n.Complex128,
		Text:       n.Text,
		parent:     parent,
	}
}

func analyseBool(n *parse.BoolNode, parent Node, _ *analysisCtx) Node {
	return &BoolNode{
		NodeType: NodeBool,
		Pos:      Pos(n.Position()),
		True:     n.True,
		parent:   parent,
	}
}

func analyseNil(n *parse.NilNode, parent Node, _ *analysisCtx) Node {
	return &NilNode{
		NodeType: NodeNil,
		Pos:      Pos(n.Position()),
		parent:   parent,
	}
}

func analyseDot(n *parse.DotNode, parent Node, ctx *analysisCtx) Node {
	return &DotNode{
		NodeType: NodeDot,
		Pos:      Pos(n.Position()),
		parent:   parent,
		typ:      ctx.dotType,
	}
}

func analyseChain(n *parse.ChainNode, parent Node, ctx *analysisCtx) Node {
	cn := &ChainNode{
		NodeType: NodeChain,
		Pos:      Pos(n.Position()),
		Field:    n.Field,
		parent:   parent,
	}
	keepVars := len(ctx.vars)
	cn.Node = analyseNode(n.Node, cn, ctx)
	ctx.vars = ctx.vars[:keepVars]

	baseType := getNodeType(cn.Node)
	if baseType == nil || len(n.Field) == 0 {
		return cn
	}

	if typ, ok := walkFieldChain(ctx, cn, baseType, n.Field); ok {
		cn.typ = typ
	}
	return cn
}

// walkFieldChain walks a chain of field/method names from a starting type,
// reporting any lookup errors on errNode. It returns the final type and a
// bool indicating whether the entire chain resolved successfully.
func walkFieldChain(
	ctx *analysisCtx,
	errNode Node,
	base types.Type,
	path []string,
) (types.Type, bool) {
	pkg := ctx.tree.Pkg
	currentType := base
	for _, name := range path {
		obj, _, _ := types.LookupFieldOrMethod(currentType, true, pkg, name)
		if obj == nil {
			ctx.errorf(
				errNode,
				ErrorTypeInvalidField,
				"type %s has no field or method %q",
				currentType.String(),
				name,
			)
			return nil, false
		}
		switch o := obj.(type) {
		case *types.Var:
			currentType = o.Type()
		case *types.Func:
			sig, ok := o.Type().Underlying().(*types.Signature)
			if !ok || sig.Results().Len() == 0 {
				ctx.errorf(
					errNode,
					ErrorTypeInvalidField,
					"method %q on type %s returns no values",
					name,
					currentType.String(),
				)
				return nil, false
			}
			if sig.Results().Len() > 2 {
				ctx.errorf(
					errNode,
					ErrorTypeInvalidField,
					"method %q on type %s returns more than 2 parameters",
					name,
					currentType.String(),
				)
			}
			// At(1) can be an error
			currentType = sig.Results().At(0).Type()
		default:
			ctx.errorf(
				errNode,
				ErrorTypeInvalidField,
				"unexpected object type for %q on %s",
				name,
				currentType.String(),
			)
			return nil, false
		}
	}
	return currentType, true
}

// walkFieldChainWithMethodInfo is like walkFieldChain but additionally returns an isMethod slice
// whose i-th element is true when path[i] resolves to a *types.Func (method) and false when it
// resolves to a *types.Var (struct field). On failure the returned slice is nil.
func walkFieldChainWithMethodInfo(
	ctx *analysisCtx,
	errNode Node,
	base types.Type,
	path []string,
) (types.Type, []bool, bool) {
	pkg := ctx.tree.Pkg
	currentType := base
	isMethod := make([]bool, len(path))
	for i, name := range path {
		obj, _, _ := types.LookupFieldOrMethod(currentType, true, pkg, name)
		if obj == nil {
			ctx.errorf(
				errNode,
				ErrorTypeInvalidField,
				"type %s has no field or method %q",
				currentType.String(),
				name,
			)
			return nil, nil, false
		}
		switch o := obj.(type) {
		case *types.Var:
			currentType = o.Type()
			isMethod[i] = false
		case *types.Func:
			sig, ok := o.Type().Underlying().(*types.Signature)
			if !ok || sig.Results().Len() == 0 {
				ctx.errorf(
					errNode,
					ErrorTypeInvalidField,
					"method %q on type %s returns no values",
					name,
					currentType.String(),
				)
				return nil, nil, false
			}
			if sig.Results().Len() > 2 {
				ctx.errorf(
					errNode,
					ErrorTypeInvalidField,
					"method %q on type %s returns more than 2 parameters",
					name,
					currentType.String(),
				)
			}
			currentType = sig.Results().At(0).Type()
			isMethod[i] = true
		default:
			ctx.errorf(
				errNode,
				ErrorTypeInvalidField,
				"unexpected object type for %q on %s",
				name,
				currentType.String(),
			)
			return nil, nil, false
		}
	}
	return currentType, isMethod, true
}

func analyseIdentifier(n *parse.IdentifierNode, parent Node, ctx *analysisCtx) Node {
	ident := &IdentifierNode{
		NodeType: NodeIdentifier,
		Pos:      Pos(n.Position()),
		Ident:    n.Ident,
		parent:   parent,
	}

	name := n.Ident
	if fn := ctx.funcs[name]; fn != nil {
		ident.typ = fn.Type()
		return ident
	}

	// if no function with that name -> resolve a method on the current dot type
	if named := dotAsNamed(ctx.dotType); named != nil {
		for _, m := range NamedMethods(named) {
			if m.Name == name && m.Func != nil {
				ident.typ = m.Func.Type()
				return ident
			}
		}
	}

	ctx.errorf(ident, ErrorTypeInvalidFunction, "undefined function: %s", name)
	return ident
}

func dotAsNamed(t types.Type) *types.Named {
	if t == nil {
		return nil
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	n, _ := t.(*types.Named)
	return n
}

func analyseVariable(n *parse.VariableNode, parent Node, ctx *analysisCtx) *VariableNode {
	v := &VariableNode{
		NodeType: NodeVariable,
		Pos:      Pos(n.Position()),
		Ident:    n.Ident,
		parent:   parent,
	}
	// Look up base variable in context
	var baseType types.Type
	found := false
	for i := len(ctx.vars) - 1; i >= 0; i-- {
		if ctx.vars[i].Ident[0] == n.Ident[0] {
			baseType = ctx.vars[i].typ
			found = true
			break
		}
	}
	if !found {
		return v
	}

	// $var with no field path -- type is the variable's type.
	if len(n.Ident) == 1 {
		v.typ = baseType
		return v
	}

	// $var.A.B... -- walk the field/method chain from the variable's type.
	if baseType == nil {
		return v
	}
	if typ, ok := walkFieldChain(ctx, v, baseType, n.Ident[1:]); ok {
		v.typ = typ
	}
	return v
}

func analyseField(n *parse.FieldNode, parent Node, ctx *analysisCtx) Node {
	fn := &FieldNode{
		NodeType: NodeField,
		Pos:      Pos(n.Position()),
		Ident:    n.Ident,
		parent:   parent,
	}

	if ctx.dotType == nil || len(n.Ident) == 0 {
		return fn
	}

	if typ, isMethod, ok := walkFieldChainWithMethodInfo(ctx, fn, ctx.dotType, n.Ident); ok {
		fn.typ = typ
		fn.isMethod = isMethod
	}
	return fn
}

func analyseAction(n *parse.ActionNode, parent Node, ctx *analysisCtx) Node {
	action := &ActionNode{
		NodeType: NodeAction,
		Pos:      Pos(n.Position()),
		Line:     n.Line,
		parent:   parent,
	}
	action.Pipe = analysePipe(n.Pipe, action, ctx)
	return action
}

func analyseText(n *parse.TextNode, parent Node) *TextNode {
	return &TextNode{
		NodeType: NodeText,
		Pos:      Pos(n.Position()),
		Text:     n.Text,
		parent:   parent,
	}
}

// analysePipe converts a parse PipeNode to a typed PipeNode.
func analysePipe(pipeNode *parse.PipeNode, parent Node, ctx *analysisCtx) *PipeNode {
	if pipeNode == nil {
		return nil
	}

	typePipe := &PipeNode{
		NodeType: NodePipe,
		Pos:      Pos(pipeNode.Position()),
		Line:     pipeNode.Line,
		IsAssign: pipeNode.IsAssign,
		Decl:     make([]*VariableNode, len(pipeNode.Decl)),
		Cmds:     make([]*CommandNode, len(pipeNode.Cmds)),
		parent:   parent,
	}

	// Convert commands
	for i, cmd := range pipeNode.Cmds {
		typePipe.Cmds[i] = analyseCommand(cmd, typePipe, ctx)
	}

	// The type of the pipe is literal
	if len(typePipe.Cmds) == 1 {
		typePipe.typ = getNodeType(typePipe.Cmds[0])
	} else {

		// f :: int -> string
		// g :: string -> bool
		// h :: bool -> String
		// {{ . | | h }}

		// TODO: type checking between pipe segments
		resType := getNodeType(typePipe.Cmds[len(typePipe.Cmds)-1])
		if resType == nil {
			typePipe.typ = nil
		} else {
			switch resType.Underlying().(type) {
			case *types.Signature:
				typePipe.typ = resType.Underlying().(*types.Signature).Results().At(0).Type()
			default:
				// not sure how to handle this, should be impossible, do we return nil or invalid?
				typePipe.typ = types.Typ[types.Invalid]
			}
		}
	}

	// Convert declarations
	for i, decl := range pipeNode.Decl {
		typePipe.Decl[i] = analyseVariable(decl, typePipe, ctx)
	}

	if !typePipe.IsAssign {

		if len(typePipe.Decl) == 1 {
			typePipe.Decl[0].typ = typePipe.typ
			for i := len(ctx.vars) - 1; i >= 0; i-- {
				if ctx.vars[i].Ident[0] == typePipe.Decl[0].Ident[0] {
					ctx.errorf(
						typePipe.Decl[0],
						ErrorDoubleDeclaredVariable,
						"variable %s already declared in this scope",
						ctx.vars[i].Ident[0],
					)
				}
			}
			ctx.vars = append(ctx.vars, typePipe.Decl[0])
		}

		if len(typePipe.Decl) == 2 {
			typePipe.Decl[1].typ = typePipe.typ
			ctx.vars = append(ctx.vars, typePipe.Decl[0])
			typePipe.Decl[0].typ = types.Typ[types.Uint] // unsigned int for index
			ctx.vars = append(ctx.vars, typePipe.Decl[1])
		}

	} else {
		if len(typePipe.Decl) == 1 {
			// find the variable in the context and update its type
			for i := len(ctx.vars) - 1; i >= 0; i-- {
				if ctx.vars[i].Ident[0] == typePipe.Decl[0].Ident[0] {
					if ctx.vars[i].typ != nil && typePipe.typ != nil &&
						!types.Identical(ctx.vars[i].typ, typePipe.typ) {
						ctx.errorf(
							typePipe.Decl[0],
							ErrorTypeInvalidCommand,
							"type mismatch: variable %s already has type %s, cannot assign type %s",
							ctx.vars[i].Ident[0],
							ctx.vars[i].typ.String(),
							typePipe.typ.String(),
						)
					}
					ctx.vars[i].typ = typePipe.typ
					return typePipe
				}
			}
			ctx.errorf(
				typePipe,
				ErrorUndeclaredVariable,
				"undeclared variable: %s is assigned to",
				typePipe.Decl[0].Ident[0],
			)
		}
	}

	return typePipe
}

// analyseCommand converts a parse CommandNode to a typed CommandNode.
func analyseCommand(cmdNode *parse.CommandNode, parent Node, ctx *analysisCtx) *CommandNode {
	if cmdNode == nil {
		return nil
	}

	typeCmd := &CommandNode{
		NodeType: NodeCommand,
		Pos:      Pos(cmdNode.Position()),
		Args:     make([]Node, len(cmdNode.Args)),
		parent:   parent,
	}

	for i, arg := range cmdNode.Args {
		typeCmd.Args[i] = analyseNode(arg, typeCmd, ctx)
	}

	resultType := getNodeType(typeCmd.Args[0])

	// TODO: special case for `call` builtin

	// TODO: Typecheck between the command and its arguments to see errors

	// call :: (... -> a) -> ... -> a

	if resultType != nil {
		switch t := resultType.Underlying().(type) {
		case *types.Signature:
			// If it's a function with all params provided, use the return type
			// If one param is missing, use one step curried function type
			// TODO: may be variadic ??
			if t.Params().Len() == len(typeCmd.Args)-1 {
				typeCmd.typ = t.Results().At(0).Type()
			} else if t.Params().Len() == len(typeCmd.Args) {
				typeCmd.typ = types.NewSignatureType(
					nil,
					nil,
					nil,
					types.NewTuple(t.Params().At(t.Params().Len()-1)),
					t.Results(),
					false,
				)
			}
		default:
			typeCmd.typ = resultType
		}

		return typeCmd
	}
	return typeCmd
}

// getNodeType returns the type of a node without modifying it.
func getNodeType(node Node) types.Type {
	if node == nil {
		return nil
	}
	return node.ValueType()
}

// analysisCtx carries type information through the analysis.
// It can be extended to track variable bindings, method signatures, etc.
type analysisCtx struct {
	vars               []*VariableNode        // Type of each variable in scope
	dotType            types.Type             // Current dot context type
	funcs              map[string]*types.Func // Available functions with their signatures
	tree               *Tree                  // Reference to the tree being built, for error reporting
	templateInputTypes map[string]types.Type  // Expected input type per template name (from gotype hints on {{define}} blocks)
}
