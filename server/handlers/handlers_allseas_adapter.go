package handlers

import (
	parse "text-template-parser"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// walkTable walks the Pipe and List children of a TableNode, invoking walk on each.
func walkTable(node *parse.TableNode, walk func(parse.Node)) {
	walk(node.Pipe)
	walk(node.List)
}

// buildPathTable builds the context path through a TableNode.
// TODO: should be removed later because just needed for a deprecated context
func buildPathTable(n *parse.TableNode, target parse.Node, ctx *Context) bool {
	prevDot := ctx.DotType
	ctx.DotType = resolvePipeDotType(n.Pipe, false, ctx)
	found := buildPathBranch(n.Pipe, n.List, nil, target, ctx)
	if !found {
		ctx.DotType = prevDot
	}
	return found
}

// walkAndAnalyzeTable recursively walks and analyzes a TableNode, maintaining scope
// TODO: should be removed later because just needed for a deprecated context
func walkAndAnalyzeTable(
	n *parse.TableNode,
	text string,
	ctx *Context,
	visited map[parse.Node]bool,
	fn func(parse.Node, string, *Context) []protocol.Diagnostic,
) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic
	if ctx.Vars == nil {
		ctx.Vars = make(map[string]parse.Node)
	}
	snapshot := snapshotVars(ctx.Vars)
	if n.Pipe != nil {
		diagnostics = append(diagnostics, walkAndAnalyze(n.Pipe, text, ctx, visited, fn)...)
	}
	if n.List != nil {
		diagnostics = append(diagnostics, walkAndAnalyze(n.List, text, ctx, visited, fn)...)
	}
	ctx.Vars = snapshot
	return diagnostics
}
