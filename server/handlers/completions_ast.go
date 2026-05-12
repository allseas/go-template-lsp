// Package handlers provides a Language Server Protocol implementation for Go text/templates, featuring scope-aware variable completion and built-in function support.
package handlers

import (
	"fmt"
	parse "text-template-parser"

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

	ctx := &Context{Vars: map[string]parse.Node{"$": nil}}

	offset := positionToOffset(text, params.Position)

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
		log.Debug().
			Int("offset new ", offset).
			Msg("completion: cursor is not inside a template block, skipping")
		return nil
	}

	if !result {
		log.Error().Msg("The target node is not found")
		return nil
	}

	log.Debug().Msg(tree.Root.String())
	log.Debug().
		Str("type cur", fmt.Sprintf("%T, %c", curNode, text[curNode.Position()])).
		Msg(curNode.String())
	if parent == nil {
		log.Debug().Msg("parent is nil")
	} else {
		log.Debug().Str("type", fmt.Sprintf("%T", parent)).Msg(parent.String())
	}
	items := suggest(curNode, parent, ctx, text[curNode.Position()], isInvoked)

	return protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

var functionsAccepting = map[outputKind][]string{
	outputInt: {
		"eq", "ne", "lt", "le", "gt", "ge",
		"not",
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
	precedingIdx := len(cmds) - 2

	log.Debug().Int("%d", precedingIdx).Msg("THIS IS A PRECEDING INDEX")
	if isInvoked {
		precedingIdx = len(cmds) - 1
	}
	if precedingIdx < 0 || len(cmds) < 1 {
		return outputAny
	}
	preceding := cmds[precedingIdx]
	log.Debug().Msg(preceding.String() + "THIS IS A PRECEDING NODE")
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

func pipeFilteredItems(kind outputKind, ctx *Context) []protocol.CompletionItem {
	names, ok := functionsAccepting[kind]
	if !ok || kind == outputUntyped {
		// type is unknown, might be because we don't know the type of the dot object or overlooked function
		return append(append(dotItem(), varsToItems(ctx, false)...), builtinItems()...)
	}

	fnKind := protocol.CompletionItemKindFunction
	items := make([]protocol.CompletionItem, 0, len(names)+1+len(ctx.Vars))

	items = append(items, dotItem()...)
	items = append(items, varsToItems(ctx, false)...)
	for _, name := range names {
		n := name
		items = append(items, protocol.CompletionItem{
			Label: n,
			Kind:  &fnKind,
		})
	}
	return items
}

func suggest(
	cur parse.Node,
	parent parse.Node,
	ctx *Context,
	sChar uint8,
	isInvoked bool,
) []protocol.CompletionItem {
	if sChar == '$' {
		return varsToItems(ctx, true)
	}

	if sChar == '.' {
		return dotItem()
	}

	all := func() []protocol.CompletionItem {
		if kind := pipeOutputKind(ctx, isInvoked); kind != outputAny {
			return pipeFilteredItems(kind, ctx)
		}
		return append(append(dotItem(), varsToItems(ctx, false)...), builtinItems()...)
	}
	dotAndVars := func() []protocol.CompletionItem {
		return append(dotItem(), varsToItems(ctx, false)...)
	}

	switch p := parent.(type) {
	case *parse.CommandNode:
		if len(p.Args) > 0 && p.Args[0] == cur {
			return builtinItems()
		}
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

func dotItem() []protocol.CompletionItem {
	kind := protocol.CompletionItemKindVariable
	return []protocol.CompletionItem{{
		Label: ".",
		Kind:  &kind,
	}}
}

func varsToItems(ctx *Context, delSign bool) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(ctx.Vars))
	kind := protocol.CompletionItemKindVariable
	for name := range ctx.Vars {
		label := name
		if delSign {
			label = name[1:]
		}
		items = append(items, protocol.CompletionItem{
			Label: label,
			Kind:  &kind,
		})
	}
	return items
}

func builtinItems() []protocol.CompletionItem {
	statics := []string{
		"and", "call", "html", "index", "slice", "js", "len",
		"not", "or", "print", "printf", "println", "urlquery",
		"eq", "ne", "lt", "le", "gt", "ge", "if", "range",
	}
	kind := protocol.CompletionItemKindFunction
	items := make([]protocol.CompletionItem, 0, len(statics))
	for _, name := range statics {
		items = append(items, protocol.CompletionItem{
			Label: name,
			Kind:  &kind,
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
		return buildPathBranch(n.Pipe, n.List, n.ElseList, target, ctx)

	case *parse.WithNode:
		return buildPathBranch(n.Pipe, n.List, n.ElseList, target, ctx)

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
