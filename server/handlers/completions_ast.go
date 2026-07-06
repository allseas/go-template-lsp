// Package handlers provides a Language Server Protocol implementation for Go text/templates, featuring scope-aware variable completion and built-in function support.
package handlers

import (
	"go/types"
	"strings"
	serverTypes "text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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
	if !GetConfig().EnableAutocompletion {
		log.Debug().Msg("completion requested but autocompletion is disabled by config")
		return nil
	}
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok {
		log.Error().Str("uri", params.TextDocument.URI).Msg("document not found in store")
		return nil
	}

	text := doc.text
	offset := positionToOffset(text, params.Position)

	if !isInsideTemplate(text, offset) {
		return nil
	}

	typedTree := doc.typedTreeAtTyped(serverTypes.Pos(offset))
	if typedTree == nil || typedTree.Root == nil {
		log.Error().Str("uri", params.TextDocument.URI).Msg("document has no typed tree")
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
	cur := serverTypes.NodeFind(typedTree.Root, serverTypes.Pos(findOffset))
	if cur == nil {
		log.Error().Msg("The target node is not found")
		return nil
	}

	items := suggest(cur, sChar, isInvoked, serverTypes.Pos(offset), text, wordRange)

	return protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
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

// pipeOutputType returns the value type produced by the command preceding the
// cursor's position in the enclosing pipe, or nil when there is none.
func pipeOutputType(cur serverTypes.Node, isInvoked bool) types.Type {
	cmd := precedingCmd(cur, isInvoked)
	if cmd == nil {
		return nil
	}
	t := cmd.ValueType()
	if t == nil {
		return nil
	}
	// analyseCommand represents a partially-applied function (pipe target) as a
	// curried *types.Signature. Unwrap it to obtain the actual return type.
	if sig, isSig := t.Underlying().(*types.Signature); isSig {
		if sig.Results().Len() == 0 {
			return nil
		}
		t = sig.Results().At(0).Type()
	}
	return t
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
		return n.PrefixType(), true
	case *serverTypes.FieldNode:
		if cur != arg {
			return n.ValueType(), true
		}
		return n.PrefixType(), true
	case *serverTypes.ChainNode:
		if cur != arg {
			return n.ValueType(), true
		}
		return n.PrefixType(), true
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

// suggest builds the completion list for cur, deriving all scope information
// from the typed tree (parent chain, enclosing list/pipe/command, value types).
func suggest(
	cur serverTypes.Node,
	sChar uint8,
	isInvoked bool,
	offset serverTypes.Pos,
	text string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	if cur == nil {
		return nil
	}

	if sChar == '$' {
		return varsItemsT(serverTypes.VisibleVarsAt(cur), true, wordRange)
	}

	// argType is the type required at cur's position by the function that consumes the value here
	argType := targetTypeForArg(cur, offset, text)

	if sChar == '.' {
		if t, inChain := chainContext(cur); inChain {
			if t == nil {
				return []protocol.CompletionItem{}
			}
			items := fieldChainItemsT(t, argType, wordRange)
			items = append(items, chainExpansionItems(cur, t, "", offset, text, wordRange)...)
			return items
		}
		pipeIn := pipeOutputType(cur, false)
		items := dotItemsT(cur, true, pipeIn, argType, wordRange)
		items = append(
			items,
			chainExpansionItems(cur, dotTypeAt(cur), "", offset, text, wordRange)...)
		return items
	}

	switch cur.Parent().(type) {
	case *serverTypes.ChainNode, *serverTypes.TemplateNode:
		items := dotItemsT(cur, false, nil, argType, wordRange)
		return append(items, varsItemsT(serverTypes.VisibleVarsAt(cur), false, wordRange)...)
	}

	pipeIn := pipeOutputType(cur, isInvoked)
	// cursor value slot prevents the bug of suggestion fields overflowing to the next pipe
	if cursorInValueSlot(cur, offset, text) {
		pipeIn = nil
	}
	items := pipeFilteredItemsT(cur, pipeIn, argType, wordRange)
	if pipeIn == nil {
		items = append(
			items,
			chainExpansionItems(cur, dotTypeAt(cur), ".", offset, text, wordRange)...)
	}
	return items
}

// funcReturnsType reports whether fn's first result type is assignable to
// argType. Signatures with dynamic (interface{}) results always match.
func funcReturnsType(fn *types.Func, argType types.Type) bool {
	if serverTypes.IsEmptyInterface(argType) {
		return true
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Results().Len() == 0 {
		return true
	}
	return serverTypes.TypeConvertibleTo(sig.Results().At(0).Type(), argType)
}

// pipeFilteredItemsT assembles the suggestion list: dot's fields/methods,
// visible variables, and global funcs whose signature is compatible with the
// pipe input (as accepted last param) and argType (as returned result).
func pipeFilteredItemsT(
	cur serverTypes.Node,
	inputType types.Type,
	argType types.Type,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := dotItemsT(cur, false, inputType, argType, wordRange)
	items = append(items, varsItemsT(serverTypes.VisibleVarsAt(cur), false, wordRange)...)
	for name, fn := range serverTypes.GlobalFuncs() {
		accepts, concrete := funcAcceptsPipeInput(fn, inputType)
		if !accepts || !funcReturnsType(fn, argType) {
			continue
		}
		item := newItem(name, protocol.CompletionItemKindFunction, wordRange)
		sortText := pipeSortPrefix(concrete) + name
		item.SortText = &sortText
		items = append(items, item)
	}
	if inputType == nil && serverTypes.IsEmptyInterface(argType) {
		items = append(items, templateActionItems(wordRange)...)
	}
	return items
}

// Appends to the name so that functions that accept a concrete type are presented before those that only accept interface{}.
func pipeSortPrefix(concrete bool) string {
	if concrete {
		return "0_"
	}
	return "1_"
}

func funcAcceptsPipeInput(fn *types.Func, pipeInputType types.Type) (accepts bool, concrete bool) {
	if pipeInputType == nil {
		// No pipe input to match: concreteness is meaningless, so don't
		// promote funcs above fields/methods in the sort order.
		return true, false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false, false
	}
	params := sig.Params()
	if params.Len() == 0 {
		return false, false
	}

	last := params.At(params.Len() - 1).Type()
	if sig.Variadic() {
		if slice, ok := last.(*types.Slice); ok {
			last = slice.Elem()
		}
	}
	if !types.AssignableTo(pipeInputType, last) {
		return false, false
	}
	return true, !serverTypes.IsEmptyInterface(last)
}

// dotItemsT returns dot, fields, and methods of the dot type at cur.
func dotItemsT(
	cur serverTypes.Node,
	delSign bool,
	inputType types.Type,
	argType types.Type,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}
	if inputType != nil && !serverTypes.IsEmptyInterface(inputType) {
		return items
	}
	prefix := ""
	if !delSign {
		if serverTypes.TypeConvertibleTo(dotTypeAt(cur), argType) {
			items = append(items, newItem(".", protocol.CompletionItemKindVariable, wordRange))
		}
		prefix = "."
	}
	items = append(items, getItems(dotTypeAt(cur), inputType, argType, prefix, wordRange)...)
	return items
}

// fieldChainItemsT returns fields/methods of t, with no prefix (the dot
// trigger has already been consumed). argType, when non-nil, restricts the
// result to fields/methods producing a compatible type.
func fieldChainItemsT(
	t types.Type,
	argType types.Type,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	return getItems(t, nil, argType, "", wordRange)
}

// getItems renders the field + method completions for t. Named types contribute
// both; dicts contribute only their keys as fields. Any other type yields no
// items. Each item is prefixed with prefix (used to keep or strip the leading
// dot).
func getItems(
	t types.Type,
	inputType types.Type,
	argType types.Type,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	fields, methods := collectFieldsAndMethods(t)
	fields = filterFieldsByReturn(fields, argType)
	methods = filterMethods(methods, inputType, argType)
	items := fieldCompletionItems(fields, prefix, wordRange)
	return append(items, methodCompletionItems(methods, prefix, wordRange)...)
}

// collectFieldsAndMethods returns the raw fields and methods reachable from t.
// Dicts contribute keys as fields and no methods; named types contribute both.
func collectFieldsAndMethods(t types.Type) ([]serverTypes.TypeField, []serverTypes.MethodType) {
	if dict, ok := t.(*serverTypes.DictType); ok && dict != nil {
		return serverTypes.DictTypeFields(dict), nil
	}
	named := toNamed(t)
	if named == nil {
		return nil, nil
	}
	return serverTypes.StructFields(named), serverTypes.NamedMethods(named)
}

// filterFieldsByReturn keeps only fields whose type is assignable to argType.
func filterFieldsByReturn(fields []serverTypes.TypeField, argType types.Type) []serverTypes.TypeField {
	if serverTypes.IsEmptyInterface(argType) {
		return fields
	}
	out := fields[:0:0]
	for _, f := range fields {
		if serverTypes.TypeConvertibleTo(f.Type, argType) {
			out = append(out, f)
		}
	}
	return out
}

// filterMethods keeps only usable methods that accept inputType and whose
// return type is assignable to argType.
func filterMethods(methods []serverTypes.MethodType, inputType, argType types.Type) []serverTypes.MethodType {
	out := methods[:0:0]
	for _, m := range methods {
		if !methodIsUsable(m) || !methodAcceptsInput(m, inputType) {
			continue
		}
		if !serverTypes.TypeConvertibleTo(m.ReturnType, argType) {
			continue
		}
		out = append(out, m)
	}
	return out
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

// fieldCompletionItems renders fields as completion items with an optional prefix.
func fieldCompletionItems(
	fields []serverTypes.TypeField,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(fields))
	funcs := serverTypes.GlobalFuncs()
	for _, f := range fields {
		item := newDetailItem(
			prefix+f.Name, f.TypeName, protocol.CompletionItemKindField, wordRange,
		)
		if _, ok := funcs[f.Name]; !ok {
			sortText := "0_" + f.Name
			item.SortText = &sortText
		}
		items = append(items, item)
	}
	return items
}

// methodAcceptsInput checks whether the function can accept the input
func methodAcceptsInput(m serverTypes.MethodType, inputType types.Type) bool {
	if inputType == nil || serverTypes.IsEmptyInterface(inputType) {
		return true
	}
	for _, p := range m.Params {
		if types.Identical(p.Type, inputType) || serverTypes.IsEmptyInterface(p.Type) {
			return true
		}
	}
	return false
}

// methodCompletionItems renders methods as completion items with an optional prefix.
func methodCompletionItems(
	methods []serverTypes.MethodType,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(methods))
	funcs := serverTypes.GlobalFuncs()
	for _, m := range methods {
		item := newDetailItem(
			prefix+m.Name, m.ReturnName, protocol.CompletionItemKindMethod, wordRange,
		)
		if _, ok := funcs[m.Name]; !ok {
			sortText := "1_" + m.Name
			item.SortText = &sortText
		}
		items = append(items, item)
	}
	return items
}

// templateActionNames lists the Go-template control keywords that should be
// suggested alongside functions when no pipe constraint is active.
// These are NOT in any FuncMap so they are kept separately from GlobalFuncs.
var templateActionNames = []string{"if", "range"}

func templateActionItems(wordRange protocol.Range) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(templateActionNames))
	for _, name := range templateActionNames {
		item := newItem(name, protocol.CompletionItemKindKeyword, wordRange)
		sortText := "1_" + name
		item.SortText = &sortText
		items = append(items, item)
	}
	return items
}
