// Package handlers provides a Language Server Protocol implementation for Go text/templates, featuring scope-aware variable completion and built-in function support.
package handlers

import (
	"go/types"
	"strings"
	parse "text-template-parser"
	serverTypes "text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Context struct is used only in completion_ast to construct Context to be scope aware
// not needed anymore
// Deprecated: Use type tree instead
type Context struct {
	// chain from root to the Node
	Path []parse.Node
	// variables in the scope on the node
	Vars map[string]parse.Node
	// used for the previous functions in the pipe to extract the context using Pipe.Cmds
	Pipe *parse.PipeNode
	// DotType is the resolved Go type of the current dot (.) object.
	DotType *serverTypes.Tree
}

type outputKind int

const (
	outputAny     outputKind = iota
	outputInt                // len
	outputBool               // not, and, or, eq, ne, lt, le, gt, ge
	outputString             // html, js, urlquery, print, printf, println
	outputUntyped            // call, index, slice — dynamic, don't restrict
)

// deprecated, built in output can be derived from the type tree
var builtinOutput = map[string]outputKind{
	"len":      outputInt,
	"not":      outputBool,
	"and":      outputBool,
	"or":       outputBool,
	"eq":       outputBool,
	"ne":       outputBool,
	"lt":       outputBool,
	"le":       outputBool,
	"gt":       outputBool,
	"ge":       outputBool,
	"html":     outputString,
	"js":       outputString,
	"urlquery": outputString,
	"print":    outputString,
	"printf":   outputString,
	"println":  outputString,
	"call":     outputUntyped,
	"index":    outputUntyped,
	"slice":    outputUntyped,
}

// CompletionWithFallback is an entry point that has a fallback option
func CompletionWithFallback(_ *glsp.Context, params *protocol.CompletionParams) (any, error) {
	result := completionAst(nil, params)
	if result == nil {
		log.Debug().Msg("ast completion failed or returned nil, falling back to regex completion")
		return completion(nil, params)
	}
	return result, nil
}

// completions that use the typed AST exclusively. All scope/context
// information (variables, dot type, parent) is derived from traversing the typed tree
func completionAst(_ *glsp.Context, params *protocol.CompletionParams) any {
	if !GetConfig().EnableServer {
		log.Debug().Msg("completion requested but server is disabled by config")
		return nil
	}
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok {
		log.Error().Str("uri", params.TextDocument.URI).Msg("document not found in store")
		return nil
	}
	if doc.typedTree == nil || doc.typedTree.Root == nil {
		log.Error().Str("uri", params.TextDocument.URI).Msg("document has no typed tree")
		return nil
	}

	text := doc.text
	offset := positionToOffset(text, params.Position)

	if !isInsideTemplate(text, offset) {
		return nil
	}

	currentWord := getWordAtOffset(text, offset)
	wordUTF16Len := utf16Len(currentWord)
	startChar := int(params.Position.Character) - wordUTF16Len
	if startChar < 0 {
		startChar = 0
	}
	wordRange := protocol.Range{
		Start: protocol.Position{
			Line:      params.Position.Line,
			Character: protocol.UInteger(startChar),
		},
		End: params.Position,
	}

	cur := serverTypes.NodeFind(doc.typedTree.Root, serverTypes.Pos(offset))
	if cur == nil {
		log.Error().Msg("The target node is not found")
		return nil
	}

	isInvoked := params.Context != nil &&
		params.Context.TriggerKind == protocol.CompletionTriggerKindInvoked

	var sChar uint8
	if offset > 0 && offset <= len(text) {
		sChar = text[offset-1]
	}
	// Ctrl+Space doesn't add a trigger character; infer variable/dot context from the word under cursor.
	if isInvoked {
		if strings.HasPrefix(currentWord, "$") {
			sChar = '$'
		} else if strings.HasPrefix(currentWord, ".") {
			sChar = '.'
		}
	}

	items := suggest(cur, sChar, isInvoked, wordRange)

	return protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

var functionsAccepting = map[outputKind][]string{
	outputInt: {
		"eq", "ne", "lt", "le", "gt", "ge",
		"print", "printf", "println",
	},
	outputBool: {
		"and", "or", "not",
		"print", "printf", "println",
	},
	outputString: {
		"html", "js", "urlquery",
		"len",
		"print", "printf", "println",
		"index",
	},
}

func pipeOutputKind(ctx *Context, isInvoked bool) outputKind {
	if ctx.Pipe == nil {
		log.Debug().Msg("no pipe")
		return outputAny
	}
	cmds := ctx.Pipe.Cmds
	// precedingIdx is -1 when the user invokes the suggestion mid-typing and -2 when automatically suggested
	// TODO: preceding is not always len(cmds) - 2 as we are not always editing the last cmd
	precedingIdx := len(cmds) - 2
	if isInvoked {
		precedingIdx = len(cmds) - 1
	}
	if precedingIdx < 0 || len(cmds) < 1 {
		return outputAny
	}
	preceding := cmds[precedingIdx]
	if len(preceding.Args) == 0 {
		return outputAny
	}
	id, ok := preceding.Args[0].(*parse.IdentifierNode)
	if !ok {
		return outputAny
	}
	if kind, found := builtinOutput[id.Ident]; found {
		return kind
	}
	return outputAny
}

// pipeOutputTypeT returns the output type of the command preceding the cursor's
// command in the enclosing pipe. The type is taken straight from the typed tree
func pipeOutputTypeT(cur serverTypes.Node, isInvoked bool) types.Type {
	pipe := serverTypes.EnclosingPipe(cur)
	if pipe == nil {
		return nil
	}
	cmds := pipe.Cmds
	// TODO: not always -2
	precedingIdx := len(cmds) - 2
	if isInvoked {
		precedingIdx = len(cmds) - 1
	}
	if precedingIdx < 0 || precedingIdx >= len(cmds) {
		return nil
	}
	return cmds[precedingIdx].ValueType()
}

// pipeOutputKindT determines the output kind of the preceding pipe command.
// Falls back to a name lookup for builtin identifiers when the typed node
// has no resolved type (no funcs map is wired in during analysis).
func pipeOutputKindT(cur serverTypes.Node, isInvoked bool) outputKind {
	// TODO: change that to actually get the type from the tree
	pipe := serverTypes.EnclosingPipe(cur)
	if pipe == nil {
		return outputAny
	}
	cmds := pipe.Cmds
	precedingIdx := len(cmds) - 2
	if isInvoked {
		precedingIdx = len(cmds) - 1
	}
	if precedingIdx < 0 || precedingIdx >= len(cmds) {
		return outputAny
	}
	preceding := cmds[precedingIdx]
	if t := preceding.ValueType(); t != nil {
		return typeToOutputKind(t)
	}
	if len(preceding.Args) > 0 {
		if id, ok := preceding.Args[0].(*serverTypes.IdentifierNode); ok {
			if k, found := builtinOutput[id.Ident]; found {
				return k
			}
		}
	}
	return outputAny
}

// dotTypeAt returns the type of the dot in scope at cur, derived from the
// enclosing typed ListNode (set during analysis).
func dotTypeAt(cur serverTypes.Node) types.Type {
	if l := serverTypes.EnclosingList(cur); l != nil {
		return l.ValueType()
	}
	return nil
}

// toNamed unwraps optional pointer indirection and returns the underlying
// *types.Named, or nil if the type is not a named type.
func toNamed(t types.Type) *types.Named {
	if t == nil {
		return nil
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	n, _ := t.(*types.Named)
	return n
}

// visibleVarsAt returns variables in scope at cur. It starts from the snapshot
// stored on the enclosing ListNode and adds top-level decls from
// preceding actions in the same list whose start precedes cur.
func visibleVarsAt(cur serverTypes.Node) []*serverTypes.VariableNode {
	list := serverTypes.EnclosingList(cur)
	if list == nil {
		return nil
	}
	vars := append([]*serverTypes.VariableNode{}, list.Vars()...)
	curPos := cur.Position()
	for _, child := range list.Nodes {
		if child == nil {
			continue
		}
		if child.Position() >= curPos {
			break
		}
		a, ok := child.(*serverTypes.ActionNode)
		if !ok {
			continue
		}
		if a.Pipe != nil && !a.Pipe.IsAssign {
			vars = append(vars, a.Pipe.Decl...)
		}
	}
	return vars
}

// suggest builds the completion list for cur, deriving all scope information
// from the typed tree (parent chain, enclosing list/pipe/command, value types).
func suggest(
	cur serverTypes.Node,
	sChar uint8,
	isInvoked bool,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	if cur == nil {
		return nil
	}

	if sChar == '$' {
		return varsItemsT(visibleVarsAt(cur), true, wordRange)
	}

	if sChar == '.' {
		if cmd := serverTypes.EnclosingCommand(cur); cmd != nil && len(cmd.Args) >= 2 {
			// TODO: not always cmd.args - 2
			arg := cmd.Args[len(cmd.Args)-2]
			switch arg.(type) {
			case *serverTypes.FieldNode,
				*serverTypes.ChainNode,
				*serverTypes.VariableNode,
				*serverTypes.PipeNode:
				if t := arg.ValueType(); t != nil {
					return fieldChainItemsT(t, wordRange)
				}
				return []protocol.CompletionItem{}
			}
		}
		kind := pipeOutputKindT(cur, false)
		pipeIn := pipeOutputTypeT(cur, false)
		return dotItemsT(cur, true, pipeIn, kind, wordRange)
	}

	all := func() []protocol.CompletionItem {
		kind := pipeOutputKindT(cur, isInvoked)
		pipeIn := pipeOutputTypeT(cur, isInvoked)
		inputType := pipeIn
		if inputType == nil && kind == outputAny {
			inputType = dotTypeAt(cur)
		}
		return pipeFilteredItemsT(cur, kind, inputType, pipeIn, wordRange)
	}

	dotAndVars := func() []protocol.CompletionItem {
		items := dotItemsT(cur, false, nil, outputAny, wordRange)
		items = append(items, varsItemsT(visibleVarsAt(cur), false, wordRange)...)
		return items
	}

	parent := cur.Parent()
	switch parent.(type) {
	case *serverTypes.CommandNode:
		return all()
	case *serverTypes.ChainNode, *serverTypes.TemplateNode:
		return dotAndVars()
	case *serverTypes.PipeNode,
		*serverTypes.IfNode,
		*serverTypes.RangeNode,
		*serverTypes.WithNode,
		*serverTypes.ListNode,
		*serverTypes.ActionNode:
		return all()
	default:
		return all()
	}
}

// pipeFilteredItemsT assembles the suggestion list
func pipeFilteredItemsT(
	cur serverTypes.Node,
	kind outputKind,
	inputType types.Type,
	pipeInputType types.Type,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := dotItemsT(cur, false, pipeInputType, kind, wordRange)
	items = append(items, varsItemsT(visibleVarsAt(cur), false, wordRange)...)

	effectiveKind := kind
	if effectiveKind == outputAny && inputType != nil {
		effectiveKind = typeToOutputKind(inputType)
	}

	names, ok := functionsAccepting[effectiveKind]
	if !ok || effectiveKind == outputUntyped || effectiveKind == outputAny {
		items = append(items, builtinItems(wordRange)...)
		return items
	}

	fnKind := protocol.CompletionItemKindFunction
	for _, name := range names {
		items = append(items, protocol.CompletionItem{
			Label:    name,
			Kind:     &fnKind,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: name},
		})
	}
	return items
}

// dotItemsT returns dot, fields, and methods of the dot type at cur. Skipped
// entirely if a pipe input is present (dot does not refer to it).
func dotItemsT(
	cur serverTypes.Node,
	delSign bool,
	inputType types.Type,
	pipeKind outputKind,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}
	inPipe := inputType != nil || (pipeKind != outputAny && pipeKind != outputUntyped)
	if inPipe {
		return items
	}
	prefix := ""
	if !delSign {
		k := protocol.CompletionItemKindVariable
		items = append(items, protocol.CompletionItem{
			Label:    ".",
			Kind:     &k,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: "."},
		})
		prefix = "."
	}
	named := toNamed(dotTypeAt(cur))
	if named != nil {
		items = append(
			items,
			fieldCompletionItems(serverTypes.StructFields(named), prefix, wordRange)...)
		items = append(items, methodCompletionItems(
			serverTypes.NamedMethods(named),
			inputType, pipeKind, prefix, wordRange,
		)...)
	}
	return items
}

