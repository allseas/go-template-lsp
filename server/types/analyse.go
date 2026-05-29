package types

import (
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
	Name      string                   // name of the template represented by the tree.
	ParseName string                   // name of the top-level template during parsing, for error messages.
	Root      *ListNode                // top-level root of the tree.
	Errors    []error                  // errors collected during partial parsing; only populated when Mode&ParsePartial != 0.
	funcs     []map[string]*types.Func // available functions with their signatures
	DotType   *types.Named             // optional: type of dot context (from gotype hint)
	Pkg       *types.Package           // optional: package containing DotType
}

// NewTree creates a typed tree from a parse tree, optionally with type information.
func NewTree(parseTree parse.Tree, funcs []map[string]*types.Func) Tree {
	typeTree := Tree{
		Name:      parseTree.Name,
		ParseName: parseTree.ParseName,
		Errors:    parseTree.Errors,
		funcs:     funcs,
	}

	if parseTree.Root != nil {
		typeTree.Root = analyseList(parseTree.Root, &analysisCtx{funcs: funcs})
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
	funcs []map[string]*types.Func,
	dotType *types.Named,
	pkg *types.Package,
) Tree {
	typeTree := NewTree(parseTree, funcs)
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
func analyseList(listNode *parse.ListNode, ctx *analysisCtx) *ListNode {
	if listNode == nil {
		return nil
	}

	typeList := &ListNode{
		NodeType: NodeList,
		Pos:      Pos(listNode.Position()),
		Nodes:    make([]Node, len(listNode.Nodes)),
	}

	for i, node := range listNode.Nodes {
		typeList.Nodes[i] = analyseNode(node, ctx)
	}

	return typeList
}

// analyseNode converts a parse Node to a typed Node.
func analyseNode(node parse.Node, ctx *analysisCtx) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *parse.ListNode:
		return analyseList(n, ctx)

	case *parse.TextNode:
		return &TextNode{
			NodeType: NodeText,
			Pos:      Pos(n.Position()),
			Text:     n.Text,
		}

	case *parse.ActionNode:
		return &ActionNode{
			NodeType: NodeAction,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
			Pipe:     analysePipe(n.Pipe, ctx),
		}

	case *parse.CommandNode:
		return analyseCommand(n, ctx)

	case *parse.FieldNode:
		return &FieldNode{
			NodeType: NodeField,
			Pos:      Pos(n.Position()),
			Ident:    n.Ident,
			// TODO: add typ logic here
		}

	case *parse.VariableNode:
		return &VariableNode{
			NodeType: NodeVariable,
			Pos:      Pos(n.Position()),
			Ident:    n.Ident,
			// TODO: add typ logic here
		}

	case *parse.IdentifierNode:
		return NewIdentifier(n.Ident).SetPos(Pos(n.Position()))

	case *parse.ChainNode:
		return &ChainNode{
			NodeType: NodeChain,
			Pos:      Pos(n.Position()),
			Node:     analyseNode(n.Node, ctx),
			Field:    n.Field,
		}

	case *parse.DotNode:
		return &DotNode{
			NodeType: NodeDot,
			Pos:      Pos(n.Position()),
		}

	case *parse.NilNode:
		return &NilNode{
			NodeType: NodeNil,
			Pos:      Pos(n.Position()),
		}

	case *parse.BoolNode:
		return &BoolNode{
			NodeType: NodeBool,
			Pos:      Pos(n.Position()),
			True:     n.True,
		}

	case *parse.NumberNode:
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
		}

	case *parse.StringNode:
		return &StringNode{
			NodeType: NodeString,
			Pos:      Pos(n.Position()),
			Quoted:   n.Quoted,
			Text:     n.Text,
		}

	case *parse.CommentNode:
		return &CommentNode{
			NodeType: NodeComment,
			Pos:      Pos(n.Position()),
			Text:     n.Text,
		}

	case *parse.IfNode:
		return &IfNode{
			BranchNode{
				NodeType: NodeIf,
				Pos:      Pos(n.Position()),
				Line:     n.Line,
				Pipe:     analysePipe(n.Pipe, ctx),
				List:     analyseList(n.List, ctx),
				ElseList: analyseList(n.ElseList, ctx),
			},
		}

	case *parse.RangeNode:
		return &RangeNode{
			BranchNode{
				NodeType: NodeRange,
				Pos:      Pos(n.Position()),
				Line:     n.Line,
				Pipe:     analysePipe(n.Pipe, ctx),
				List:     analyseList(n.List, ctx),
				ElseList: analyseList(n.ElseList, ctx),
			},
		}

	case *parse.WithNode:
		return &WithNode{
			BranchNode{
				NodeType: NodeWith,
				Pos:      Pos(n.Position()),
				Line:     n.Line,
				Pipe:     analysePipe(n.Pipe, ctx),
				List:     analyseList(n.List, ctx),
				ElseList: analyseList(n.ElseList, ctx),
			},
		}

	case *parse.TemplateNode:
		return &TemplateNode{
			NodeType: NodeTemplate,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
			Name:     n.Name,
			Pipe:     analysePipe(n.Pipe, ctx),
		}

	case *parse.BreakNode:
		return &BreakNode{
			NodeType: NodeBreak,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
		}

	case *parse.ContinueNode:
		return &ContinueNode{
			NodeType: NodeContinue,
			Pos:      Pos(n.Position()),
			Line:     n.Line,
		}

	default:
		// Unknown node type
		return nil
	}
}

