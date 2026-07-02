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

	// log.Debug().
	// 	Int("num diagnostics", len(diagnostics)).
	// 	Any("diagnostics", diagnostics).
	// 	Msg("publishDiagnostics")

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
		diagnostics = append(diagnostics, collectTemplateDiagnostics(tree, text)...)
	}

	// Surface hint load failures and empty define names from the stored document.
	if doc, ok := store.Get(uri); ok {
		diagnostics = append(diagnostics, collectHintLoadFailureDiagnostics(doc, text)...)
		diagnostics = append(diagnostics, collectEmptyDefineNameDiagnostics(doc, text)...)
	}

	return diagnostics
}

func collectHintLoadFailureDiagnostics(doc *document, text string) []protocol.Diagnostic {
	if doc == nil || len(doc.failedHints) == 0 {
		return nil
	}
	cfg := GetConfig().Diagnostics
	loadSev := cfg[types.ErrorHintLoadFailure]
	malformedSev := cfg[types.ErrorTypeMalformedHint]
	var diagnostics []protocol.Diagnostic

	for _, fh := range doc.failedHints {
		severity := loadSev
		msg := "gotype: could not load type: " + fh.Err
		if fh.Hint.IsMalformed() {
			severity = malformedSev
			msg = "gotype: " + fh.Err
		}
		if severity == DiagnosticSeverityDisabled {
			continue
		}
		offset := gotypeHintOffset(text, fh.Hint.Line)
		if offset < 0 {
			continue
		}
		rng := expandToFullBracketsFromOffset(offset, text)
		diagnostics = append(diagnostics, createDiagnostic(
			msg,
			rng,
			severity,
		))
	}
	return diagnostics
}

// collectEmptyDefineNameDiagnostics returns diagnostics for define blocks with empty names.
func collectEmptyDefineNameDiagnostics(doc *document, text string) []protocol.Diagnostic {
	if doc == nil || len(doc.typedTrees) == 0 {
		return nil
	}
	severity := GetConfig().Diagnostics[types.ErrorTypeEmptyDefineName]
	if severity == DiagnosticSeverityDisabled {
		return nil
	}

	var diagnostics []protocol.Diagnostic
	for name, t := range doc.typedTrees {
		if t == nil || t.Root == nil || name == "t" || name != "" {
			continue
		}

		bodyStart := int(t.Root.Position())
		blockEnd := int(t.End)

		// Search backward from the body start to find the nearest {{define.
		blockStart := bodyStart
		if ms := reDefine.FindAllStringIndex(text[:bodyStart], -1); ms != nil {
			blockStart = ms[len(ms)-1][0]
		}

		rng := bytesToRange(text, blockStart, blockEnd)
		diagnostics = append(diagnostics, createDiagnostic(
			"define block has an empty name",
			rng,
			severity,
		))
	}
	return diagnostics
}

// lineStartOffset returns the 0-based byte offset of the start of the 1-based lineNumber.
func lineStartOffset(text string, lineNumber int) int {
	if lineNumber <= 1 {
		return 0
	}
	currentLine := 1
	for offset, char := range text {
		if char == '\n' {
			currentLine++
			if currentLine == lineNumber {
				return offset + 1
			}
		}
	}
	return -1
}

// gotypeHintOffset returns the absolute byte offset of the "gotype:" token using the hint's line number.
func gotypeHintOffset(text string, hintLine int) int {
	lineStart := lineStartOffset(text, hintLine)
	if lineStart < 0 || lineStart >= len(text) {
		return -1
	}

	// Restrict search area to just this single line
	searchIn := text[lineStart:]
	if nl := strings.IndexByte(searchIn, '\n'); nl >= 0 {
		searchIn = searchIn[:nl]
	}

	rel := strings.Index(searchIn, "gotype:")
	if rel < 0 {
		return -1
	}
	return lineStart + rel
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

	case *types.ActionNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text)...)
		}

	case *types.RangeNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text)...)
		}

	case *types.IfNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text)...)
		}

	case *types.WithNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text)...)
		}

	case *types.IdentifierNode:
		// handled by analyse.go (ErrorTypeInvalidFunction)
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
						GetConfig().Diagnostics[types.ErrorUndeclaredVariable],
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