// fieldChainItemsT returns fields/methods of t, with no prefix (the dot
// trigger has already been consumed).
func fieldChainItemsT(t types.Type, wordRange protocol.Range) []protocol.CompletionItem {
	named := toNamed(t)
	if named == nil {
		return []protocol.CompletionItem{}
	}
	items := fieldCompletionItems(serverTypes.StructFields(named), "", wordRange)
	items = append(items, methodCompletionItems(
		serverTypes.NamedMethods(named), nil, outputAny, "", wordRange,
	)...)
	return items
}

// varsItemsT renders visible variables as completion items.
func varsItemsT(
	vars []*serverTypes.VariableNode,
	delSign bool,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	if len(vars) == 0 {
		return nil
	}
	items := make([]protocol.CompletionItem, 0, len(vars))
	k := protocol.CompletionItemKindVariable
	seen := map[string]struct{}{}
	for _, v := range vars {
		if v == nil || len(v.Ident) == 0 {
			continue
		}
		name := v.Ident[0]
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		if delSign && name == "$" {
			continue
		}
		label := name
		filter := name
		if delSign {
			label = name[1:]
		}
		items = append(items, protocol.CompletionItem{
			Label:      label,
			Kind:       &k,
			FilterText: &filter,
			TextEdit:   &protocol.TextEdit{Range: wordRange, NewText: name},
		})
	}
	return items
}

