// Package main provides a Language Server Protocol implementation
// for Go text/templates, featuring scope-aware variable completion
// and built-in function support.
package main

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	variablePattern = regexp.MustCompile(`\$[a-zA-Z_][a-zA-Z0-9_]*`)

	globalFunctions = []string{
		"len", "index", "slice", "print", "printf", "println",
		"urlquery", "html", "js", "eq", "ne", "lt", "gt", "le", "ge",
		"and", "or", "not", "call",
	}
)

// completion handles LSP "textDocument/completion" requests by identifying
// the current template context and returning relevant variables and functions.
func completion(_ *glsp.Context, params *protocol.CompletionParams) (any, error) {
	text, ok := store.Get(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	offset := positionToOffset(text, params.Position)
	if !isInsideTemplate(text, offset) {
		return nil, nil
	}

	currentWord := getWordAtOffset(text, offset)
	startChar := int(params.Position.Character) - len(currentWord)
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

	vars := extractVariables(text, offset)

	items := make([]protocol.CompletionItem, 0, len(vars)+len(globalFunctions))
	seen := make(map[string]bool)

	varKind := protocol.CompletionItemKindVariable
	fnKind := protocol.CompletionItemKindFunction

	for _, v := range vars {
		if seen[v] {
			continue
		}
		seen[v] = true

		items = append(items, protocol.CompletionItem{
			Label:    v,
			Kind:     &varKind,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: v},
		})
	}

	for _, fn := range globalFunctions {
		items = append(items, protocol.CompletionItem{
			Label:    fn,
			Kind:     &fnKind,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: fn},
		})
	}

	return protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// extractVariables scans the template text up to the cursor to build a scope-aware list of defined variables.
func extractVariables(text string, cursor int) []string {
	if cursor > len(text) {
		cursor = len(text)
	}
	if cursor < 0 {
		cursor = 0
	}

	visible := make(map[string]int)
	scopeStack := []map[string]struct{}{
		{},
	}
	order := make([]string, 0)
	addVisible := func(name string) {
		if name == "" {
			return
		}
		if visible[name] == 0 {
			order = append(order, name)
		}
		visible[name]++
	}
	removeVisible := func(name string) {
		if visible[name] > 0 {
			visible[name]--
		}
	}
	addToCurrentScope := func(name string) {
		if name == "" {
			return
		}
		cur := scopeStack[len(scopeStack)-1]
		cur[name] = struct{}{}
		addVisible(name)
	}
	pushScope := func() {
		scopeStack = append(scopeStack, map[string]struct{}{})
	}
	popScope := func() {
		if len(scopeStack) <= 1 {
			return
		}
		top := scopeStack[len(scopeStack)-1]
		scopeStack = scopeStack[:len(scopeStack)-1]
		for name := range top {
			removeVisible(name)
		}
	}

	i := 0
	for i < cursor {
		openRel := strings.Index(text[i:cursor], "{{")
		if openRel == -1 {
			break
		}
		open := i + openRel
		if open+2 > cursor {
			break
		}

		closeRel := strings.Index(text[open+2:cursor], "}}")
		end := cursor
		hasClose := false
		if closeRel != -1 {
			end = open + 2 + closeRel
			hasClose = true
		}

		action := text[open+2 : end]
		trimmed := strings.TrimSpace(action)

		if trimmed != "" && !strings.HasPrefix(trimmed, "/*") {
			switch {
			case trimmed == "end" || strings.HasPrefix(trimmed, "end "):
				popScope()

			case strings.HasPrefix(trimmed, "range") && hasDecl(trimmed):
				pushScope()
				for _, name := range declaredVars(trimmed) {
					addToCurrentScope(name)
				}

			case strings.HasPrefix(trimmed, "if") && hasDecl(trimmed):
				pushScope()
				for _, name := range declaredVars(trimmed) {
					addToCurrentScope(name)
				}

			case strings.HasPrefix(trimmed, "with") && hasDecl(trimmed):
				pushScope()
				for _, name := range declaredVars(trimmed) {
					addToCurrentScope(name)
				}

			case strings.Contains(trimmed, ":="):
				for _, name := range declaredVars(trimmed) {
					addToCurrentScope(name)
				}
			}
		}

		if !hasClose {
			break
		}
		i = end + 2
	}

	result := make([]string, 0, len(order))
	for _, name := range order {
		if visible[name] > 0 {
			result = append(result, name)
		}
	}
	return result
}

// hasDecl determines whether a template action string contains a variable declaration operator (:=)
func hasDecl(action string) bool {
	return strings.Contains(action, ":=")
}

// declaredVars parses the left-hand side of a variable declaration and returns the names of the variables being defined.
func declaredVars(action string) []string {
	left, _, found := strings.Cut(action, ":=")
	if !found {
		return nil
	}

	matches := variablePattern.FindAllString(left, -1)
	if len(matches) == 0 {
		return nil
	}

	out := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// isInsideTemplate determines if a given byte offset resides within the delimiters of a template action and is not a comment.
func isInsideTemplate(text string, offset int) bool {
	if offset > len(text) {
		offset = len(text)
	}
	if offset < 0 {
		offset = 0
	}

	sub := text[:offset]

	lastOpen := strings.LastIndex(sub, "{{")
	if lastOpen == -1 {
		return false
	}

	lastClose := strings.LastIndex(sub, "}}")
	if lastClose > lastOpen {
		return false
	}

	if strings.HasPrefix(sub[lastOpen:], "{{/*") {
		return false
	}

	return true
}

// getWordAtOffset returns the sequence of valid identifier characters immediately preceding the given byte offset.
func getWordAtOffset(text string, offset int) string {
	if offset > len(text) {
		offset = len(text)
	}
	if offset < 0 {
		offset = 0
	}

	start := offset
	for start > 0 && isWordChar(rune(text[start-1])) {
		start--
	}
	return text[start:offset]
}

// positionToOffset translates an LSP line and character position into a flat byte offset, accounting for multibyte UTF-8 characters.
func positionToOffset(text string, pos protocol.Position) int {
	lines := strings.Split(text, "\n")
	line := int(pos.Line)
	if line < 0 {
		return 0
	}
	if line >= len(lines) {
		return len(text)
	}

	offset := 0
	for i := 0; i < line; i++ {
		offset += len(lines[i]) + 1
	}

	lineText := lines[line]
	char := int(pos.Character)
	if char < 0 {
		char = 0
	}

	byteOffset := 0
	for i := 0; i < len(lineText) && char > 0; {
		_, size := utf8.DecodeRuneInString(lineText[i:])
		i += size
		byteOffset += size
		char--
	}

	return offset + byteOffset
}

// isWordChar reports whether a rune is a valid character for a template variable or function name.
func isWordChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '$'
}
