// Package handlers provides a Language Server Protocol implementation for Go text/templates, featuring scope-aware variable completion and built-in function support.
package handlers

import (
	"fmt"
	"go/types"
	"strings"
	parse "text-template-parser"
	servertypes "text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Context struct is used only in completion_ast to construct Context to be scope aware
type Context struct {
	// chain from root to the Node
	Path []parse.Node
	// variables in the scope on the node
	Vars map[string]parse.Node
	// used for the previous functions in the pipe to extract the context using Pipe.Cmds
	Pipe *parse.PipeNode
	// DotType is the resolved Go type of the current dot (.) object.
	DotType *servertypes.Tree
}

type outputKind int

const (
	outputAny     outputKind = iota
	outputInt                // len
	outputBool               // not, and, or, eq, ne, lt, le, gt, ge
	outputString             // html, js, urlquery, print, printf, println
	outputUntyped            // call, index, slice — dynamic, don't restrict
)

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

// completion entry point that has a fallback option
func completionWithFallback(_ *glsp.Context, params *protocol.CompletionParams) (any, error) {
	result := completionAst(nil, params)
	if result == nil {
		log.Debug().Msg("ast completion failed or returned nil, falling back to regex completion")
		return completion(nil, params)
	}
	return result, nil
}

// completions that use AST
func completionAst(_ *glsp.Context, params *protocol.CompletionParams) any {
	doc, ok := store.Get(params.TextDocument.URI)
	if !GetConfig().EnableServer {
		log.Debug().Msg("completion requested but server is disabled by config")
		return nil
	}

	if !ok {
		log.Error().Str("uri", params.TextDocument.URI).Msg("document not found in store")
		return nil
	}

	if doc.tree == nil {
		log.Error().Str("uri", params.TextDocument.URI).Msg("document has no parsed tree")
		return nil
	}

	text := doc.text
	tree := doc.tree

	ctx := &Context{
		Vars:    map[string]parse.Node{"$": nil},
		DotType: doc.loadedType,
	}

	offset := positionToOffset(text, params.Position)

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

	curNode := nodeFind(tree.Root, parse.Pos(offset))
	result := buildPath(tree.Root, curNode, ctx)

	logPath(ctx)

	isInvoked := params.Context != nil &&
		params.Context.TriggerKind == protocol.CompletionTriggerKindInvoked

	var parent parse.Node
	if len(ctx.Path) <= 1 {
		log.Debug().Msg("context not passed")
	} else {
		parent = ctx.Path[len(ctx.Path)-2]
	}

	if !isInsideTemplate(text, offset) {
		return nil
	}

	if !result {
		log.Error().Msg("The target node is not found")
		return nil
	}

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

	items := suggest(parent, ctx, sChar, isInvoked, wordRange)

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

// pipeOutputType returns the type of a previous command in a pipe.
// If the current command is broken, then nil is returned, as the pipe becomes nil
func pipeOutputType(ctx *Context, isInvoked bool) types.Type {
	if ctx.Pipe == nil || ctx.DotType == nil {
		return nil
	}
	cmds := ctx.Pipe.Cmds
	precedingIdx := len(cmds) - 2
	if isInvoked {
		precedingIdx = len(cmds) - 1
	}
	if precedingIdx < 0 {
		return nil
	}
	preceding := cmds[precedingIdx]
	if len(preceding.Args) == 0 {
		return nil
	}
	switch arg := preceding.Args[0].(type) {
	case *parse.DotNode:
		return ctx.DotType.DotType
	case *parse.FieldNode:
		if len(arg.Ident) != 1 {
			return nil
		}
		name := arg.Ident[0]
		for _, f := range structFields(ctx.DotType.DotType) {
			if f.Name == name {
				return f.Type
			}
		}
		for _, m := range namedMethods(ctx.DotType.DotType) {
			if m.Name == name {
				return m.ReturnType
			}
		}
	case *parse.IdentifierNode:
		for _, m := range namedMethods(ctx.DotType.DotType) {
			if m.Name == arg.Ident {
				return m.ReturnType
			}
		}
		return nil
	}
	return nil
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
func methodIsUsable(m MethodType) bool {
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

// pipeFilteredItems determines based on the input types what should be the suggestion list
func pipeFilteredItems(
	kind outputKind,
	inputType types.Type,
	pipeInputType types.Type,
	ctx *Context,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0)
	items = append(items, dotItem(*ctx, false, pipeInputType, kind, wordRange)...)
	items = append(items, varsToItems(ctx, false, wordRange)...)

	if ctx.DotType == nil {
		items = append(items, builtinItems(wordRange)...)
		return items
	}

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

func suggest(
	parent parse.Node,
	ctx *Context,
	sChar uint8,
	isInvoked bool,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	if sChar == '$' {
		return varsToItems(ctx, true, wordRange)
	}

	if sChar == '.' {
		kind := pipeOutputKind(ctx, isInvoked)
		pipeInputType := pipeOutputType(ctx, isInvoked)

		// if both pipeInputType is nil and kind = outputAny, it means that there was an issue while parsing and the whole pipe became an UndefinedNode
		// one solution would be to remove the incorrect part of the pipe and give suggestions, but this is clumsy, so the parser should be adjusted
		return dotItem(*ctx, true, pipeInputType, kind, wordRange)
	}

	all := func() []protocol.CompletionItem {
		// kind is primarily used to determine the output type of a function that is the previous node in a pipe
		kind := pipeOutputKind(ctx, isInvoked)
		// pipeInputType is used to determine the type of the dot object that the function is going to be applied on
		pipeInputType := pipeOutputType(ctx, isInvoked)
		inputType := pipeInputType
		if inputType == nil && kind == outputAny && ctx.DotType != nil {
			inputType = ctx.DotType.DotType
		}
		return pipeFilteredItems(kind, inputType, pipeInputType, ctx, wordRange)
	}

	dotAndVars := func() []protocol.CompletionItem {
		return append(
			dotItem(*ctx, false, nil, outputAny, wordRange),
			varsToItems(ctx, false, wordRange)...)
	}

	switch parent.(type) {
	case *parse.CommandNode:
		return all()

	case *parse.ChainNode:
		return dotAndVars()

	case *parse.TemplateNode:
		return dotAndVars()

	case *parse.PipeNode,
		*parse.IfNode,
		*parse.RangeNode,
		*parse.WithNode,
		*parse.ListNode,
		*parse.ActionNode:
		return all()

	default:
		return all()
	}
}

// dotItem returns the list of items that should be suggested on dot
func dotItem(
	ctx Context,
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
		kind := protocol.CompletionItemKindVariable
		items = append(items, protocol.CompletionItem{
			Label:    ".",
			Kind:     &kind,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: "."},
		})
		prefix = "."
	}
	if lt := ctx.DotType; lt != nil {
		items = append(items, fieldCompletionItems(structFields(lt.DotType), prefix, wordRange)...)
		items = append(
			items,
			methodCompletionItems(namedMethods(lt.DotType), inputType, pipeKind, prefix, wordRange)...)
	}
	return items
}

// fieldCompletionItems returns the list of fields with or without the dot
func fieldCompletionItems(
	fields []TypeField,
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
func methodAcceptsInput(m MethodType, inputType types.Type, pipeKind outputKind) bool {
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
	methods []MethodType,
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

// varsToItems returns the list of variables
func varsToItems(ctx *Context, delSign bool, wordRange protocol.Range) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(ctx.Vars))
	kind := protocol.CompletionItemKindVariable
	for name := range ctx.Vars {
		if name == "$" {
			continue
		}
		label := name
		filter := name
		if delSign {
			label = name[1:]
		}
		items = append(items, protocol.CompletionItem{
			Label:      label,
			Kind:       &kind,
			FilterText: &filter,
			TextEdit:   &protocol.TextEdit{Range: wordRange, NewText: name},
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

func logPath(ctx *Context) {
	for i, node := range ctx.Path {
		log.Debug().
			Int("depth", i).
			Str("type", fmt.Sprintf("%T", node)).
			Str("content", node.String()).
			Msg("path")
	}
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
func resolvePipeDotType(pipe *parse.PipeNode, unwrapSlice bool, ctx *Context) *servertypes.Tree {
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
	for _, f := range structFields(ctx.DotType.DotType) {
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
	tree := servertypes.Tree{DotType: named, Pkg: ctx.DotType.Pkg}
	return &tree
}

// main traversal logic
func buildPathChildren(n parse.Node, target parse.Node, ctx *Context) bool {
	if isLeafNode(n) {
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
		for _, v := range n.Decl {
			if v == target {
				return true
			}
			ctx.Vars[v.Ident[0]] = n
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