// typeToOutputKind maps a concrete Go type to the outputKind used to filter builtins.
func typeToOutputKind(t types.Type) outputKind {
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return outputUntyped
	}
	switch {
	case basic.Info()&types.IsString != 0:
		return outputString
	case basic.Info()&types.IsInteger != 0:
		return outputInt
	case basic.Info()&types.IsBoolean != 0:
		return outputBool
	}
	return outputUntyped
}

// basicTypeMatchesKind reports whether t is compatible with the given output kind.
func basicTypeMatchesKind(t types.Type, kind outputKind) bool {
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	switch kind {
	case outputInt:
		return basic.Info()&types.IsInteger != 0
	case outputBool:
		return basic.Info()&types.IsBoolean != 0
	case outputString:
		return basic.Info()&types.IsString != 0
	}
	return false
}

// methodIsUsable checks whether there are issues in the function definition
// only valid go template functions should be accepted
// functions that return 2 or more arguments are not accepted, except those where one of them is an error
func methodIsUsable(m serverTypes.MethodType) bool {
	if m.Func == nil {
		return false
	}
	sig, ok := m.Func.Type().(*types.Signature)
	if !ok {
		return false
	}
	results := sig.Results()
	switch results.Len() {
	case 1:
		return true
	case 2:
		second := results.At(1).Type()
		errType := types.Universe.Lookup("error").Type()
		return types.Implements(second, errType.Underlying().(*types.Interface))
	default:
		return false
	}
}

