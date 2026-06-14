package handlers

import (
	"strings"
	"text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// diagCtx tracks the per-walk scope state for diagnostics traversal.
// It is the typed counterpart of Context, scoped to diagnostics only.
type diagCtx struct {
	Vars map[string]types.Node
	Pipe *types.PipeNode
}

func createDiagnostic(
	msg string,
	rang protocol.Range,
	isError bool,
) (diagnostic protocol.Diagnostic) {
	sev := new(protocol.DiagnosticSeverityError)
	if !isError {
		sev = new(protocol.DiagnosticSeverityWarning)
	}

	source := "text-template-support"

	diagnostic = protocol.Diagnostic{
		Range:    rang,
		Message:  msg,
		Severity: sev,
		Source:   &source,
	}

	return
}

// publishDiagnostics analyzes the document and sends diagnostics to the client.
func publishDiagnostics(ctx *glsp.Context, uri, text string) {
	if ctx == nil {
		return
	}

	if !GetConfig().EnableDiagnostics {
		ctx.Notify(
			protocol.ServerTextDocumentPublishDiagnostics,
			&protocol.PublishDiagnosticsParams{
				URI:         uri,
				Diagnostics: []protocol.Diagnostic{},
			},
		)
		return
	}

	diagnostics := collectDiagnostics(text, uri)
	if diagnostics == nil {
		diagnostics = []protocol.Diagnostic{}
	}
	for i := range diagnostics {
		if strings.TrimSpace(diagnostics[i].Message) == "" {
			diagnostics[i].Message = "unknown diagnostic"
		}
	}

	log.Debug().
		Int("num diagnostics", len(diagnostics)).
		Any("diagnostics", diagnostics).
		Msg("publishDiagnostics")

	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

// collectDiagnostics returns diagnostics from AST analysis using the improved parser.
func collectDiagnostics(text, uri string) (diagnostics []protocol.Diagnostic) {
	var trees map[string]*types.Tree

	if doc, ok := store.Get(uri); ok && len(doc.typedTrees) > 0 {
		trees = doc.typedTrees
	} else {
		_, parsed, err := tryParse(text)
		if err != nil {
			log.Debug().Err(err).Msg("failed to parse template")
		}
		trees = make(map[string]*types.Tree, len(parsed))
		for name, tr := range parsed {
			trees[name] = buildTypedTree(tr, nil, nil)
		}
	}

	if len(trees) == 0 {
		log.Debug().Msg("no parse trees available for diagnostics")
		return diagnostics
	}

	for _, tree := range trees {
		if tree == nil || tree.Root == nil {
			continue
		}
		ctx := &diagCtx{Vars: map[string]types.Node{"$": nil}}
		diagnostics = append(
			diagnostics,
			walkAndAnalyzeTyped(tree.Root, text, ctx, map[types.Node]bool{}, analyzeNode)...,
		)
	}

	// Surface template argument type errors from the typed trees.
	if doc, ok := store.Get(uri); ok {
		for _, tt := range doc.typedTrees {
			diagnostics = append(diagnostics, collectTemplateArgTypeDiagnostics(tt, text)...)
		}
	}

	return diagnostics
}

// collectTemplateArgTypeDiagnostics converts ErrorTypeInvalidTemplateArg entries from
// a typed tree into protocol diagnostics.
func collectTemplateArgTypeDiagnostics(typedTree *types.Tree, text string) []protocol.Diagnostic {
	if typedTree == nil {
		return nil
	}
	var diagnostics []protocol.Diagnostic
	for _, terr := range typedTree.TypeErrors {
		if terr.ErrType() != types.ErrorTypeInvalidTemplateArg {
			continue
		}
		if terr.Node == nil {
			continue
		}
		pos := int(terr.Node.Position())
		rng := expandToFullBracketsFromOffset(pos, text)
		diagnostics = append(diagnostics, createDiagnostic(
			withPos(text, pos, terr.Err),
			rng,
			true,
		))
	}
	return diagnostics
}

// analyzeNode is the visitor passed to walkAndAnalyzeTyped; it declares variables then validates the node.
func analyzeNode(node types.Node, text string, ctx *diagCtx) (diagnostics []protocol.Diagnostic) {
	if ctx == nil || node == nil {
		return nil
	}
	diagnostics = append(diagnostics, declareNode(node, text, ctx)...)

	config := GetConfig()

	switch n := node.(type) {
	case *types.UndefinedNode:
		if !config.Diagnostics.SyntaxError {
			break
		}
		if n.Err == nil && strings.TrimSpace(n.String()) == "" {
			// Structural artifact with no attached error: skip.
			break
		}
		var msg string
		if n.Err != nil {
			msg = n.Err.Error()
		} else {
			msg = msgParseError(text, int(n.Position()), strings.TrimSpace(n.String()))
		}
		diagnostics = append(
			diagnostics,
			createDiagnostic(msg, expandToFullBracketsFromOffset(int(n.Position()), text), true),
		)

	case *types.ActionNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
		}

	case *types.RangeNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
		}

	case *types.IfNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
		}

	case *types.WithNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
		}

	case *types.CommandNode:
		if !config.Diagnostics.IncorrectFunction {
			break
		}
		if len(n.Args) > 0 {
			if identNode, ok := n.Args[0].(*types.IdentifierNode); ok {
				funcName := identNode.Ident
				rng := expandToFullBracketsFromOffset(int(identNode.Position()), text)
				offset := int(identNode.Position())
				if _, known := types.GlobalFuncs()[funcName]; !known {
					diagnostics = append(
						diagnostics,
						createDiagnostic(msgUnknownFunction(text, offset, funcName), rng, true),
					)
				} else {
					currentKind := pipeOutputKind(ctx.Pipe, false)
					if currentKind != outputAny && currentKind != outputUntyped {
						if !funcAcceptsKind(funcName, currentKind) {
							msg := msgTypeMismatch(text, offset, funcName)
							diagnostics = append(
								diagnostics,
								createDiagnostic(msg, rng, true),
							)
						}
					}
				}
			}
		}
	}

	return diagnostics
}

