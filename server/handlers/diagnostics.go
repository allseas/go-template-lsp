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
	severity DiagnosticsSeverity,
) (diagnostic protocol.Diagnostic) {
	sev := new(protocol.DiagnosticSeverityError)
	switch severity {
	case DiagnosticSeverityDisabled:
		sev = nil
	case DiagnosticSeverityError:
		sev = new(protocol.DiagnosticSeverityError)
	case DiagnosticSeverityWarning:
		sev = new(protocol.DiagnosticSeverityWarning)
	case DiagnosticSeverityInformation:
		sev = new(protocol.DiagnosticSeverityInformation)
	case DiagnosticSeverityHint:
		sev = new(protocol.DiagnosticSeverityHint)
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

	// Surface template argument type errors and hint load failures from the stored document.
	if doc, ok := store.Get(uri); ok {
		for _, tt := range doc.typedTrees {
			diagnostics = append(diagnostics, collectTemplateDiagnostics(tt, text)...)
		}
		diagnostics = append(diagnostics, collectHintLoadFailureDiagnostics(doc, text)...)
	}

	return diagnostics
}

func collectHintLoadFailureDiagnostics(doc *document, text string) []protocol.Diagnostic {
	if doc == nil || len(doc.failedHints) == 0 {
		return nil
	}
	severity := GetConfig().Diagnostics[types.ErrorHintLoadFailure]
	if severity == DiagnosticSeverityDisabled {
		return nil
	}
	var diagnostics []protocol.Diagnostic
	for treeName, errMsg := range doc.failedHints {
		isRoot := doc.tree != nil && doc.tree.Name == treeName
		offset := gotypeHintOffset(text, treeName, isRoot)
		if offset < 0 {
			continue
		}
		rng := expandToFullBracketsFromOffset(offset, text)
		diagnostics = append(diagnostics, createDiagnostic(
			"gotype: could not load type: "+errMsg,
			rng,
			severity,
		))
	}
	return diagnostics
}

// gotypeHintOffset returns the byte offset of the start of the "gotype:" token
func gotypeHintOffset(text, treeName string, isRoot bool) int {
	var searchIn string
	var searchStart int
	if isRoot {
		// Hint is on the first line of the file.
		nl := strings.IndexByte(text, '\n')
		if nl < 0 {
			searchIn = text
		} else {
			searchIn = text[:nl]
		}
		searchStart = 0
	} else {
		// Hint is on the line immediately after {{define "treeName"}}.
		defineLine := findDefineLine(text, treeName)
		if defineLine <= 0 {
			return -1
		}
		lineStart := lineStartOffset(text, defineLine+1)
		if lineStart < 0 {
			return -1
		}
		searchStart = lineStart
		nl := strings.IndexByte(text[lineStart:], '\n')
		if nl < 0 {
			searchIn = text[lineStart:]
		} else {
			searchIn = text[lineStart : lineStart+nl]
		}
	}
	rel := strings.Index(searchIn, "gotype:")
	if rel < 0 {
		return -1
	}
	return searchStart + rel
}

// lineStartOffset returns the byte offset of the start of the given 1-based line.
func lineStartOffset(text string, line int) int {
	if line <= 1 {
		if line == 1 {
			return 0
		}
		return -1
	}
	current := 1
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			current++
			if current == line {
				return i + 1
			}
		}
	}
	return -1
}

// collectTemplateDiagnostics collects TypeErrors entries from
// a typed tree, and converts them into protocol diagnostics.
func collectTemplateDiagnostics(typedTree *types.Tree, text string) []protocol.Diagnostic {
	if typedTree == nil {
		return nil
	}
	config := GetConfig()

	var diagnostics []protocol.Diagnostic
	for _, terr := range typedTree.TypeErrors {
		if terr.Node == nil || config.Diagnostics[terr.ErrType()] == DiagnosticSeverityDisabled {
			continue
		}
		pos := int(terr.Node.Position())
		rng := expandToFullBracketsFromOffset(pos, text)
		diagnostics = append(diagnostics, createDiagnostic(
			withPos(text, pos, terr.Err),
			rng,
			config.Diagnostics[terr.ErrType()],
		))
	}
	return diagnostics
}

// analyzeNode is the visitor passed to walkAndAnalyze; it declares variables then validates the node.
func analyzeNode(node parse.Node, text string, ctx *Context) (diagnostics []protocol.Diagnostic) {
	if ctx == nil || node == nil {
		return nil
	}
	declareNode(node, text, ctx)

	config := GetConfig()

	switch n := node.(type) {
	case *parse.UndefinedNode:
		if config.Diagnostics[types.ErrorSyntaxError] == DiagnosticSeverityDisabled {
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
			createDiagnostic(
				msg,
				expandToFullBracketsFromOffset(int(n.Position()), text),
				config.Diagnostics[types.ErrorSyntaxError],
			),
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

	case *parse.IdentifierNode:
		if config.Diagnostics[types.ErrorTypeInvalidFunction] != DiagnosticSeverityDisabled {
			if _, known := types.GlobalFuncs()[n.Ident]; !known {
				offset := int(n.Position())
				diagnostics = append(
					diagnostics,
					createDiagnostic(
						msgUnknownFunction(text, offset, n.Ident),
						expandToFullBracketsFromOffset(offset, text),
						config.Diagnostics[types.ErrorTypeInvalidFunction],
					),
				)
			}
		}

	case *parse.CommandNode:
		for _, arg := range n.Args {
			diagnostics = append(diagnostics, analyzeNode(arg, text, ctx)...)
		}
	}

	return diagnostics
}

// declareNode registers variable declarations into ctx.Vars before validation runs.
func declareNode(node parse.Node, text string, ctx *Context) {
	if ctx == nil || node == nil {
		return
	}
	switch n := node.(type) {
	case *parse.ActionNode:
		if n.Pipe != nil {
			collectDeclarations(n.Pipe, ctx)
		}
	case *parse.RangeNode:
		if n.Pipe != nil {
			collectDeclarations(n.Pipe, ctx)
		}
	case *parse.IfNode:
		if n.Pipe != nil {
			collectDeclarations(n.Pipe, ctx)
		}
	}
}

// collectDeclarations registers variables into ctx.Vars so checkPipeUsage can
// detect undeclared variable usage. Duplicate declaration detection is handled
// by the type analysis (ErrorDoubleDeclaredVariable in types.analyse).
func collectDeclarations(pipe *parse.PipeNode, ctx *Context) {
	if pipe == nil || ctx == nil {
		return
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
			if pipe.IsAssign {
				ctx.Vars[ident] = pipe
			} else {
				ctx.Vars[ident] = decl
			}
		}
	}
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
							GetConfig().Diagnostics[types.ErrorUndeclaredVariable],
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
