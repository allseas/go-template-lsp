package handlers

import (
	"strings"
	parse "text-template-parser"
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
	var trees map[string]*parse.Tree

	if doc, ok := store.Get(uri); ok && len(doc.trees) > 0 {
		trees = doc.trees
	} else {
		_, parsed, err := tryParse(text)
		if err != nil {
			log.Debug().Err(err).Msg("failed to parse template")
		}
		trees = parsed
	}

	if len(trees) == 0 {
		log.Debug().Msg("no parse trees available for diagnostics")
		return diagnostics
	}

	for _, tree := range trees {
		if tree == nil || tree.Root == nil {
			continue
		}
		ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
		diagnostics = append(
			diagnostics,
			walkAndAnalyze(tree.Root, text, ctx, map[parse.Node]bool{}, analyzeNode)...,
		)
	}

	return diagnostics
}

// collectTemplateArgTypeDiagnostics converts ErrorTypeInvalidTemplateArg entries from
// the typed tree into protocol diagnostics.
func collectTemplateArgTypeDiagnostics(typedTree *types.Tree, text string) []protocol.Diagnostic {
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

	// Surface template argument type errors from the typed tree.
	if typedTree != nil {
		diagnostics = append(diagnostics, collectTemplateArgTypeDiagnostics(typedTree, text)...)
	}

	return diagnostics
}

// collectTemplateArgTypeDiagnostics converts ErrorTypeInvalidTemplateArg entries from
// the typed tree into protocol diagnostics.
func collectTemplateArgTypeDiagnostics(typedTree *types.Tree, text string) []protocol.Diagnostic {
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

// analyzeNode is the visitor passed to walkAndAnalyze; it declares variables then validates the node.
func analyzeNode(node parse.Node, text string, ctx *Context) (diagnostics []protocol.Diagnostic) {
	if ctx == nil || node == nil {
		return nil
	}
	diagnostics = append(diagnostics, declareNode(node, text, ctx)...)

	switch n := node.(type) {
	case *parse.UndefinedNode:
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

	case *parse.ActionNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
		}

	case *parse.RangeNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
		}

	case *parse.IfNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
		}

	case *parse.WithNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
		}

	case *parse.CommandNode:
		if len(n.Args) > 0 {
			if identNode, ok := n.Args[0].(*parse.IdentifierNode); ok {
				funcName := identNode.Ident
				rng := expandToFullBracketsFromOffset(int(identNode.Position()), text)
				offset := int(identNode.Position())
				if _, known := types.GlobalFuncs()[funcName]; !known {
					diagnostics = append(
						diagnostics,
						createDiagnostic(msgUnknownFunction(text, offset, funcName), rng, true),
					)
				} else {
					currentKind := pipeOutputKind(ctx, false)
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
func declareNode(node parse.Node, text string, ctx *Context) (diagnostics []protocol.Diagnostic) {
	if ctx == nil || node == nil {
		return nil
	}
	switch n := node.(type) {
	case *parse.ActionNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text, ctx)...)
		}
	case *parse.RangeNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text, ctx)...)
		}
	case *parse.IfNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text, ctx)...)
		}
	}
	return diagnostics
}

// collectDeclarations registers variables into ctx.Vars and flags duplicate := declarations.
func collectDeclarations(
	pipe *parse.PipeNode,
	text string,
	ctx *Context,
) (diagnostics []protocol.Diagnostic) {
	if pipe == nil || ctx == nil {
		return nil
	}
	if ctx.Vars == nil {
		ctx.Vars = make(map[string]parse.Node)
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
				diagnostics = append(
					diagnostics,
					createDiagnostic(
						msgDuplicateDeclaration(text, int(decl.Position()), ident),
						nodeToRange(decl, text),
						false,
					),
				)
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
	pipe *parse.PipeNode,
	text string,
	ctx *Context,
) (diagnostics []protocol.Diagnostic) {
	if pipe == nil || ctx == nil {
		return nil
	}
	if ctx.Vars == nil {
		ctx.Vars = make(map[string]parse.Node)
	}
	for _, cmd := range pipe.Cmds {
		if cmd == nil {
			continue
		}
		for _, arg := range cmd.Args {
			if vnode, ok := arg.(*parse.VariableNode); ok && len(vnode.Ident) > 0 {
				if name := vnode.Ident[0]; name != "" && name != "$" && ctx.Vars[name] == nil {
					diagnostics = append(
						diagnostics,
						createDiagnostic(
							msgUndeclaredVariable(text, int(vnode.Position()), name),
							nodeToRange(vnode, text),
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
func extractBranchNodes(node parse.Node) (*parse.PipeNode, *parse.ListNode, *parse.ListNode) {
	switch n := node.(type) {
	case *parse.IfNode:
		return n.Pipe, n.List, n.ElseList
	case *parse.RangeNode:
		return n.Pipe, n.List, n.ElseList
	case *parse.WithNode:
		return n.Pipe, n.List, n.ElseList
	default:
		return nil, nil, nil
	}
}