// declareNode registers variable declarations into ctx.Vars before validation runs.
func declareNode(node types.Node, text string, ctx *diagCtx) (diagnostics []protocol.Diagnostic) {
	if ctx == nil || node == nil {
		return nil
	}
	switch n := node.(type) {
	case *types.ActionNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text, ctx)...)
		}
	case *types.RangeNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text, ctx)...)
		}
	case *types.IfNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text, ctx)...)
		}
	}
	return diagnostics
}

// collectDeclarations registers variables into ctx.Vars and flags duplicate := declarations.
func collectDeclarations(
	pipe *types.PipeNode,
	text string,
	ctx *diagCtx,
) (diagnostics []protocol.Diagnostic) {
	if pipe == nil || ctx == nil {
		return nil
	}
	if ctx.Vars == nil {
		ctx.Vars = make(map[string]types.Node)
	}
	for _, decl := range pipe.Decl {
		if decl == nil {
			continue
		}
		for _, ident := range decl.Ident {
			if name := strings.TrimPrefix(ident, "$"); name == "" {
				continue
			}
			if ctx.Vars[ident] != nil && !pipe.IsAssign {
				if GetConfig().Diagnostics.VariableRedeclaration {
					diagnostics = append(
						diagnostics,
						createDiagnostic(
							msgDuplicateDeclaration(text, int(decl.Position()), ident),
							nodeRange(decl, text),
							false,
						),
					)
				}
				continue
			}
			if pipe.IsAssign {
				ctx.Vars[ident] = pipe
			} else {
				ctx.Vars[ident] = decl
			}
		}
	}
	return diagnostics
}

// checkPipeUsage flags any variable references in the pipe that were not previously declared.
func checkPipeUsage(
	pipe *types.PipeNode,
	text string,
	ctx *diagCtx,
) (diagnostics []protocol.Diagnostic) {
	if pipe == nil || ctx == nil {
		return nil
	}
	if ctx.Vars == nil {
		ctx.Vars = make(map[string]types.Node)
	}
	for _, cmd := range pipe.Cmds {
		if cmd == nil {
			continue
		}
		for _, arg := range cmd.Args {
			if vnode, ok := arg.(*types.VariableNode); ok && len(vnode.Ident) > 0 {
				if name := vnode.Ident[0]; name != "" && name != "$" && ctx.Vars[name] == nil {
					diagnostics = append(
						diagnostics,
						createDiagnostic(
							msgUndeclaredVariable(text, int(vnode.Position()), name),
							nodeRange(vnode, text),
							true,
						),
					)
				}
			}
		}
	}
	return diagnostics
}

// expandToFullBracketsFromOffset returns a Range that includes the surrounding {{ and }}.
func expandToFullBracketsFromOffset(pos int, text string) protocol.Range {
	startOffset, endOffset := pos, pos
	if pos < len(text) {
		if openIdx := strings.LastIndex(text[:pos], "{{"); openIdx != -1 {
			startOffset = openIdx
		}
		if closeIdx := strings.Index(text[pos:], "}}"); closeIdx != -1 {
			endOffset = pos + closeIdx + 2
		} else if nextLine := strings.IndexByte(text[pos:], '\n'); nextLine != -1 {
			endOffset = pos + nextLine
		} else {
			endOffset = len(text)
		}
	}
	if startOffset >= endOffset {
		endOffset = startOffset + 1
	}
	return protocol.Range{
		Start: offsetToPosition(text, startOffset),
		End:   offsetToPosition(text, endOffset),
	}
}

