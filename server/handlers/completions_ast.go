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

// funcAcceptsKind reports whether funcName can receive a piped value of kind.
//
// For functions with at least one concrete (non-interface{}) parameter the
// check is performed by type: the function accepts the kind iff at least one
// concrete param matches. For functions whose every parameter is interface{}
// (all current builtins) the check falls back to the curated functionsAccepting
// table, preserving the existing semantic filtering.
func funcAcceptsKind(funcName string, kind outputKind) bool {
	if kind == outputAny || kind == outputUntyped {
		return true
	}
	fn := serverTypes.GlobalFuncs()[funcName]
	if fn == nil {
		return true // unknown signature — don't flag
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return true
	}
	params := sig.Params()
	hasConcreteParam := false
	for i := range params.Len() {
		t := params.At(i).Type()
		// Unwrap variadic slice wrapper.
		if sl, isSl := t.Underlying().(*types.Slice); isSl {
			t = sl.Elem()
		}
		if _, isIface := t.Underlying().(*types.Interface); isIface {
			continue
		}
		hasConcreteParam = true
		if basicTypeMatchesKind(t, kind) {
			return true
		}
	}
	if hasConcreteParam {
		return false // concrete params present but none matched
	}
	// All params are interface{} — use the curated semantic list.
	allowed := functionsAccepting[kind]
	for _, name := range allowed {
		if name == funcName {
			return true
		}
	}
	return false
}

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

	isInvoked := params.Context != nil &&
		params.Context.TriggerKind == protocol.CompletionTriggerKindInvoked

	var sChar uint8
	prefixOffset := offset - len(currentWord)
	if prefixOffset > 0 && prefixOffset <= len(text) {
		sChar = text[prefixOffset-1]
	}
	if strings.HasPrefix(currentWord, "$") {
		sChar = '$'
	}

	findOffset := offset
	if sChar == '.' && findOffset > 0 {
		findOffset--
	}
	cur := serverTypes.NodeFind(doc.typedTree.Root, serverTypes.Pos(findOffset))
	if cur == nil {
		log.Error().Msg("The target node is not found")
		return nil
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

// precedingCmd returns the command whose output flows into the current node
func precedingCmd(cur serverTypes.Node, isInvoked bool) *serverTypes.CommandNode {
	pipe := serverTypes.EnclosingPipe(cur)
	if pipe == nil || len(pipe.Cmds) == 0 {
		return nil
	}
	idx := -1
	for n := cur; n != nil && idx < 0; n = n.Parent() {
		cmd, ok := n.(*serverTypes.CommandNode)
		if !ok {
			continue
		}
		for i, c := range pipe.Cmds {
			if c == cmd {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		idx = len(pipe.Cmds) - 1
	}
	if !isInvoked {
		idx--
	}
	if idx < 0 || idx >= len(pipe.Cmds) {
		return nil
	}
	return pipe.Cmds[idx]
}

// pipeOutputInfo returns the value type and output kind produced by the command
// preceding the cursor's position in the enclosing pipe.
func pipeOutputInfo(cur serverTypes.Node, isInvoked bool) (types.Type, outputKind) {
	cmd := precedingCmd(cur, isInvoked)
	if cmd == nil {
		return nil, outputAny
	}
	if t := cmd.ValueType(); t != nil {
		return t, typeToOutputKind(t)
	}
	if len(cmd.Args) > 0 {
		if id, ok := cmd.Args[0].(*serverTypes.IdentifierNode); ok {
			// TODO: get rid of the deprecated builtin
			if k, found := builtinOutput[id.Ident]; found {
				return nil, k
			}
		}
	}
	return nil, outputAny
}

// chainContext returns the type whose fields/methods should be suggested for a `.` trigger at cur
// ok is true when the node is chainable
func chainContext(cur serverTypes.Node) (types.Type, bool) {
	var arg serverTypes.Node
	var cmd *serverTypes.CommandNode
	for n := cur; n != nil; n = n.Parent() {
		if c, ok := n.Parent().(*serverTypes.CommandNode); ok {
			arg = n
			cmd = c
			break
		}
	}
	if arg == nil {
		return nil, false
	}
	switch n := arg.(type) {
	case *serverTypes.PipeNode:
		return arg.ValueType(), true
	case *serverTypes.VariableNode:
		if cur != arg || len(n.Ident) <= 1 {
			return n.ValueType(), true
		}
		return chainPrefix(varBaseType(cur, n.Ident[0]), n.Ident[1:]), true
	case *serverTypes.FieldNode:
		if cur != arg {
			return n.ValueType(), true
		}
		return chainPrefix(dotTypeAt(cur), n.Ident), true
	case *serverTypes.ChainNode:
		if cur != arg {
			return n.ValueType(), true
		}
		var base types.Type
		if n.Node != nil {
			base = n.Node.ValueType()
		}
		return chainPrefix(base, n.Field), true
	}
	for i, a := range cmd.Args {
		if a != arg || i == 0 {
			continue
		}
		prev := cmd.Args[i-1]
		switch prev.(type) {
		case *serverTypes.VariableNode, *serverTypes.PipeNode,
			*serverTypes.FieldNode, *serverTypes.ChainNode:
			return prev.ValueType(), true
		}
		break
	}
	return nil, false
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

// walkChainType walks idents from base through fields/methods and returns the
// resulting type, or nil if any step fails to resolve.
func walkChainType(base types.Type, idents []string) types.Type {
	cur := base
	for _, name := range idents {
		if cur == nil {
			return nil
		}
		obj, _, _ := types.LookupFieldOrMethod(cur, true, nil, name)
		switch o := obj.(type) {
		case *types.Var:
			cur = o.Type()
		case *types.Func:
			sig, ok := o.Type().Underlying().(*types.Signature)
			if !ok || sig.Results().Len() == 0 {
				return nil
			}
			cur = sig.Results().At(0).Type()
		default:
			return nil
		}
	}
	return cur
}

// chainPrefix returns the type reached by walking all but the last ident of
// path from base. Used when the cursor is mid-typing the trailing component.
func chainPrefix(base types.Type, path []string) types.Type {
	if len(path) <= 1 {
		return base
	}
	return walkChainType(base, path[:len(path)-1])
}

// varBaseType returns the resolved type of the variable named name (e.g. "$" or
// "$top") visible at cur, by walking the same scope used for variable
// completions.
func varBaseType(cur serverTypes.Node, name string) types.Type {
	for _, v := range visibleVarsAt(cur) {
		if v != nil && len(v.Ident) > 0 && v.Ident[0] == name {
			return v.ValueType()
		}
	}
	return nil
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
		if t, inChain := chainContext(cur); inChain {
			if t == nil {
				return []protocol.CompletionItem{}
			}
			return fieldChainItemsT(t, wordRange)
		}
		pipeIn, kind := pipeOutputInfo(cur, false)
		return dotItemsT(cur, true, pipeIn, kind, wordRange)
	}

	switch cur.Parent().(type) {
	case *serverTypes.ChainNode, *serverTypes.TemplateNode:
		items := dotItemsT(cur, false, nil, outputAny, wordRange)
		return append(items, varsItemsT(visibleVarsAt(cur), false, wordRange)...)
	}

	pipeIn, kind := pipeOutputInfo(cur, isInvoked)
	inputType := pipeIn
	if inputType == nil && kind == outputAny {
		inputType = dotTypeAt(cur)
	}
	return pipeFilteredItemsT(cur, kind, inputType, pipeIn, wordRange)
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
	if !ok {
		return append(items, builtinItems(wordRange)...)
	}
	for _, name := range names {
		items = append(items, newItem(name, protocol.CompletionItemKindFunction, wordRange))
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
	if inputType != nil || (pipeKind != outputAny && pipeKind != outputUntyped) {
		return items
	}
	prefix := ""
	if !delSign {
		items = append(items, newItem(".", protocol.CompletionItemKindVariable, wordRange))
		prefix = "."
	}
	if named := toNamed(dotTypeAt(cur)); named != nil {
		items = append(items, namedItems(named, inputType, pipeKind, prefix, wordRange)...)
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
	return namedItems(named, nil, outputAny, "", wordRange)
}

// namedItems returns the field + filtered method completions for a named type,
// each prefixed with prefix (used to keep or strip the leading dot).
func namedItems(
	named *types.Named,
	inputType types.Type,
	pipeKind outputKind,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := fieldCompletionItems(serverTypes.StructFields(named), prefix, wordRange)
	return append(items, methodCompletionItems(
		serverTypes.NamedMethods(named), inputType, pipeKind, prefix, wordRange,
	)...)
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
		if delSign {
			label = name[1:]
		}
		k := protocol.CompletionItemKindVariable
		filter := name
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

// newItem builds a CompletionItem whose TextEdit replaces wordRange with label.
func newItem(
	label string,
	kind protocol.CompletionItemKind,
	wordRange protocol.Range,
) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label:    label,
		Kind:     &kind,
		TextEdit: &protocol.TextEdit{Range: wordRange, NewText: label},
	}
}

// newDetailItem is newItem plus a Detail string.
func newDetailItem(
	label, detail string,
	kind protocol.CompletionItemKind,
	wordRange protocol.Range,
) protocol.CompletionItem {
	item := newItem(label, kind, wordRange)
	item.Detail = &detail
	return item
}

// fieldCompletionItems returns the list of fields with or without the dot
func fieldCompletionItems(
	fields []serverTypes.TypeField,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(fields))
	for _, f := range fields {
		items = append(items, newDetailItem(
			prefix+f.Name, f.TypeName, protocol.CompletionItemKindField, wordRange,
		))
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
	items := make([]protocol.CompletionItem, 0, len(methods))
	for _, m := range methods {
		if !methodIsUsable(m) || !methodAcceptsInput(m, inputType, pipeKind) {
			continue
		}
		items = append(items, newDetailItem(
			prefix+m.Name, m.ReturnName, protocol.CompletionItemKindMethod, wordRange,
		))
	}
	return items
}

// templateActionNames lists the Go-template control keywords that should be
// suggested alongside functions when no pipe constraint is active.
// These are NOT in any FuncMap so they are kept separately from GlobalFuncs.
var templateActionNames = []string{"if", "range"}

func builtinItems(wordRange protocol.Range) []protocol.CompletionItem {
	funcs := serverTypes.GlobalFuncs()
	items := make([]protocol.CompletionItem, 0, len(funcs)+len(templateActionNames))
	for name := range funcs {
		items = append(items, newItem(name, protocol.CompletionItemKindFunction, wordRange))
	}
	for _, name := range templateActionNames {
		items = append(items, newItem(name, protocol.CompletionItemKindKeyword, wordRange))
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
