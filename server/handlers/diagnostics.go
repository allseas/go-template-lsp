package handlers

import (
	"regexp"
	"strings"
	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	templateBlockRegex      = regexp.MustCompile(`\{\{-?\s*(.*?)\s*-?\}\}`)
	errorCleanPositionRegex = regexp.MustCompile(`\s*at position \d+`)
)

// publishDiagnostics runs a diagnostics analysis and notifies the client with any discovered warnings or errors.
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
	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

// collectDiagnostics builds the list of issues by combining tree analysis with regex-based text parsing rules to catch raw token anomalies.
func collectDiagnostics(text, uri string) (diagnostics []protocol.Diagnostic) {
	if doc, ok := store.Get(uri); ok && doc.tree != nil && doc.tree.Root != nil {
		ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
		diagnostics = walkAndAnalyze(doc.tree.Root, text, ctx, map[parse.Node]bool{})
	} else {
		log.Debug().Msg("doc, tree, or root is nil")
	}
	for _, match := range templateBlockRegex.FindAllStringIndex(text, -1) {
		if len(match) < 2 {
			continue
		}
		start, end := match[0], match[1]
		if start < 0 || end > len(text) || start >= end {
			continue
		}
		inner := strings.TrimSpace(
			strings.TrimSuffix(
				strings.TrimPrefix(strings.TrimSpace(text[start:end]), "{{-"),
				"-}}",
			),
		)

		inner = strings.TrimSpace(
			strings.TrimSuffix(
				strings.TrimPrefix(inner, "{{"),
				"}}",
			),
		)
		if inner == "" || strings.HasPrefix(inner, "/*") || inner == "end" || inner == "else" ||
			hasExactDiagnosticAtRange(diagnostics, start, end, text) {
			continue
		}
		if isUnparsedText(inner) || strings.ContainsAny(inner, "[]") {
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: protocol.Range{
					Start: offsetToPosition(text, start),
					End:   offsetToPosition(text, end),
				},
				Message:  "syntax error: unexpected token or unparseable action '" + inner + "'",
				Severity: new(protocol.DiagnosticSeverityError),
			})
		}
	}
	return diagnostics
}

// isUnparsedText checks if a text block contains invalid or unparseable template syntax.
func isUnparsedText(content string) bool {
	if strings.ContainsAny(content, `$.":|`+"`") || strings.Contains(content, ":=") {
		return false
	}
	return strings.Trim(content, "0123456789 ") != ""
}

// hasExactDiagnosticAtRange returns true if an existing diagnostic problem has already been reported within the exact range or line context specified.
func hasExactDiagnosticAtRange(
	diagnostics []protocol.Diagnostic,
	start, end int,
	text string,
) bool {
	pStart, pEnd := offsetToPosition(text, start), offsetToPosition(text, end)
	for _, d := range diagnostics {
		if d.Range.Start.Line == pStart.Line && d.Range.Start.Character >= pStart.Character &&
			d.Range.End.Character <= pEnd.Character {
			return true
		}
	}
	return false
}