// fieldCompletionItems returns the list of fields with or without the dot
func fieldCompletionItems(
	fields []serverTypes.TypeField,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindField
	items := make([]protocol.CompletionItem, 0, len(fields))
	for _, f := range fields {
		detail := f.TypeName
		label := prefix + f.Name
		items = append(items, protocol.CompletionItem{
			Label:    label,
			Kind:     &kind,
			Detail:   &detail,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: label},
		})
	}
	return items
}

// methodAcceptsInput checks whether the function can accept the input
func methodAcceptsInput(m serverTypes.MethodType, inputType types.Type, pipeKind outputKind) bool {
	if inputType != nil {
		for _, p := range m.Params {
			if types.Identical(p.Type, inputType) {
				return true
			}
		}
		return false
	}
	if pipeKind != outputAny && pipeKind != outputUntyped {
		lastParam := m.Params[len(m.Params)-1]
		return basicTypeMatchesKind(lastParam.Type, pipeKind)
	}
	return true
}

// methodCompletionItems builds the function completion list with or without the dot
func methodCompletionItems(
	methods []serverTypes.MethodType,
	inputType types.Type,
	pipeKind outputKind,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindMethod
	items := make([]protocol.CompletionItem, 0, len(methods))
	for _, m := range methods {
		if !methodIsUsable(m) || !methodAcceptsInput(m, inputType, pipeKind) {
			continue
		}
		detail := m.ReturnName
		label := prefix + m.Name
		items = append(items, protocol.CompletionItem{
			Label:    label,
			Kind:     &kind,
			Detail:   &detail,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: label},
		})
	}
	return items
}