// extractBranchNodes returns the pipe, list, and else-list for if/range/with nodes.
func extractBranchNodes(node types.Node) (*types.PipeNode, *types.ListNode, *types.ListNode) {
	switch n := node.(type) {
	case *types.IfNode:
		return n.Pipe, n.List, n.ElseList
	case *types.RangeNode:
		return n.Pipe, n.List, n.ElseList
	case *types.WithNode:
		return n.Pipe, n.List, n.ElseList
	default:
		return nil, nil, nil
	}
}

// walkAndAnalyzeTyped recursively walks the typed node tree, maintaining scope
// context, and calls fn on every node. It is the typed counterpart of
// walkAndAnalyze used during the diagnostics pass.
func walkAndAnalyzeTyped(
	node types.Node,
	text string,
	ctx *diagCtx,
	visited map[types.Node]bool,
	fn func(types.Node, string, *diagCtx) []protocol.Diagnostic,
) (diagnostics []protocol.Diagnostic) {
	if node == nil || visited[node] {
		return nil
	}
	if ctx == nil {
		ctx = &diagCtx{Vars: make(map[string]types.Node)}
	}
	visited[node] = true
	defer delete(visited, node)

	diagnostics = append(diagnostics, fn(node, text, ctx)...)

	switch n := node.(type) {
	case *types.ListNode:
		for _, child := range n.Nodes {
			diagnostics = append(diagnostics, walkAndAnalyzeTyped(child, text, ctx, visited, fn)...)
		}
	case *types.ActionNode:
		if n.Pipe != nil {
			diagnostics = append(
				diagnostics,
				walkAndAnalyzeTyped(n.Pipe, text, ctx, visited, fn)...)
		}
	case *types.PipeNode:
		if ctx.Vars == nil {
			ctx.Vars = make(map[string]types.Node)
		}
		for _, v := range n.Decl {
			if v != nil && len(v.Ident) > 0 {
				ctx.Vars[v.Ident[0]] = n
			}
		}
		prevPipe := ctx.Pipe
		ctx.Pipe = n
		for _, cmd := range n.Cmds {
			diagnostics = append(diagnostics, walkAndAnalyzeTyped(cmd, text, ctx, visited, fn)...)
		}
		ctx.Pipe = prevPipe
	case *types.CommandNode:
		for _, arg := range n.Args {
			diagnostics = append(diagnostics, walkAndAnalyzeTyped(arg, text, ctx, visited, fn)...)
		}
	case *types.RangeNode, *types.IfNode, *types.WithNode:
		pipe, list, elseList := extractBranchNodes(n)
		if ctx.Vars == nil {
			ctx.Vars = make(map[string]types.Node)
		}
		snapshot := snapshotDiagVars(ctx.Vars)
		if pipe != nil {
			diagnostics = append(diagnostics, walkAndAnalyzeTyped(pipe, text, ctx, visited, fn)...)
		}
		if list != nil {
			diagnostics = append(diagnostics, walkAndAnalyzeTyped(list, text, ctx, visited, fn)...)
		}
		ctx.Vars = snapshot
		if elseList != nil {
			diagnostics = append(
				diagnostics,
				walkAndAnalyzeTyped(elseList, text, ctx, visited, fn)...)
		}
		ctx.Vars = snapshot
	case *types.TableNode:
		diagnostics = append(diagnostics, walkAndAnalyzeTypedTable(n, text, ctx, visited, fn)...)
	}

	return diagnostics
}

// walkAndAnalyzeTypedTable recursively walks and analyzes a typed TableNode,
// maintaining variable scope around the table body. When TableNode is the
// !allseas stub (no real table syntax in templates), this branch is reachable
// only via crafted ASTs and is a no-op for ordinary inputs.
func walkAndAnalyzeTypedTable(
	n *types.TableNode,
	text string,
	ctx *diagCtx,
	visited map[types.Node]bool,
	fn func(types.Node, string, *diagCtx) []protocol.Diagnostic,
) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic
	if ctx.Vars == nil {
		ctx.Vars = make(map[string]types.Node)
	}
	snapshot := snapshotDiagVars(ctx.Vars)
	if n.Pipe != nil {
		diagnostics = append(diagnostics, walkAndAnalyzeTyped(n.Pipe, text, ctx, visited, fn)...)
	}
	if n.List != nil {
		diagnostics = append(diagnostics, walkAndAnalyzeTyped(n.List, text, ctx, visited, fn)...)
	}
	ctx.Vars = snapshot
	return diagnostics
}

// snapshotDiagVars copies the scope-variables map so it can be restored after
// descending into a branch.
func snapshotDiagVars(vars map[string]types.Node) map[string]types.Node {
	snap := make(map[string]types.Node, len(vars))
	for k, v := range vars {
		snap[k] = v
	}
	return snap
}
