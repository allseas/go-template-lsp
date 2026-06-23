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
	// ErrorTypeInvalidWith With dot is not a struct/interface
	ErrorTypeInvalidWith
	// ErrorUndeclaredVariable Variable used without declaration
	ErrorUndeclaredVariable
	// ErrorDoubleDeclaredVariable Variable declared more than once in the same scope
	ErrorDoubleDeclaredVariable
	// ErrorTypeInvalidTemplateArg Template called with an argument of the wrong type
	ErrorTypeInvalidTemplateArg
	// ErrorArgumentNumberMismatch function called with incorrect amound of arguments
	ErrorArgumentNumberMismatch
	// ErrorUnknownType Type information is missing or could not be resolved (most likely because of an `any` signature)
	ErrorUnknownType
	// ErrorSyntaxError Syntax error in the template, for diagnostics that come from the parser rather than type checking
	ErrorSyntaxError
	// ErrorHintLoadFailure A gotype hint type could not be loaded/resolved
	ErrorHintLoadFailure
	// ErrorTypeUnknownRangeType Range over a value whose type could not be determined
	ErrorTypeUnknownRangeType
	// ErrorTypeEmptyDefineName Define block has an empty name
	ErrorTypeEmptyDefineName
	// Add more error types as needed
)

var errorTypeNames = map[ErrorType]string{
	ErrorTypeInvalidField:       "invalidField",
	ErrorTypeInvalidFunction:    "invalidFunction",
	ErrorTypeInvalidCommand:     "invalidCommand",
	ErrorTypeInvalidRange:       "invalidRange",
	ErrorTypeInvalidWith:        "invalidWith",
	ErrorUndeclaredVariable:     "undeclaredVariable",
	ErrorDoubleDeclaredVariable: "doubleDeclaredVariable",
	ErrorTypeInvalidTemplateArg: "invalidTemplateArg",
	ErrorArgumentNumberMismatch: "argumentNumberMismatch",
	ErrorUnknownType:            "unknownType",
	ErrorSyntaxError:            "syntaxError",
	ErrorHintLoadFailure:        "hintLoadFailure",
	ErrorTypeUnknownRangeType:   "unknownRangeType",
	ErrorTypeEmptyDefineName:    "emptyDefineName",
}

// MarshalText implements encoding.TextMarshaler so ErrorType is serialized as a string (e.g. in JSON map keys).
func (e ErrorType) MarshalText() ([]byte, error) {
	if name, ok := errorTypeNames[e]; ok {
		return []byte(name), nil
	}
	return nil, fmt.Errorf("unknown ErrorType: %d", int(e))
}

