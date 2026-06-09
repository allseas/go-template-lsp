// Package handlers provides a Language Server Protocol implementation for Go text/templates, featuring scope-aware variable completion and built-in function support.
package handlers

import (
	"regexp"
	"strings"
	"text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	variablePattern = regexp.MustCompile(`\$[a-zA-Z_][a-zA-Z0-9_]*`)

	templateKeywords = []string{
		"range", "if", "with", "else", "end", "template", "block", "define",
	}
)

// globalFunctionNames returns the names of all global functions
func globalFunctionNames() []string {
	hinted := types.GlobalFuncs()
	out := make([]string, 0, len(hinted))
	for name := range hinted {
		out = append(out, name)
	}
	return out
}

// completion handles LSP "textDocument/completion" requests by identifying
// the current template context and returning relevant globalFunctions and variable names.
func completion(_ *glsp.Context, params *protocol.CompletionParams) (any, error) {
	doc, ok := store.Get(params.TextDocument.URI)
	if !GetConfig().EnableServer {
		log.Debug().Msg("completion requested but server is disabled by config")
		return nil, nil
	}

	if !ok {
		log.Error().Str("uri", params.TextDocument.URI).Msg("document not found in store")
		return nil, nil
	}

	// later also use tree
	text := doc.text

	offset := positionToOffset(text, params.Position)
	if !isInsideTemplate(text, offset) {
		log.Debug().
			Int("offset new ", offset).
			Msg("completion: cursor is not inside a template block, skipping")
		return nil, nil
	}

	currentWord := getWordAtOffset(text, offset)
	wordUTF16Len := utf16Len(currentWord)
	startChar := int(params.Position.Character) - wordUTF16Len
	if startChar < 0 {
		log.Warn().
			Int("char_position", int(params.Position.Character)).
			Msg("completion: calculated negative start character; clamping to 0")
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

	globalFns := globalFunctionNames()

	items := make([]protocol.CompletionItem, 0, len(vars)+len(globalFns))
	seen := make(map[string]bool)

	varKind := protocol.CompletionItemKindVariable
	fnKind := protocol.CompletionItemKindFunction

	for _, v := range vars {
		if seen[v] {
			continue
		}
		seen[v] = true

		filter := strings.TrimPrefix(v, "$")

		items = append(items, protocol.CompletionItem{
			Label:      v,
			Kind:       &varKind,
			FilterText: &filter, // <-- key line
			TextEdit:   &protocol.TextEdit{Range: wordRange, NewText: v},
		})
	}

	for _, fn := range globalFns {
		fnLabel := fn
		items = append(items, protocol.CompletionItem{
			Label:      fn,
			Kind:       &fnKind,
			FilterText: &fnLabel,
			TextEdit:   &protocol.TextEdit{Range: wordRange, NewText: fn},
		})
	}

	keywordKind := protocol.CompletionItemKindKeyword

	for _, kw := range templateKeywords {
		items = append(items, protocol.CompletionItem{
			Label:    kw,
			Kind:     &keywordKind,
			TextEdit: &protocol.TextEdit{Range: wordRange, NewText: kw},
		})
	}

	return protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// extractVariables scans the template text up to the cursor to build a scope-aware list of defined variables.
func extractVariables(text string, cursor int) []string {
	if cursor > len(text) || cursor < 0 {
		log.Warn().Int("cursor", cursor).Msg("extractVariables: cursor out of bounds; clamping")
		if cursor > len(text) {
			cursor = len(text)
		} else {
			cursor = 0
		}
	}

	s := newScopeTracker()

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
		var action string
		var next int
		if closeRel == -1 {
			action = strings.TrimSpace(text[open+2 : cursor])
			next = cursor
		} else {
			closePos := open + 2 + closeRel
			action = strings.TrimSpace(text[open+2 : closePos])
			next = closePos + 2
		}

		if action != "" && !strings.HasPrefix(action, "/*") {
			switch {
			case action == "end" || strings.HasPrefix(action, "end "):
				s.pop()

			case (strings.HasPrefix(action, "range") ||
				strings.HasPrefix(action, "if") ||
				strings.HasPrefix(action, "with")) &&
				hasDecl(action): // Using the helper here
				s.push()
				for _, name := range declaredVars(action) { // Using the helper here
					s.declare(name)
				}

			case hasDecl(action): // Using the helper for simple assignments too
				for _, name := range declaredVars(action) {
					s.declare(name)
				}
			}
		}

		i = next
	}

	return s.visibleVars()
}

// scopeTracker maintains the variable scope stack and visibility counts.
// - scopeStack: a stack of scopes, each scope is a set of variable names declared in it
// - visible: counts how many times a variable is in scope
// - order: insertion order so results are stable and predictable
type scopeTracker struct {
	// stack of scopes - each scope holds variables declared at that level
	// e.g. global scope at index 0, range block at index 1, etc.
	stack []map[string]struct{}
	// order tracks declaration order for stable results
	order []string
}

func newScopeTracker() *scopeTracker {
	return &scopeTracker{
		stack: []map[string]struct{}{{}},
		order: []string{},
	}
}

func (s *scopeTracker) push() {
	s.stack = append(s.stack, map[string]struct{}{})
}

func (s *scopeTracker) pop() {
	if len(s.stack) > 1 {
		s.stack = s.stack[:len(s.stack)-1]
	}
}

func (s *scopeTracker) declare(name string) {
	if name == "" {
		return
	}
	// add to current (innermost) scope
	top := s.stack[len(s.stack)-1]
	if _, exists := top[name]; !exists {
		top[name] = struct{}{}
		s.order = append(s.order, name)
	}
}

// visibleVars returns all variables present in any scope on the stack
func (s *scopeTracker) visibleVars() []string {
	// build a set of all visible names from all scopes
	visible := map[string]struct{}{}
	for _, scope := range s.stack {
		for name := range scope {
			visible[name] = struct{}{}
		}
	}

	// return in declaration order
	var result []string
	for _, name := range s.order {
		if _, ok := visible[name]; ok {
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
	if offset > len(text) || offset < 0 {
		log.Warn().
			Int("offset", offset).
			Int("text_len", len(text)).
			Msg("isInsideTemplate: offset out of bounds, clamping value")

		if offset > len(text) {
			offset = len(text)
		} else {
			offset = 0
		}
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
	if offset > len(text) || offset < 0 {
		log.Warn().
			Int("offset", offset).
			Msg("getWordAtOffset: offset out of bounds; clamping")

		if offset > len(text) {
			offset = len(text)
		} else {
			offset = 0
		}
	}

	start := offset
	for start > 0 && isWordChar(rune(text[start-1])) {
		start--
	}
	return text[start:offset]
}

// positionToOffset translates an LSP line and character position into a flat byte offset, accounting for multibyte UTF-8 characters.
func positionToOffset(text string, pos protocol.Position) int {
	line := uint32(0)
	charUTF16 := uint32(0)

	for byteOffset, r := range text {
		if line == pos.Line && charUTF16 >= pos.Character {
			return byteOffset
		}

		if r == '\n' {
			line++
			charUTF16 = 0
			continue
		}

		if line == pos.Line {
			if r > 0xFFFF {
				charUTF16 += 2
			} else {
				charUTF16++
			}
		}
	}

	log.Debug().
		Int("line", int(line)).
		Int("chars", int(charUTF16)).
		Msg("character emitted by pos")

	return len(text)
}

// isWordChar reports whether a rune is a valid character for a template variable or function name.
func isWordChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '$'
}

func utf16Len(s string) int {
	count := 0
	for _, r := range s {
		if r > 0xFFFF {
			count += 2
		} else {
			count++
		}
	}
	return count
}
