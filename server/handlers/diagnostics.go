package handlers

import (
	"strings"
	"text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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
		types.Inspect(tree.Root, func(n types.Node) bool {
			diagnostics = append(diagnostics, analyzeNode(n, text)...)
			return true
		})
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

// analyzeNode validates a single typed node and returns any diagnostics
// raised against it. Scope (visible variables, enclosing pipe) is derived
// from the node's parent chain rather than threaded through a context.
func analyzeNode(node types.Node, text string) (diagnostics []protocol.Diagnostic) {
	if node == nil {
		return nil
	}
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
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text)...)
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text)...)
		}

	case *types.RangeNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text)...)
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text)...)
		}

	case *types.IfNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text)...)
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text)...)
		}

	case *types.WithNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text)...)
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
					currentKind := pipeOutputKind(types.EnclosingPipe(n), false)
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

// collectDeclarations flags duplicate := declarations in pipe by comparing its
// decls against the variables already visible at the pipe's position. Names
// that pipe itself introduces are not registered separately: each call works
// with on-demand scope derived from the typed tree's parent chain.
func collectDeclarations(
	pipe *types.PipeNode,
	text string,
) (diagnostics []protocol.Diagnostic) {
	if pipe == nil || pipe.IsAssign {
		return nil
	}
	visible := visibleVarNames(pipe)
	seen := map[string]bool{}
	for _, decl := range pipe.Decl {
		if decl == nil {
			continue
		}
		for _, ident := range decl.Ident {
			if name := strings.TrimPrefix(ident, "$"); name == "" {
				continue
			}
			isDup := visible[ident] || seen[ident]
			seen[ident] = true
			if !isDup {
				continue
			}
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
		}
	}
	return diagnostics
}

// checkPipeUsage flags any variable references in the pipe that are not
// declared in any visible scope. The pipe's own decls count as visible (to
// preserve historical behaviour where {{$x := $x}} did not flag the RHS).
func checkPipeUsage(
	pipe *types.PipeNode,
	text string,
) (diagnostics []protocol.Diagnostic) {
	if pipe == nil {
		return nil
	}
	visible := visibleVarNames(pipe)
	// $ is always implicitly available.
	visible["$"] = true
	// Merge the pipe's own decls so the RHS can reference them.
	for _, decl := range pipe.Decl {
		if decl == nil {
			continue
		}
		for _, ident := range decl.Ident {
			visible[ident] = true
		}
	}
	for _, cmd := range pipe.Cmds {
		if cmd == nil {
			continue
		}
		for _, arg := range cmd.Args {
			if vnode, ok := arg.(*types.VariableNode); ok && len(vnode.Ident) > 0 {
				name := vnode.Ident[0]
				if name == "" || visible[name] {
					continue
				}
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
	return diagnostics
}

// visibleVarNames returns the set of variable identifiers (with leading "$")
// in scope at n, derived from types.VisibleVarsAt.
func visibleVarNames(n types.Node) map[string]bool {
	vars := types.VisibleVarsAt(n)
	out := make(map[string]bool, len(vars))
	for _, v := range vars {
		if v == nil {
			continue
		}
		for _, ident := range v.Ident {
			out[ident] = true
		}
	}
	return out
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