// UnmarshalText implements encoding.TextUnmarshaler so ErrorType can be deserialized from a string.
func (e *ErrorType) UnmarshalText(data []byte) error {
	for k, v := range errorTypeNames {
		if v == string(data) {
			*e = k
			return nil
		}
	}
	return fmt.Errorf("unknown ErrorType: %q", string(data))
}

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
	rootVar := VariableNode{
		Pos:      0,
		NodeType: NodeVariable,
		Ident:    []string{"$"},
		typ:      dotType,
	}
	if dotType == nil {
		rootVar.typ = anyType()
		// empty interface if no dot type is provided
	}
	if parseTree.Root != nil {
		typeTree.Root = analyseList(parseTree.Root, nil, &analysisCtx{
			funcs:              funcs,
			dotType:            rootVar.typ,
			tree:               &typeTree,
			templateInputTypes: templateInputTypes,
			vars: []*VariableNode{
				&rootVar,
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
	if dotType == nil {
		return NewTree(parseTree, funcs, nil, pkg, templateInputTypes)
	}
	return NewTree(parseTree, funcs, dotType, pkg, templateInputTypes)
}

// analyseList converts a parse ListNode to a typed ListNode.
// ctx contains type information that flows through the analysis.
func analyseList(listNode *parse.ListNode, parent Node, ctx *analysisCtx) *ListNode {
	if listNode == nil {
		return nil
	}
	keepVars := len(ctx.vars)

	listTyp := ctx.dotType
	if listTyp == nil {
		listTyp = anyType()
	}
	typeList := &ListNode{
		NodeType: NodeList,
		Pos:      Pos(listNode.Position()),
		Nodes:    make([]Node, len(listNode.Nodes)),
		parent:   parent,
		vars:     make([]*VariableNode, keepVars),
		typ:      listTyp,
	}
	copy(typeList.vars, ctx.vars) // Preserve current variables in scope

	for i, node := range listNode.Nodes {
		typeList.Nodes[i] = analyseNode(node, typeList, ctx)
	}

	ctx.vars = ctx.vars[:keepVars] // Pop any variables declared in this list

	return typeList
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
			if IsEmptyInterface(argType) || IsEmptyInterface(expectedType) {
				ctx.errorf(
					t,
					ErrorUnknownType,
					"template %q expects argument of type %s, it's impossible to determine the type of the argument provided",
					n.Name,
					expectedType.String(),
				)
			} else if argType != nil && argType != expectedType &&
				!types.Identical(argType, expectedType) &&
				!types.AssignableTo(argType, expectedType) &&
				!types.ConvertibleTo(argType, expectedType) &&
				argType.String() != expectedType.String() { // fallback to string comparison to handle loading a package multiple times
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

// anyType returns the empty interface (Go's `any`), used as the fall-back type
// whenever the analyzer cannot deduce a more specific type. Producing `any`
// instead of `nil` keeps downstream consumers (hover, completions, validators)
// from having to special-case missing type information; the analyzer emits an
// ErrorUnknownType diagnostic where the loss of precision is user-relevant.
func anyType() types.Type {
	return types.NewInterfaceType(nil, nil).Complete()
}

// unalias resolves alias types and dereferences pointers recursively, returning
// the underlying named/struct/etc. type. nil in -> nil out.
func unalias(t types.Type) types.Type {
	for {
		t = types.Unalias(t)
		p, ok := t.(*types.Pointer)
		if !ok {
			return t
		}
		t = p.Elem()
	}
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
	if w.Pipe.typ != nil && !IsEmptyInterface(w.Pipe.typ) {
		if _, ok := unalias(w.Pipe.typ).Underlying().(*types.Struct); !ok {
			ctx.errorf(
				w.Pipe,
				ErrorTypeInvalidWith,
				"cannot use type %v in with statement; expected struct type",
				w.Pipe.typ,
			)
		}
	}
	ctx.dotType = w.Pipe.typ
	if ctx.dotType == nil {
		ctx.dotType = anyType()
	}
	w.List = analyseList(n.List, w, ctx)
	ctx.dotType = keepDot
	ctx.vars = ctx.vars[:keepVars]
	w.ElseList = analyseList(n.ElseList, w, ctx)
	return w
}

// IsTemplateSeq reports whether t is rangeable by text/template as an iter.Seq.
// Returns (true, V, nil) for iter.Seq[V] and (true, K, V) for iter.Seq2[K,V].
func isTemplateSeq(t types.Type) (ok bool, key, val types.Type) {
	sig, ok := t.Underlying().(*types.Signature)
	if !ok || sig.Params().Len() != 1 || sig.Results().Len() != 0 {
		return false, nil, nil
	}

	yield, ok := sig.Params().At(0).Type().Underlying().(*types.Signature)
	if !ok || yield.Results().Len() != 1 {
		return false, nil, nil
	}
	if b, ok := yield.Results().At(0).Type().(*types.Basic); !ok || b.Kind() != types.Bool {
		return false, nil, nil
	}

	switch yield.Params().Len() {
	case 1: // iter.Seq[V]
		return true, nil, yield.Params().At(0).Type()
	case 2: // iter.Seq2[K, V]
		return true, yield.Params().At(0).Type(), yield.Params().At(1).Type()
	}
	return false, nil, nil
}

func getRangeableType(typ types.Type, ctx *analysisCtx) (types.Type, types.Type) {
	if typ == nil {
		return nil, nil
	}
	switch t := typ.Underlying().(type) {
	case *types.Pointer:
		return getRangeableType(types.Unalias(t.Elem()), ctx)
	case *types.Array:
		return types.Typ[types.Int], t.Elem()
	case *types.Slice:
		return types.Typ[types.Int], t.Elem()
	case *types.Map:
		return t.Key(), t.Elem()
	case *types.Chan:
		return types.Typ[types.Int], t.Elem()
	case *types.Basic:
		if t.Info()&types.IsInteger != 0 {
			return types.Typ[types.Int], t
		}
		return nil, nil
	case *types.Interface:
		// Special case: empty interface can range over any type
		if t.NumMethods() == 0 {
			ctx.errorf(
				nil,
				ErrorUnknownType,
				"cannot determine range element type of empty interface; assuming any",
			)
			return nil, types.NewInterfaceType(nil, nil).Complete()
		}
		return nil, nil
	default:
		// TODO: handle Seq
		if ok, key, val := isTemplateSeq(t); ok {
			return key, val
		}
		return nil, nil
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
	k, v := getRangeableType(r.Pipe.typ, ctx)
	if r.Pipe.typ == nil {
		ctx.errorf(r.Pipe, ErrorTypeUnknownRangeType, "cannot range over untyped value")
		ctx.dotType = anyType()
	} else if v == nil {
		ctx.errorf(r.Pipe, ErrorTypeInvalidRange, "cannot range over type %v", r.Pipe.typ)
		ctx.dotType = anyType()
	} else {
		ctx.dotType = v
		// override the range var if it was set
		if len(r.Pipe.Decl) == 1 {
			r.Pipe.Decl[0].typ = v
		} else if len(r.Pipe.Decl) == 2 {
			r.Pipe.Decl[0].typ = k
			r.Pipe.Decl[1].typ = v
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
	d := &DotNode{
		NodeType: NodeDot,
		Pos:      Pos(n.Position()),
		parent:   parent,
		typ:      ctx.dotType,
	}
	if d.typ == nil {
		d.typ = anyType()
		ctx.errorf(
			d,
			ErrorUnknownType,
			"cannot determine type of dot; no gotype hint in scope",
		)
	}
	return d
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

	if typ, _ := walkFieldChain(ctx, cn, baseType, n.Field); typ != nil {
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
	// special case: if base is an empty interface, allow any field/method access and return the empty interface type
	if base != nil {
		if iface, ok := base.Underlying().(*types.Interface); ok && iface.NumMethods() == 0 {
			ctx.errorf(
				errNode,
				ErrorUnknownType,
				"cannot determine range element type of empty interface; assuming any",
			)
			return types.NewInterfaceType(nil, nil).Complete(), true
		}
	}

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
			return types.NewInterfaceType(nil, nil).Complete(), false
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
				return types.NewInterfaceType(nil, nil).Complete(), false
			}
			if sig.Results().Len() > 2 {
				ctx.errorf(
					errNode,
					ErrorTypeInvalidField,
					"method %q on type %s returns more than 2 results",
					name,
					currentType.String(),
				)
			}
			// At(1) can be an error
			if sig.Params().Len() == 0 {
				currentType = sig.Results().At(0).Type()
			} else {
				currentType = sig.Results()
			}
		default:
			ctx.errorf(
				errNode,
				ErrorTypeInvalidField,
				"unexpected object type for %q on %s",
				name,
				currentType.String(),
			)
			return types.NewInterfaceType(nil, nil).Complete(), false
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
) (types.Type, []bool) {
	// special case: if base is an empty interface, allow any field/method access and return the empty interface type
	if base != nil {
		if iface, ok := base.Underlying().(*types.Interface); ok && iface.NumMethods() == 0 {
			return types.NewInterfaceType(nil, nil).Complete(), make([]bool, len(path))
		}
	}

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
			return types.NewInterfaceType(nil, nil).Complete(), isMethod
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
				return types.NewInterfaceType(nil, nil).Complete(), isMethod
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
			if sig.Params().Len() == 0 {
				currentType = sig.Results().At(0).Type()
			} else {
				currentType = sig
			}
			isMethod[i] = true
		default:
			ctx.errorf(
				errNode,
				ErrorTypeInvalidField,
				"unexpected object type for %q on %s",
				name,
				currentType.String(),
			)
			return types.NewInterfaceType(nil, nil).Complete(), isMethod
		}
	}
	return currentType, isMethod
}

func analyseIdentifier(n *parse.IdentifierNode, parent Node, ctx *analysisCtx) Node {
	ident := &IdentifierNode{
		NodeType: NodeIdentifier,
		Pos:      Pos(n.Position()),
		Ident:    n.Ident,
		parent:   parent,
		typ:      anyType(),
	}

	name := n.Ident
	if fn, ok := ctx.funcs[name]; ok {
		if fn != nil {
			ident.typ = fn.Type()
		}
		return ident
	}

	ctx.errorf(ident, ErrorTypeInvalidFunction, "undefined function: %s", name)
	return ident
}

func analyseVariable(n *parse.VariableNode, parent Node, ctx *analysisCtx) *VariableNode {
	v := &VariableNode{
		NodeType: NodeVariable,
		Pos:      Pos(n.Position()),
		Ident:    n.Ident,
		parent:   parent,
		typ:      anyType(),
	}
	// Look up base variable in context
	var baseType types.Type
	found := false
	for i := len(ctx.vars) - 1; i >= 0; i-- {
		if len(ctx.vars[i].Ident) == 1 && ctx.vars[i].Ident[0] == n.Ident[0] {
			baseType = ctx.vars[i].typ
			found = true
			break
		}
	}
	if !found {
		return v
	}
	v.Base = baseType

	// $var with no field path -- type is the variable's type.
	if len(n.Ident) == 1 {
		if baseType != nil {
			v.typ = baseType
		}
		return v
	}

	// $var.A.B... -- walk the field/method chain from the variable's type.
	if baseType == nil {
		return v
	}
	if typ, isMethod := walkFieldChainWithMethodInfo(ctx, v, baseType, n.Ident[1:]); typ != nil {
		v.typ = typ
		v.isMethod = isMethod
	}
	return v
}

func analyseField(n *parse.FieldNode, parent Node, ctx *analysisCtx) Node {
	fn := &FieldNode{
		NodeType: NodeField,
		Pos:      Pos(n.Position()),
		Ident:    n.Ident,
		parent:   parent,
		dotType:  ctx.dotType,
		typ:      anyType(),
	}

	if len(n.Ident) == 0 {
		return fn
	}

	if ctx.dotType == nil {
		return fn
	}

	if typ, isMethod := walkFieldChainWithMethodInfo(ctx, fn, ctx.dotType, n.Ident); typ != nil {
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
	next := false
	t := (types.Type)(nil)

	for i, cmd := range pipeNode.Cmds {
		typePipe.Cmds[i] = analyseCommand(cmd, typePipe, ctx, next, t)
		t = typePipe.Cmds[i].typ
		next = true
	}
	typePipe.typ = getNodeType(typePipe.Cmds[len(typePipe.Cmds)-1])
	if typePipe.typ == nil {
		typePipe.typ = anyType()
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
			typePipe.Decl[0].typ = types.Typ[types.Int] // unsigned int for index
			ctx.vars = append(ctx.vars, typePipe.Decl[1])

			if typePipe.Decl[0].Ident[0] == typePipe.Decl[1].Ident[0] {
				ctx.errorf(
					typePipe,
					ErrorDoubleDeclaredVariable,
					"assignment to multiple variables with the same name is not supported",
				)
			}
		}

	} else {
		if len(typePipe.Decl) == 1 {
			// find the variable in the context and update its type
			for i := len(ctx.vars) - 1; i >= 0; i-- {
				if ctx.vars[i].Ident[0] == typePipe.Decl[0].Ident[0] {
					if ctx.vars[i].typ != nil && typePipe.typ != nil &&
						!IsEmptyInterface(ctx.vars[i].typ) && !IsEmptyInterface(typePipe.typ) &&
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
		} else {
			ctx.errorf(
				typePipe,
				ErrorTypeInvalidCommand,
				"assignment to multiple variables is not supported",
			)
		}
	}

	return typePipe
}

// analyseCommand converts a parse CommandNode to a typed CommandNode.
func analyseCommand(
	cmdNode *parse.CommandNode,
	parent Node,
	ctx *analysisCtx,
	next bool,
	pipedT types.Type,
) *CommandNode {
	if cmdNode == nil {
		return nil
	}

	typeCmd := &CommandNode{
		NodeType: NodeCommand,
		Pos:      Pos(cmdNode.Position()),
		Args:     make([]Node, len(cmdNode.Args)),
		parent:   parent,
		typ:      anyType(),
	}

	for i, arg := range cmdNode.Args {
		typeCmd.Args[i] = analyseNode(arg, typeCmd, ctx)
	}

	resultType := getNodeType(typeCmd.Args[0])

	if resultType == nil {
		return typeCmd
	}

	args := []types.Type{}

	for _, arg := range typeCmd.Args[1:] {
		args = append(args, arg.ValueType())
	}
	if fst, ok := cmdNode.Args[0].(*parse.IdentifierNode); ok && fst.Ident == "call" {
		if len(args) == 0 {
			ctx.errorf(
				typeCmd,
				ErrorTypeInvalidCommand,
				"call: missing function argument",
			)
			return typeCmd
		}
		resultType = args[0]
		args = args[1:]
	}

	if next {
		args = append(args, pipedT)
	}

	// TODO: special case for `call` builtin

	// TODO: Typecheck between the command and its arguments to see errors

	// call :: (... -> a) -> ... -> a

	switch t := resultType.Underlying().(type) {
	case *types.Signature:
		result, shouldReturn := validateCommandArguments(t, args, typeCmd, ctx)
		if shouldReturn {
			return result
		}
	default:
		typeCmd.typ = resultType
	}

	return typeCmd
}

func validateCommandArguments(
	t *types.Signature,
	args []types.Type,
	typeCmd *CommandNode,
	ctx *analysisCtx,
) (*CommandNode, bool) {
	if !t.Variadic() && t.Params().Len() != len(args) {
		if ok, _, _ := isTemplateSeq(t); ok {
			typeCmd.typ = t
			return typeCmd, true
		}
		ctx.errorf(
			typeCmd,
			ErrorArgumentNumberMismatch,
			"Expected %d arguments but got %d",
			t.Params().Len(),
			len(args),
		)
		if t.Results().Len() > 0 {
			typeCmd.typ = t.Results().At(0).Type()
		} else {
			typeCmd.typ = t
		}
		return typeCmd, true
	}

	if t.Variadic() {
		if len(args) < t.Params().Len()-1 {
			ctx.errorf(
				typeCmd,
				ErrorArgumentNumberMismatch,
				"Expected at least %d arguments but got %d",
				t.Params().Len()-1,
				len(args),
			)
		}
		nonVariadicCount := t.Params().Len() - 1
		if nonVariadicCount > len(args) {
			nonVariadicCount = len(args)
		}
		for i := 0; i < nonVariadicCount; i++ {
			if !typesCompatible(t.Params().At(i).Type(), args[i]) {
				tstring := "nil"
				if args[i] != nil {
					tstring = args[i].String()
				}
				ctx.errorf(
					typeCmd,
					ErrorTypeInvalidCommand,
					"argument %d: expected type %s but got %s",
					i+1,
					t.Params().At(i).Type().String(),
					tstring,
				)
			} else if IsEmptyInterface(args[i]) && !IsEmptyInterface(t.Params().At(i).Type()) {
				ctx.errorf(
					typeCmd,
					ErrorUnknownType,
					"argument %d: any value passed to %s parameter, may fail at runtime",
					i+1,
					t.Params().At(i).Type().String(),
				)
			}
		}
		variadicType := t.Params().At(t.Params().Len() - 1).Type().(*types.Slice).Elem()
		for i := t.Params().Len() - 1; i < len(args); i++ {
			if !typesCompatible(variadicType, args[i]) {
				tstring := "nil"
				if args[i] != nil {
					tstring = args[i].String()
				}
				ctx.errorf(
					typeCmd,
					ErrorTypeInvalidCommand,
					"variadic argument %d: expected type %s but got %s",
					i+1,
					variadicType.String(),
					tstring,
				)
			} else if IsEmptyInterface(args[i]) && !IsEmptyInterface(variadicType) {
				ctx.errorf(
					typeCmd,
					ErrorUnknownType,
					"variadic argument %d: any value passed to %s parameter, may fail at runtime",
					i+1,
					variadicType.String(),
				)
			}
		}
		if t.Results().Len() > 0 {
			typeCmd.typ = t.Results().At(0).Type()
		}
		return typeCmd, true
	}

	for i := 0; i < t.Params().Len(); i++ {
		if !typesCompatible(t.Params().At(i).Type(), args[i]) {
			tstring := "nil"
			if args[i] != nil {
				tstring = args[i].String()
			}
			ctx.errorf(
				typeCmd,
				ErrorTypeInvalidCommand,
				"argument %d: expected type %s but got %s",
				i+1,
				t.Params().At(i).Type().String(),
				tstring,
			)
		} else if IsEmptyInterface(args[i]) && !IsEmptyInterface(t.Params().At(i).Type()) {
			ctx.errorf(
				typeCmd,
				ErrorUnknownType,
				"argument %d: any value passed to %s parameter, may fail at runtime",
				i+1,
				t.Params().At(i).Type().String(),
			)
		}
	}
	if t.Results().Len() > 0 {
		typeCmd.typ = t.Results().At(0).Type()
	}
	return typeCmd, false
}

// IsEmptyInterface reports whether t is the empty interface (i.e. `any` / `interface{}`).
func IsEmptyInterface(t types.Type) bool {
	if t == nil {
		return false
	}
	iface, ok := t.Underlying().(*types.Interface)
	return ok && iface.NumMethods() == 0
}

// typesCompatible reports whether a value of type got is assignable to a parameter
// of type want. When either side is the empty interface (any), we always accept.
func typesCompatible(want, got types.Type) bool {
	if IsEmptyInterface(want) || IsEmptyInterface(got) {
		return true
	}
	if want == nil || got == nil {
		return false
	}
	return types.Identical(want, got) || types.AssignableTo(got, want)
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