// walkAndAnalyze recursively walks down the tree, keeping track of the scope context to run validation rules.
func walkAndAnalyze(
	node parse.Node,
	text string,
	ctx *Context,
	visited map[parse.Node]bool,
) (diagnostics []protocol.Diagnostic) {
	if node == nil || visited[node] {
		return nil
	}
	visited[node] = true
	defer delete(visited, node)
	if unode, ok := node.(*parse.UndefinedNode); ok {
		msg := strings.TrimSpace(
			errorCleanPositionRegex.ReplaceAllString(unode.String(), ""),
		)
		if msg == "" {
			msg = "undefined node"
		}
		return []protocol.Diagnostic{
			{
				Range:    expandToFullBracketsFromOffset(int(unode.Position()), text),
				Message:  msg,
				Severity: new(protocol.DiagnosticSeverityError),
			},
		}
	}
	switch n := node.(type) {
	case *parse.ListNode:
		for _, child := range n.Nodes {
			diagnostics = append(diagnostics, walkAndAnalyze(child, text, ctx, visited)...)
		}
	case *parse.ActionNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(n.Pipe, text, ctx)...)
			diagnostics = append(diagnostics, checkPipeUsage(n.Pipe, text, ctx)...)
			diagnostics = append(diagnostics, walkAndAnalyze(n.Pipe, text, ctx, visited)...)
		}
	case *parse.PipeNode:
		for _, v := range n.Decl {
			if len(v.Ident) > 0 {
				ctx.Vars[v.Ident[0]] = n
			}
		}
		prevPipe := ctx.Pipe
		ctx.Pipe = n
		for _, cmd := range n.Cmds {
			diagnostics = append(diagnostics, walkAndAnalyze(cmd, text, ctx, visited)...)
		}
		ctx.Pipe = prevPipe
	case *parse.CommandNode:
		if len(n.Args) > 0 {
			if identNode, ok := n.Args[0].(*parse.IdentifierNode); ok {
				funcName := identNode.Ident
				rng := expandToFullBracketsFromOffset(int(identNode.Position()), text)
				if _, exists := builtinOutput[funcName]; !exists {
					diagnostics = append(diagnostics, protocol.Diagnostic{
						Range:    rng,
						Message:  "unsupported function or unregistered command: " + funcName,
						Severity: new(protocol.DiagnosticSeverityError),
					})
				} else if currentKind := pipeOutputKind(ctx, false); currentKind != outputAny && currentKind != outputUntyped {
					isMatch := false
					for _, allowed := range functionsAccepting[currentKind] {
						if allowed == funcName {
							isMatch = true
							break
						}
					}
					if !isMatch {
						diagnostics = append(diagnostics, protocol.Diagnostic{
							Range:    rng,
							Message:  "type mismatch: function '" + funcName + "' does not accept piped data of this output kind",
							Severity: new(protocol.DiagnosticSeverityError),
						})
					}
				}
			}
		}
		for _, arg := range n.Args {
			diagnostics = append(diagnostics, walkAndAnalyze(arg, text, ctx, visited)...)
		}
	case *parse.RangeNode, *parse.IfNode, *parse.WithNode:
		pipe, list, elseList := extractBranchNodes(n)
		snapshot := snapshotVars(ctx.Vars)
		if _, isWith := n.(*parse.WithNode); !isWith && pipe != nil {
			diagnostics = append(diagnostics, collectDeclarations(pipe, text, ctx)...)
		}
		if pipe != nil {
			diagnostics = append(diagnostics, checkPipeUsage(pipe, text, ctx)...)
			diagnostics = append(diagnostics, walkAndAnalyze(pipe, text, ctx, visited)...)
		}
		diagnostics = append(diagnostics, walkAndAnalyze(list, text, ctx, visited)...)
		ctx.Vars = snapshot
		diagnostics = append(diagnostics, walkAndAnalyze(elseList, text, ctx, visited)...)
		ctx.Vars = snapshot
	}

	return diagnostics
}

// collectDeclarations registers new template variables in the current scope context, flagging if a variable is illegally re-declared.
func collectDeclarations(
	pipe *parse.PipeNode,
	text string,
	ctx *Context,
) (diagnostics []protocol.Diagnostic) {
	if pipe == nil {
		return nil
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
				diagnostics = append(diagnostics, protocol.Diagnostic{
					Range:    nodeToRange(decl, text),
					Message:  "duplicate variable declaration: " + ident,
					Severity: new(protocol.DiagnosticSeverityWarning),
				})
				continue
			}
			ctx.Vars[ident] = pipe
		}
	}
	return diagnostics
}

// checkPipeUsage scans through values to ensure that any references to custom variables have been explicitly defined beforehand.
func checkPipeUsage(
	pipe *parse.PipeNode,
	text string,
	ctx *Context,
) (diagnostics []protocol.Diagnostic) {
	if pipe == nil {
		return nil
	}
	for _, cmd := range pipe.Cmds {
		if cmd == nil {
			continue
		}
		for _, arg := range cmd.Args {
			if vnode, ok := arg.(*parse.VariableNode); ok && len(vnode.Ident) > 0 {
				if name := vnode.Ident[0]; name != "" && name != "$" && ctx.Vars[name] == nil {
					diagnostics = append(diagnostics, protocol.Diagnostic{
						Range:    nodeToRange(vnode, text),
						Message:  "undefined variable: " + name,
						Severity: new(protocol.DiagnosticSeverityError),
					})
				}
			}
		}
	}
	return diagnostics
}

// expandToFullBracketsFromOffset computes a Range including the {{ and }}.
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

// extractBranchNodes unifies access to child properties for branch structural entities including Range, If, and With statement blocks.
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
