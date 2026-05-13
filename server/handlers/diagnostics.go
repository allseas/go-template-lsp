package handlers

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	templateLineRe = regexp.MustCompile(`template: [^:]*:(\d+):`)
	undefinedVarRe = regexp.MustCompile(`undefined variable "?(\$[a-zA-Z_][a-zA-Z0-9_]*)"?`)
)

// publishDiagnostics sends diagnostics to the LSP client.
func publishDiagnostics(ctx *glsp.Context, uri string, text string) {
	if ctx == nil {
		return
	}

	diagnostics := collectDiagnostics(text, uri)
	if diagnostics == nil {
		diagnostics = []protocol.Diagnostic{}
	}

	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

// collectDiagnostics runs strict syntax checking and semantic analysis on the template, returning all errors and warnings found.
func collectDiagnostics(text string, uri string) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	doc, ok := store.Get(uri)
	if !ok || doc.tree == nil {
		log.Debug().Msg("doc or tree is nil")
		return diagnostics
	}

	knownFunctions := buildKnownFunctions()
	masked := text

	for {
		t := parse.New("t")
		t.Mode = 0

		treeSet := map[string]*parse.Tree{}
		_, err := t.Parse(masked, "{{", "}}", treeSet, knownFunctions)
		if err == nil {
			break
		}

		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range:    findErrorRange(text, err),
			Message:  err.Error(),
			Severity: new(protocol.DiagnosticSeverityError),
		})

		newMasked := maskLine(masked, err)
		if newMasked == masked {
			break
		}
		masked = newMasked
	}

	seen := map[string]bool{}
	for _, node := range doc.tree.Root.Nodes {
		diagnostics = append(diagnostics, analyzeNode(node, text, seen)...)
	}

	return diagnostics
}

func buildKnownFunctions() map[string]any {
	knownFunctions := make(map[string]any)

	for _, fn := range globalFunctions {
		knownFunctions[fn] = func() {}
	}
	for _, kw := range templateKeywords {
		knownFunctions[kw] = func() {}
	}

	return knownFunctions
}

// maskLine replaces the line containing the parse error with spaces so the next parse iteration can find subsequent errors without re-reporting the same one.
func maskLine(text string, err error) string {
	lineIdx, ok := errorLineIndex(err)
	if !ok {
		return text
	}

	lines := strings.Split(text, "\n")
	if lineIdx >= 0 && lineIdx < len(lines) {
		lines[lineIdx] = strings.Repeat(" ", len(lines[lineIdx]))
	}

	return strings.Join(lines, "\n")
}

func errorLineIndex(err error) (int, bool) {
	m := templateLineRe.FindStringSubmatch(err.Error())
	if m == nil {
		return 0, false
	}

	lineIdx, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}

	return max(0, lineIdx-1), true
}

// analyzeNode recursively walks the AST and collects semantic warnings, passing the variable scope down into child nodes.
func analyzeNode(node parse.Node, text string, seen map[string]bool) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	switch n := node.(type) {
	case *parse.ActionNode:
		diagnostics = append(diagnostics, checkAction(n, text, seen)...)

	case *parse.IfNode:
		diagnostics = append(diagnostics, analyzeList(n.List, text, seen)...)
		diagnostics = append(diagnostics, analyzeList(n.ElseList, text, seen)...)

	case *parse.RangeNode:
		freshScope := map[string]bool{}
		diagnostics = append(diagnostics, analyzeList(n.List, text, freshScope)...)
		diagnostics = append(diagnostics, analyzeList(n.ElseList, text, freshScope)...)

	case *parse.WithNode:
		diagnostics = append(diagnostics, analyzeList(n.List, text, seen)...)
		diagnostics = append(diagnostics, analyzeList(n.ElseList, text, seen)...)

	case *parse.ListNode:
		diagnostics = append(diagnostics, analyzeList(n, text, seen)...)
	}

	return diagnostics
}

func analyzeList(list *parse.ListNode, text string, seen map[string]bool) []protocol.Diagnostic {
	if list == nil {
		return nil
	}

	var diagnostics []protocol.Diagnostic
	for _, child := range list.Nodes {
		diagnostics = append(diagnostics, analyzeNode(child, text, seen)...)
	}
	return diagnostics
}

// checkAction inspects a single action node for duplicate variable declarations.
func checkAction(
	action *parse.ActionNode,
	text string,
	seen map[string]bool,
) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	if action.Pipe == nil {
		return diagnostics
	}

	for _, decl := range action.Pipe.Decl {
		for _, ident := range decl.Ident {
			name := strings.TrimPrefix(ident, "$")
			if name == "" {
				continue
			}

			if seen[name] {
				diagnostics = append(diagnostics, protocol.Diagnostic{
					Range:    nodeToRange(decl, text),
					Message:  "duplicate variable declaration: $" + name,
					Severity: new(protocol.DiagnosticSeverityWarning),
				})
				continue
			}

			seen[name] = true
		}
	}

	return diagnostics
}

// findErrorRange extracts the range to underline from a parse error message.
// For undefined variable errors it underlines just the variable name; for all other errors it underlines the entire action block on the affected line.
func findErrorRange(text string, err error) protocol.Range {
	msg := err.Error()
	log.Debug().Str("err", msg).Msg("findErrorRange")

	if m := undefinedVarRe.FindStringSubmatch(msg); m != nil {
		varName := m[1]
		if lineIdx, ok := errorLineIndex(err); ok {
			lines := strings.Split(text, "\n")
			if lineIdx >= 0 && lineIdx < len(lines) {
				if col := strings.Index(lines[lineIdx], varName); col != -1 {
					return protocol.Range{
						Start: protocol.Position{Line: uint32(lineIdx), Character: uint32(col)},
						End: protocol.Position{
							Line:      uint32(lineIdx),
							Character: uint32(col + len(varName)),
						},
					}
				}
			}
		}
	}

	lineIdx, ok := errorLineIndex(err)
	if !ok {
		return protocol.Range{}
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return protocol.Range{}
	}

	if lineIdx >= len(lines) {
		lineIdx = len(lines) - 1
	}

	col, length := findBlockRange(lines[lineIdx])
	return protocol.Range{
		Start: protocol.Position{Line: uint32(lineIdx), Character: uint32(col)},
		End:   protocol.Position{Line: uint32(lineIdx), Character: uint32(col + length)},
	}
}

// findBlockRange returns the column and length of the action block on a given line, spanning from {{ to }} inclusive.
func findBlockRange(lineText string) (col, length int) {
	start := strings.Index(lineText, "{{")
	if start == -1 {
		for i, r := range lineText {
			if !unicode.IsSpace(r) {
				return i, 1
			}
		}
		return 0, 1
	}
	end := strings.Index(lineText[start:], "}}")
	if end == -1 {
		return start, len(lineText) - start
	}

	return start, end + 2
}
