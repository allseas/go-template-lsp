package types

import (
	"fmt"
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
	funcs      map[string]*types.Func // available functions with their signatures
	DotType    *types.Named           // optional: type of dot context (from gotype hint)
	Pkg        *types.Package         // optional: package containing DotType
	TypeErrors []TError               // scary
}

type ErrorType int

const (
	ErrorTypeInvalidField = iota
	ErrorTypeInvalidFunction
	ErrorTypeInvalidCommand
	ErrorTypeInvalidRange
	ErrorTypeInvalidIf
	ErrorTypeInvalidWith
	ErrorUndeclaredVariable
	ErrorDoubleDeclaredVariable
	// Add more error types as needed
)

type TError struct {
	Node Node
	Err  string
	typ  ErrorType // for categorization
}

// NewTree creates a typed tree from a parse tree, optionally with type information.
func NewTree(parseTree parse.Tree, funcs map[string]*types.Func, dotType types.Type) Tree {
	typeTree := Tree{
		Name:      parseTree.Name,
		ParseName: parseTree.ParseName,
		Errors:    parseTree.Errors,
		funcs:     funcs,
	}

	if parseTree.Root != nil {
		typeTree.Root = analyseList(parseTree.Root, nil, &analysisCtx{
			funcs:   funcs,
			dotType: dotType,
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
) Tree {
	typeTree := NewTree(parseTree, funcs, dotType)
	typeTree.DotType = dotType
	typeTree.Pkg = pkg

	// Populate types on context-dependent nodes
	if typeTree.Root != nil && dotType != nil {
		resolveNodeTypes(typeTree.Root, dotType, &resolveCtx{pkg: pkg})
	}

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

	default:
		// Unknown node type
		return nil
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

// TODO
func analyseTemplate(n *parse.TemplateNode, parent Node, ctx *analysisCtx) Node {
	return &TemplateNode{
		NodeType: NodeTemplate,
		Pos:      Pos(n.Position()),
		Line:     n.Line,
		Name:     n.Name,
		Pipe:     analysePipe(n.Pipe, parent, ctx),
		parent:   parent,
	}
}

func analyseWith(n *parse.WithNode, parent Node, ctx *analysisCtx) Node {
	keepDot := ctx.dotType
	keepVars := len(ctx.vars)
	pipe := analysePipe(n.Pipe, parent, ctx)
	ctx.dotType = pipe.typ
	list := analyseList(n.List, parent, ctx)
	ctx.dotType = keepDot
	ctx.vars = ctx.vars[:keepVars]
	elseList := analyseList(n.ElseList, parent, ctx)

	return &WithNode{
		BranchNode{
			NodeType: NodeWith,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
			Pipe:     pipe,
			List:     list,
			ElseList: elseList,
		},
	}
}

func getRangeableType(typ types.Type) types.Type {
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
		//TODO: handle Seq
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
	keepDot := ctx.dotType
	keepVars := len(ctx.vars)
	pipe := analysePipe(n.Pipe, parent, ctx)
	typ := getRangeableType(pipe.typ)
	if typ == nil {
		ctx.errorf(pipe, ErrorTypeInvalidRange, "cannot range over type %s", pipe.typ.String())
		ctx.dotType = nil
	}
	list := analyseList(n.List, parent, ctx)
	ctx.dotType = keepDot
	ctx.vars = ctx.vars[:keepVars]
	elseList := analyseList(n.ElseList, parent, ctx)

	return &RangeNode{
		BranchNode{
			NodeType: NodeRange,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
			Pipe:     pipe,
			List:     list,
			ElseList: elseList,
		},
	}
}

func analyseIf(n *parse.IfNode, parent Node, ctx *analysisCtx) Node {
	keepVars := len(ctx.vars)
	i := &IfNode{
		BranchNode{
			NodeType: NodeIf,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
			Pipe:     analysePipe(n.Pipe, parent, ctx),
			List:     analyseList(n.List, parent, ctx),
			ElseList: analyseList(n.ElseList, parent, ctx),
		},
	}

	ctx.vars = ctx.vars[:keepVars] // Pop any variables declared in this if block
	return i
}

func analyseComment(n *parse.CommentNode, parent Node, ctx *analysisCtx) Node {
	return &CommentNode{
		NodeType: NodeComment,
		Pos:      Pos(n.Position()),
		Text:     n.Text,
		parent:   parent,
	}
}

func analyseString(n *parse.StringNode, parent Node, ctx *analysisCtx) Node {
	return &StringNode{
		NodeType: NodeString,
		Pos:      Pos(n.Position()),
		Quoted:   n.Quoted,
		Text:     n.Text,
		parent:   parent,
	}
}

func analyseNumber(n *parse.NumberNode, parent Node, ctx *analysisCtx) Node {
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
	base := analyseNode(n.Node, parent, ctx)
	cn := &ChainNode{
		NodeType: NodeChain,
		Pos:      Pos(n.Position()),
		Node:     base,
		Field:    n.Field,
		parent:   parent,
	}

	baseType := getNodeType(base)
	if baseType == nil || len(n.Field) == 0 {
		return cn
	}

	pkg := ctx.tree.Pkg
	currentType := baseType
	for _, fieldName := range n.Field {
		obj, _, _ := types.LookupFieldOrMethod(currentType, true, pkg, fieldName)
		if obj == nil {
			ctx.errorf(cn, ErrorTypeInvalidField, "type %s has no field or method %q", currentType.String(), fieldName)
			return cn
		}
		switch o := obj.(type) {
		case *types.Var:
			currentType = o.Type()
		case *types.Func:
			sig, ok := o.Type().Underlying().(*types.Signature)
			if !ok || sig.Results().Len() == 0 {
				ctx.errorf(cn, ErrorTypeInvalidField, "method %q on type %s returns no values", fieldName, currentType.String())
				return cn
			}
			currentType = sig.Results().At(0).Type()
		default:
			ctx.errorf(cn, ErrorTypeInvalidField, "unexpected object type for %q on %s", fieldName, currentType.String())
			return cn
		}
	}
	cn.typ = currentType

	return cn
}

func analyseIdentifier(n *parse.IdentifierNode, parent Node, ctx *analysisCtx) Node {
	ident := &IdentifierNode{
		NodeType: NodeIdentifier,
		Pos:      Pos(n.Position()),
		Ident:    n.Ident,
		parent:   parent,
	}

	name := (string)(n.Ident)
	typ := ctx.funcs[name]

	if typ != nil {
		ident.typ = typ.Type()
	} else {
		ctx.errorf(ident, ErrorTypeInvalidFunction, "undefined function: %s", name)
	}

	return ident
}

func analyseVariable(n *parse.VariableNode, parent Node, ctx *analysisCtx) *VariableNode {

	return &VariableNode{
		NodeType: NodeVariable,
		Pos:      Pos(n.Position()),
		Ident:    n.Ident,
		parent:   parent,
		// TODO: add typ logic here
	}
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

	pkg := ctx.tree.Pkg
	currentType := ctx.dotType
	for _, fieldName := range n.Ident {
		obj, _, _ := types.LookupFieldOrMethod(currentType, true, pkg, fieldName)
		if obj == nil {
			ctx.errorf(fn, ErrorTypeInvalidField, "type %s has no field or method %q", currentType.String(), fieldName)
			return fn
		}
		switch o := obj.(type) {
		case *types.Var:
			currentType = o.Type()
		case *types.Func:
			sig, ok := o.Type().Underlying().(*types.Signature)
			if !ok || sig.Results().Len() == 0 {
				ctx.errorf(fn, ErrorTypeInvalidField, "method %q on type %s returns no values", fieldName, currentType.String())
				return fn
			}
			if sig.Results().Len() > 2 {
				ctx.errorf(fn, ErrorTypeInvalidField, "method %q on type %s returns more than 2 parameters", fieldName, currentType.String())
			}
			// At(1) can be an error
			currentType = sig.Results().At(0).Type()
		default:
			ctx.errorf(fn, ErrorTypeInvalidField, "unexpected object type for %q on %s", fieldName, currentType.String())
			return fn
		}
	}
	fn.typ = currentType

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
	}

	// Convert commands
	for i, cmd := range pipeNode.Cmds {
		typePipe.Cmds[i] = analyseCommand(cmd, typePipe, ctx)
	}

	// The type of the pipe is literal
	if len(typePipe.Decl) == 1 {
		typePipe.typ = getNodeType(typePipe.Decl[0])
		return typePipe
	}

	// f :: int -> string
	// g :: string -> bool
	// h :: bool -> String
	// {{ . | | h }}

	//TODO: type checking between pipe segments
	resType := getNodeType(typePipe.Cmds[len(typePipe.Cmds)-1])
	switch resType.Underlying().(type) {
	case *types.Signature:
		typePipe.typ = resType.Underlying().(*types.Signature).Results().At(0).Type()
	default:
		// not sure how to handle this, should be impossible, do we return nil or invalid?
		typePipe.typ = types.Typ[types.Invalid]
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
					ctx.errorf(typePipe.Decl[0], ErrorDoubleDeclaredVariable, "variable %s already declared in this scope", ctx.vars[i].Ident[0])
				}
			}
			ctx.vars = append(ctx.vars, typePipe.Decl[0])
		}

		if len(typePipe.Decl) == 2 {
			typePipe.Decl[0].typ = typePipe.typ
			ctx.vars = append(ctx.vars, typePipe.Decl[0])
			typePipe.Decl[1].typ = types.Typ[types.Uint] //unsigned int for index
			ctx.vars = append(ctx.vars, typePipe.Decl[1])
		}

	} else {
		if len(typePipe.Decl) == 1 {
			// find the variable in the context and update its type
			for i := len(ctx.vars) - 1; i >= 0; i-- {
				if ctx.vars[i].Ident[0] == typePipe.Decl[0].Ident[0] {
					if ctx.vars[i].typ != nil && !types.Identical(ctx.vars[i].typ, typePipe.typ) {
						ctx.errorf(typePipe.Decl[0], ErrorTypeInvalidCommand, "type mismatch: variable %s already has type %s, cannot assign type %s", ctx.vars[i].Ident[0], ctx.vars[i].typ.String(), typePipe.typ.String())
					}
					ctx.vars[i].typ = typePipe.typ
					return typePipe
				}
			}
			ctx.errorf(typePipe, ErrorUndeclaredVariable, "undeclared variable: %s is assigned to", typePipe.Decl[0].Ident[0])
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
	}

	for i, arg := range cmdNode.Args {
		typeCmd.Args[i] = analyseNode(arg, typeCmd, ctx)
	}

	resultType := getNodeType(typeCmd.Args[0])

	//TODO: special case for `call` builtin

	//TODO: Typecheck between the command and its arguments to see errors

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
	return nil
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
	// Future: Add fields for tracking type information during analysis
	// For example:
	vars    []*VariableNode        // Type of each variable in scope
	dotType types.Type             // Current dot context type
	funcs   map[string]*types.Func // Available functions with their signatures
	tree    *Tree                  // Reference to the tree being built, for error reporting
}

// resolveCtx carries context for resolving types on nodes.
type resolveCtx struct {
	pkg *types.Package
	// TODO: varTypes map[string]types.Type for variable bindings in range/with
}

// ResolveDotType resolves the type of a node within a given context.
// This is useful for hover tooltips and completions.

// Example usage:

// 	dotType := ResolveDotType(commandNode, typedTree)
// 	if dotType != nil {
// 	    // Show fields/methods of dotType in hover
// 	}

func ResolveDotType(node Node, tree *Tree) types.Type {
	if tree.DotType == nil {
		return nil
	}

	// For now, just return the root type
	// In the future, this could track through field accesses and method calls
	// to return the correct type at each point in the tree
	switch n := node.(type) {
	case *FieldNode:
		// Could resolve to the type of the field
		if len(n.Ident) > 0 && tree.DotType != nil {
			fieldName := n.Ident[0]
			// Look up field in struct
			if st, ok := tree.DotType.Underlying().(*types.Struct); ok {
				for i := 0; i < st.NumFields(); i++ {
					field := st.Field(i)
					if field.Name() == fieldName {
						return field.Type()
					}
				}
			}
		}

	case *CommandNode:
		// Could resolve to the return type of a function call
		// This would require resolving function signatures
		_ = n // TODO: implement function resolution
	}

	return nil
}

// resolveNodeTypes walks the tree and resolves types on nodes that depend on context.
// This populates ValueType() for VariableNode, FieldNode, CommandNode, ChainNode, etc.
func resolveNodeTypes(node Node, dotType *types.Named, ctx *resolveCtx) {
	if node == nil || dotType == nil {
		return
	}

	switch n := node.(type) {
	case *ListNode:
		for _, child := range n.Nodes {
			resolveNodeTypes(child, dotType, ctx)
		}

	case *FieldNode:
		// Resolve the type of the field from the current dot type
		if len(n.Ident) > 0 {
			fieldType := resolveFieldType(dotType, n.Ident)
			n.typ = fieldType
		}

	case *ChainNode:
		// First resolve the base node
		resolveNodeTypes(n.Node, dotType, ctx)

		// Then resolve through the field chain
		baseType := getNodeType(n.Node)
		if baseType != nil {
			chainType := resolveFieldChain(baseType, n.Field)
			n.typ = chainType
		}

	case *ActionNode:
		if n.Pipe != nil {
			resolvePipeTypes(n.Pipe, dotType, ctx)
		}

	case *PipeNode:
		resolvePipeTypes(n, dotType, ctx)

	case *IfNode:
		if n.Pipe != nil {
			resolvePipeTypes(n.Pipe, dotType, ctx)
		}
		if n.List != nil {
			resolveNodeTypes(n.List, dotType, ctx)
		}
		if n.ElseList != nil {
			resolveNodeTypes(n.ElseList, dotType, ctx)
		}

	case *RangeNode:
		if n.Pipe != nil {
			resolvePipeTypes(n.Pipe, dotType, ctx)
		}
		if n.List != nil {
			resolveNodeTypes(n.List, dotType, ctx)
		}
		if n.ElseList != nil {
			resolveNodeTypes(n.ElseList, dotType, ctx)
		}

	case *WithNode:
		if n.Pipe != nil {
			resolvePipeTypes(n.Pipe, dotType, ctx)
		}
		if n.List != nil {
			resolveNodeTypes(n.List, dotType, ctx)
		}
		if n.ElseList != nil {
			resolveNodeTypes(n.ElseList, dotType, ctx)
		}
	}
}

// resolvePipeTypes resolves types for a pipe and its commands.
func resolvePipeTypes(pipe *PipeNode, dotType *types.Named, ctx *resolveCtx) {
	if pipe == nil {
		return
	}

	// The pipe's type is the type of its last command
	if len(pipe.Cmds) > 0 {
		lastCmd := pipe.Cmds[len(pipe.Cmds)-1]
		resolveCommandTypes(lastCmd, dotType, ctx)
		pipe.typ = lastCmd.ValueType()
	}
}

// resolveCommandTypes resolves types for a command and its arguments.
func resolveCommandTypes(cmd *CommandNode, dotType *types.Named, ctx *resolveCtx) {
	if cmd == nil || len(cmd.Args) == 0 {
		return
	}

	// Resolve types of all arguments
	for _, arg := range cmd.Args {
		resolveNodeTypes(arg, dotType, ctx)
	}

	// The command type depends on what it does:
	// - If first arg is a function name, type is function return type
	// - If first arg is a field, type is field type
	// - Otherwise type flows through from arguments
	// TODO: Implement actual command type resolution
}

// resolveFieldType resolves the type of a field path from a struct.
func resolveFieldType(dotType *types.Named, fieldPath []string) types.Type {
	if dotType == nil || len(fieldPath) == 0 {
		return nil
	}

	currentType := dotType.Underlying()

	for _, fieldName := range fieldPath {
		// Try to get the field from a struct
		if st, ok := currentType.(*types.Struct); ok {
			found := false
			for i := 0; i < st.NumFields(); i++ {
				field := st.Field(i)
				if field.Name() == fieldName && field.Exported() {
					currentType = field.Type().Underlying()
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		} else {
			return nil
		}
	}

	return currentType
}

// resolveFieldChain resolves the type through a chain of field accesses.
func resolveFieldChain(baseType types.Type, fieldChain []string) types.Type {
	if baseType == nil || len(fieldChain) == 0 {
		return baseType
	}

	currentType := baseType.Underlying()

	for _, fieldName := range fieldChain {
		if st, ok := currentType.(*types.Struct); ok {
			found := false
			for i := 0; i < st.NumFields(); i++ {
				field := st.Field(i)
				if field.Name() == fieldName && field.Exported() {
					currentType = field.Type().Underlying()
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		} else {
			return nil
		}
	}

	return currentType
}