// analysePipe converts a parse PipeNode to a typed PipeNode.
func analysePipe(pipeNode *parse.PipeNode, ctx *analysisCtx) *PipeNode {
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

	// Convert declarations
	for i, decl := range pipeNode.Decl {
		typePipe.Decl[i] = &VariableNode{
			NodeType: NodeVariable,
			Pos:      Pos(decl.Position()),
			Ident:    decl.Ident,
		}
	}

	// Convert commands
	for i, cmd := range pipeNode.Cmds {
		typePipe.Cmds[i] = analyseCommand(cmd, ctx)
	}

	// The type of the pipe is literal
	if len(typePipe.Decl) == 1 {
		typePipe.typ = getNodeType(typePipe.Decl[0])
		return typePipe
	}

	resType := getNodeType(typePipe.Cmds[len(typePipe.Cmds)-1])
	switch resType.Underlying().(type) {
	case *types.Signature:
		typePipe.typ = resType.Underlying().(*types.Signature).Results().At(0).Type()
	default:
		// not sure how to handle this, should be impossible, do we return nil or invalid?
		typePipe.typ = types.Typ[types.Invalid]
	}

	//TODO: update vars

	return typePipe
}

// analyseCommand converts a parse CommandNode to a typed CommandNode.
func analyseCommand(cmdNode *parse.CommandNode, ctx *analysisCtx) *CommandNode {
	if cmdNode == nil {
		return nil
	}

	typeCmd := &CommandNode{
		NodeType: NodeCommand,
		Pos:      Pos(cmdNode.Position()),
		Args:     make([]Node, len(cmdNode.Args)),
	}

	for i, arg := range cmdNode.Args {
		typeCmd.Args[i] = analyseNode(arg, ctx)
	}

	resultType := getNodeType(typeCmd.Args[0])

	if resultType != nil {
		switch t := resultType.Underlying().(type) {
		case *types.Signature:
			// If it's a function with all params provided, use the return type
			// If one param is missing, use one step curried function type
			// TODO: may be variadic ??
			if t.Params().Len() == len(typeCmd.Args)-1 {
				typeCmd.typ = t.Results().At(0).Type()
			} else if t.Params().Len() == len(typeCmd.Args)-2 {
				typeCmd.typ = types.NewSignatureType(
					nil,
					nil,
					[]*types.TypeParam{t.TypeParams().At(t.TypeParams().Len() - 1)},
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
	varTypes map[string]types.Type    // Type of each variable in scope
	dotType  types.Type               // Current dot context type
	funcs    []map[string]*types.Func // Available functions with their signatures
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