func builtinItems(wordRange protocol.Range) []protocol.CompletionItem {
	statics := []string{
		"and", "call", "html", "index", "slice", "js", "len",
		"not", "or", "print", "printf", "println", "urlquery",
		"eq", "ne", "lt", "le", "gt", "ge", "if", "range",
	}
	kind := protocol.CompletionItemKindFunction
	items := make([]protocol.CompletionItem, 0, len(statics))
	for _, name := range statics {
		items = append(items, protocol.CompletionItem{
			Label:    name,
			Kind:     &kind,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: name},
		})
	}
	return items
}

// extract method for buildPath
func isLeafNode(n parse.Node) bool {
	switch n.(type) {
	case *parse.IdentifierNode,
		*parse.VariableNode,
		*parse.FieldNode,
		*parse.DotNode,
		*parse.NilNode,
		*parse.BoolNode,
		*parse.NumberNode,
		*parse.StringNode,
		*parse.TextNode,
		*parse.CommentNode,
		*parse.BreakNode,
		*parse.UndefinedNode,
		*parse.ContinueNode:
		return true
	}
	return false
}

// main build path function that calls supporting functions and builds path and context
func buildPath(n parse.Node, target parse.Node, ctx *Context) bool {
	if n == target {
		return true
	}
	ctx.Path = append(ctx.Path, n)

	found := buildPathChildren(n, target, ctx)

	if !found {
		ctx.Path = ctx.Path[:len(ctx.Path)-1]
	}
	return found
}

// builds path on the branch by taking snapshots and falling back in case not found
func buildPathBranch(
	pipe *parse.PipeNode,
	list *parse.ListNode,
	elseList *parse.ListNode,
	target parse.Node,
	ctx *Context,
) bool {
	if ctx == nil {
		return false
	}
	if ctx.Vars == nil {
		ctx.Vars = make(map[string]parse.Node)
	}
	snapshot := snapshotVars(ctx.Vars)

	found := buildPath(pipe, target, ctx) ||
		buildPath(list, target, ctx) ||
		(elseList != nil && buildPath(elseList, target, ctx))

	if !found {
		ctx.Vars = snapshot
	}
	return found
}

// resolvePipeDotType derives the dot type for the body of a range or with block.
func resolvePipeDotType(
	pipe *parse.PipeNode,
	unwrapSlice bool,
	ctx *Context,
) *serverTypes.Tree {
	if ctx.DotType == nil || pipe == nil || len(pipe.Cmds) != 1 {
		return ctx.DotType
	}
	cmd := pipe.Cmds[0]
	if len(cmd.Args) != 1 {
		return ctx.DotType
	}
	field, ok := cmd.Args[0].(*parse.FieldNode)
	if !ok || len(field.Ident) != 1 {
		return ctx.DotType
	}
	fieldName := field.Ident[0]
	var fieldType types.Type
	for _, f := range serverTypes.StructFields(ctx.DotType.DotType) {
		if f.Name == fieldName {
			fieldType = f.Type
			break
		}
	}
	if fieldType == nil {
		return ctx.DotType
	}
	t := fieldType
	if unwrapSlice {
		sl, ok := t.Underlying().(*types.Slice)
		if !ok {
			// probably trying to iterate over a struct, not a slice
			return nil
		}
		t = sl.Elem()
	} else {
		if _, ok := t.Underlying().(*types.Slice); ok {
			return nil
		}
	}
	named, ok := t.(*types.Named)
	if !ok {
		if ptr, ok2 := t.(*types.Pointer); ok2 {
			named, ok = ptr.Elem().(*types.Named)
			if !ok {
				return ctx.DotType
			}
		} else {
			return ctx.DotType
		}
	}
	return &serverTypes.Tree{
		DotType: named,
		Pkg:     ctx.DotType.Pkg,
	}
}

// main traversal logic
func buildPathChildren(n parse.Node, target parse.Node, ctx *Context) bool {
	if isLeafNode(n) {
		return false
	}
	if ctx == nil || n == nil {
		return false
	}

	switch n := n.(type) {
	case *parse.ListNode:
		for _, child := range n.Nodes {
			if buildPath(child, target, ctx) {
				return true
			}
		}

	case *parse.ActionNode:
		return buildPath(n.Pipe, target, ctx)

	case *parse.PipeNode:
		if ctx.Vars == nil {
			ctx.Vars = make(map[string]parse.Node)
		}
		for _, v := range n.Decl {
			if v != nil && v == target {
				return true
			}
			if v != nil && len(v.Ident) > 0 {
				ctx.Vars[v.Ident[0]] = n
			}
		}
		prevPipe := ctx.Pipe
		ctx.Pipe = n
		for _, cmd := range n.Cmds {
			if buildPath(cmd, target, ctx) {
				return true
			}
		}
		ctx.Pipe = prevPipe
		return false

	case *parse.IfNode:
		return buildPathBranch(n.Pipe, n.List, n.ElseList, target, ctx)

	case *parse.RangeNode:
		prevDot := ctx.DotType
		ctx.DotType = resolvePipeDotType(n.Pipe, true, ctx)
		found := buildPathBranch(n.Pipe, n.List, n.ElseList, target, ctx)
		if !found {
			ctx.DotType = prevDot
		}
		return found

	case *parse.WithNode:
		prevDot := ctx.DotType
		ctx.DotType = resolvePipeDotType(n.Pipe, false, ctx)
		found := buildPathBranch(n.Pipe, n.List, n.ElseList, target, ctx)
		if !found {
			ctx.DotType = prevDot
		}
		return found

	case *parse.CommandNode:
		for _, arg := range n.Args {
			if buildPath(arg, target, ctx) {
				return true
			}
		}

	case *parse.TemplateNode:
		if n.Pipe != nil {
			return buildPath(n.Pipe, target, ctx)
		}

	case *parse.ChainNode:
		return buildPath(n.Node, target, ctx)

	default:
		log.Error().Msg("The tree contains an incomplete Node")
	}
	return false
}

// Take snapshots of defined variables, needed to change scope when the Node wasn't found in some path
func snapshotVars(vars map[string]parse.Node) map[string]parse.Node {
	snap := make(map[string]parse.Node, len(vars))
	for k, v := range vars {
		snap[k] = v
	}
	return snap
}
