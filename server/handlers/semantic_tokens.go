package handlers

import (
	"regexp"
	"sort"
	"strings"

	serverTypes "text-template-server/types"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Token type indices.
const (
	ttKeyword  = uint32(0)
	ttFunction = uint32(1)
	ttVariable = uint32(2)
	ttProperty = uint32(3)
	ttString   = uint32(4)
	ttNumber   = uint32(5)
	ttComment  = uint32(6)
)

// TokenTypes is the legend of semantic token types advertised to LSP clients.
var TokenTypes = []string{
	"keyword",
	"function",
	"variable",
	"property",
	"string",
	"number",
	"comment",
}

// Token modifier bit flags.
const (
	tmDeclaration    = uint32(1 << 0)
	tmDefaultLibrary = uint32(1 << 1)
)

// TokenModifiers is the legend of semantic token modifiers advertised to LSP clients.
var TokenModifiers = []string{"declaration", "defaultLibrary"}

var builtinFuncs = map[string]bool{
	"and": true, "call": true, "html": true, "index": true, "slice": true,
	"js": true, "len": true, "not": true, "or": true, "print": true,
	"printf": true, "println": true, "urlquery": true,
	"eq": true, "ne": true, "lt": true, "le": true, "gt": true, "ge": true,
}

// reEnd matches a {{end}} delimiter, with optional whitespace-trimming dashes.
var reEnd = regexp.MustCompile(`\{\{-?\s*end\s*-?\}\}`)

type rawToken struct {
	startByte int
	length    int
	tokenType uint32
	modifiers uint32
}

// SemanticTokensFull implements the "textDocument/semanticTokens/full" LSP method.
// It traverses the typed syntax tree of the document and emits semantic tokens for recognized nodes.
func SemanticTokensFull(
	_ *glsp.Context,
	params *protocol.SemanticTokensParams,
) (*protocol.SemanticTokens, error) {
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	var tokens []rawToken
	for _, tt := range doc.typedTrees {
		if tt != nil && tt.Root != nil {
			walkSemanticNode(tt.Root, doc.text, &tokens)
		}
	}

	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].startByte < tokens[j].startByte
	})

	return &protocol.SemanticTokens{
		Data: encodeSemanticTokens(tokens, doc.text),
	}, nil
}

// reDefine matches the opening {{define "name"}} delimiter (with optional trim dashes).
var reDefine = regexp.MustCompile(`\{\{-?\s*define\s+`)

// DocumentSymbols implements the "textDocument/documentSymbol" LSP method.
// Returns one symbol per {{define "name"}} block found in the document.
func DocumentSymbols(
	_ *glsp.Context,
	params *protocol.DocumentSymbolParams,
) (any, error) {
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || len(doc.typedTrees) == 0 {
		return nil, nil
	}

	var symbols []protocol.DocumentSymbol
	for name, t := range doc.typedTrees {
		if t == nil || t.Root == nil || name == "t" {
			continue
		}

		bodyStart := int(t.Root.Position())
		blockEnd := int(t.End)

		// Search backward from the body start to find the nearest {{define.
		blockStart := bodyStart
		if ms := reDefine.FindAllStringIndex(doc.text[:bodyStart], -1); ms != nil {
			blockStart = ms[len(ms)-1][0]
		}

		// Verify this tree was introduced by a {{define "name"}} (not {{block}}).
		quotedName := `"` + name + `"`
		headerFull := doc.text[blockStart:bodyStart]
		rdelimIdx := strings.Index(headerFull, "}}")
		if rdelimIdx < 0 || !strings.Contains(headerFull[:rdelimIdx+2], quotedName) {
			continue
		}
		defineHeader := headerFull[:rdelimIdx+2]

		nameStart := bodyStart
		if idx := strings.Index(defineHeader, quotedName); idx >= 0 {
			nameStart = blockStart + idx + 1 // +1 skips the opening quote
		}
		nameEnd := nameStart + len(name)

		fullRange := bytesToRange(doc.text, blockStart, blockEnd)
		selRange := bytesToRange(doc.text, nameStart, nameEnd)
		kind := protocol.SymbolKindFunction

		if name == "" {
			// empty define name - diagnostic emitted by collectEmptyDefineNameDiagnostics
			continue
		}

		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           name,
			Kind:           kind,
			Range:          fullRange,
			SelectionRange: selRange,
		})
	}

	sort.Slice(symbols, func(i, j int) bool {
		ri, rj := symbols[i].Range, symbols[j].Range
		if ri.Start.Line != rj.Start.Line {
			return ri.Start.Line < rj.Start.Line
		}
		return ri.Start.Character < rj.Start.Character
	})

	return symbols, nil
}

// bytesToRange converts a [start, end) byte range in text to an LSP Range.
func bytesToRange(text string, start, end int) protocol.Range {
	startPos := offsetToPosition(text, start)
	endPos := offsetToPosition(text, end)
	return protocol.Range{Start: startPos, End: endPos}
}

func walkSemanticNode(node serverTypes.Node, text string, tokens *[]rawToken) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *serverTypes.ListNode:
		for _, child := range n.Nodes {
			walkSemanticNode(child, text, tokens)
		}

	case *serverTypes.TextNode:
		// plain text outside delimiters - no token

	case *serverTypes.CommentNode:
		emitToken(tokens, int(n.Position()), int(n.End()-n.Position()), ttComment, 0)

	case *serverTypes.ActionNode:
		if n.Pipe != nil {
			walkSemanticNode(n.Pipe, text, tokens)
		}

	case *serverTypes.PipeNode:
		walkPipeNode(n, tokens, text)

	case *serverTypes.CommandNode:
		walkCommandNode(n, tokens, text)

	case *serverTypes.VariableNode:
		walkVariableNode(n, tokens)

	case *serverTypes.IdentifierNode:
		// reached outside CommandNode first-arg - emit as function
		emitToken(tokens, int(n.Position()), len(n.Ident), ttFunction, 0)

	case *serverTypes.FieldNode:
		walkFieldNode(n, tokens)

	case *serverTypes.ChainNode:
		walkChainNode(n, text, tokens)

	case *serverTypes.DotNode:
		emitToken(tokens, int(n.Position()), 1, ttVariable, 0)

	case *serverTypes.StringNode:
		emitToken(tokens, int(n.Position()), len(n.Quoted), ttString, 0)

	case *serverTypes.NumberNode:
		emitToken(tokens, int(n.Position()), len(n.Text), ttNumber, 0)

	case *serverTypes.BoolNode:
		emitToken(tokens, int(n.Position()), len(n.String()), ttKeyword, 0)

	case *serverTypes.NilNode:
		emitToken(tokens, int(n.Position()), 3, ttKeyword, 0)

	case *serverTypes.IfNode:
		walkIfNode(tokens, text, n)

	case *serverTypes.RangeNode:
		walkRangeNode(tokens, text, n)

	case *serverTypes.WithNode:
		walkWithNode(tokens, text, n)

	case *serverTypes.BreakNode:
		emitKeywordToken(tokens, text, int(n.Position()), int(n.End()), "break")

	case *serverTypes.ContinueNode:
		emitKeywordToken(tokens, text, int(n.Position()), int(n.End()), "continue")

	case *serverTypes.TemplateNode:
		walkTemplateNode(n, text, tokens)

	case *serverTypes.UndefinedNode:
		// skip unparseable fragments
	}
}

// walkTemplateNode emits tokens for a {{template}} action.
// TemplateNode.Position() points at the template name, not the keyword, so
// "template" is located by searching backwards in the source from the name.
// The template name itself is emitted as a ttString token.
func walkTemplateNode(n *serverTypes.TemplateNode, text string, tokens *[]rawToken) {
	namePos := int(n.Position())
	if namePos > 0 {
		if kwStart := strings.LastIndex(text[:namePos], "template"); kwStart >= 0 {
			emitToken(tokens, kwStart, len("template"), ttKeyword, 0)
		}
	}
	if namePos >= 0 && namePos < len(text) {
		emitToken(tokens, namePos, len(n.Name)+2, ttString, 0)
	}
	if n.Pipe != nil {
		walkSemanticNode(n.Pipe, text, tokens)
	}
}

// walkVariableNode emits semantic tokens for a VariableNode.
// The base variable (Ident[0], e.g. "$item") is emitted as ttVariable.
// Each chained segment (Ident[1:], e.g. ".IsExpensive") is emitted as
// ttFunction when it resolves to a method, or ttProperty for struct fields.
// When type information is not available the chained segments fall back to ttProperty.
func walkVariableNode(n *serverTypes.VariableNode, tokens *[]rawToken) {
	if len(n.Ident) == 0 {
		return
	}
	pos := int(n.Position())
	// Base variable: "$item"
	emitToken(tokens, pos, len(n.Ident[0]), ttVariable, 0)
	pos += len(n.Ident[0])
	// Chained fields/methods: ".IsExpensive", ".Addr", etc.
	for i := 1; i < len(n.Ident); i++ {
		segLen := 1 + len(n.Ident[i]) // "." + ident
		tt := ttProperty
		if n.IdentIsMethod(i) {
			tt = ttFunction
		}
		emitToken(tokens, pos, segLen, tt, 0)
		pos += segLen
	}
}

// walkFieldNode emits one semantic token per segment of a FieldNode chain.
// Segments that resolve to methods (e.g. ".DisplayName") are emitted as ttFunction;
// plain struct-field segments (e.g. ".Address") are emitted as ttProperty.
// When type information is not available every segment falls back to ttProperty.
func walkFieldNode(n *serverTypes.FieldNode, tokens *[]rawToken) {
	pos := int(n.Position())
	for i, id := range n.Ident {
		segLen := 1 + len(id) // "." + ident
		tt := ttProperty
		if n.IdentIsMethod(i) {
			tt = ttFunction
		}
		emitToken(tokens, pos, segLen, tt, 0)
		pos += segLen
	}
}

func walkWithNode(tokens *[]rawToken, text string, n *serverTypes.WithNode) {
	emitKeywordToken(tokens, text, int(n.Position()), int(n.End()), "with")
	if n.Pipe != nil {
		walkSemanticNode(n.Pipe, text, tokens)
	}
	if n.List != nil {
		walkSemanticNode(n.List, text, tokens)
	}
	if n.ElseList != nil {
		emitElseKeyword(tokens, text, n.List, n.ElseList)
		walkSemanticNode(n.ElseList, text, tokens)
	}
	emitEndKeyword(tokens, text, n.List, n.ElseList)
}

func walkRangeNode(tokens *[]rawToken, text string, n *serverTypes.RangeNode) {
	emitKeywordToken(tokens, text, int(n.Position()), int(n.End()), "range")
	if n.Pipe != nil {
		walkSemanticNode(n.Pipe, text, tokens)
	}
	if n.List != nil {
		walkSemanticNode(n.List, text, tokens)
	}
	if n.ElseList != nil {
		emitElseKeyword(tokens, text, n.List, n.ElseList)
		walkSemanticNode(n.ElseList, text, tokens)
	}
	emitEndKeyword(tokens, text, n.List, n.ElseList)
}

func walkIfNode(tokens *[]rawToken, text string, n *serverTypes.IfNode) {
	emitKeywordToken(tokens, text, int(n.Position()), int(n.End()), "if")
	if n.Pipe != nil {
		walkSemanticNode(n.Pipe, text, tokens)
	}
	if n.List != nil {
		walkSemanticNode(n.List, text, tokens)
	}
	if n.ElseList != nil {
		emitElseKeyword(tokens, text, n.List, n.ElseList)
		walkSemanticNode(n.ElseList, text, tokens)
	}
	// When the ElseList contains a single nested IfNode (i.e. {{else if ...}}),
	// the inner walkIfNode already emitted the shared {{end}} token.
	// Emitting it here too would produce a duplicate at the same position.
	if !isElseIfChain(n.ElseList) {
		emitEndKeyword(tokens, text, n.List, n.ElseList)
	}
}

// isElseIfChain reports whether elseList is the desugared form of {{else if}},
// i.e. a ListNode whose sole child is an IfNode.
func isElseIfChain(elseList *serverTypes.ListNode) bool {
	if elseList == nil || len(elseList.Nodes) != 1 {
		return false
	}
	_, ok := elseList.Nodes[0].(*serverTypes.IfNode)
	return ok
}

func walkChainNode(n *serverTypes.ChainNode, text string, tokens *[]rawToken) {
	walkSemanticNode(n.Node, text, tokens)
	if n.Node != nil && len(n.Field) > 0 {
		baseLen := len(n.Node.String())
		if _, ok := n.Node.(*serverTypes.PipeNode); ok {
			baseLen += 2 // account for wrapping parens
		}
		fieldsLen := 0
		for _, f := range n.Field {
			fieldsLen += 1 + len(f)
		}
		emitToken(tokens, int(n.Position())+baseLen, fieldsLen, ttProperty, 0)
	}
}

func walkCommandNode(n *serverTypes.CommandNode, tokens *[]rawToken, text string) {
	for i, arg := range n.Args {
		if i == 0 {
			if id, ok := arg.(*serverTypes.IdentifierNode); ok {
				mod := uint32(0)
				if builtinFuncs[id.Ident] {
					mod = tmDefaultLibrary
				}
				emitToken(tokens, int(id.Position()), len(id.Ident), ttFunction, mod)
				continue
			}
		}
		walkSemanticNode(arg, text, tokens)
	}
}

func walkPipeNode(n *serverTypes.PipeNode, tokens *[]rawToken, text string) {
	for _, decl := range n.Decl {
		if decl != nil {
			emitToken(
				tokens,
				int(decl.Position()),
				len(decl.String()),
				ttVariable,
				tmDeclaration,
			)
		}
	}
	for _, cmd := range n.Cmds {
		walkSemanticNode(cmd, text, tokens)
	}
}

// emitKeywordToken searches for keyword in text[from:to] and emits a keyword token.
func emitKeywordToken(tokens *[]rawToken, text string, from, to int, keyword string) {
	if from < 0 || to > len(text) || from >= to {
		return
	}
	idx := strings.Index(text[from:to], keyword)
	if idx < 0 {
		return
	}
	emitToken(tokens, from+idx, len(keyword), ttKeyword, 0)
}

// emitElseKeyword finds the "else" keyword between list body content and the else list.
func emitElseKeyword(tokens *[]rawToken, text string, list, elseList *serverTypes.ListNode) {
	if elseList == nil || list == nil {
		return
	}
	var searchFrom int
	if len(list.Nodes) > 0 {
		searchFrom = int(list.Nodes[len(list.Nodes)-1].End())
	} else {
		searchFrom = int(list.Position())
	}
	searchTo := int(elseList.Position())
	if searchFrom < 0 || searchTo > len(text) || searchFrom >= searchTo {
		return
	}
	idx := strings.Index(text[searchFrom:searchTo], "else")
	if idx < 0 {
		return
	}
	emitToken(tokens, searchFrom+idx, 4, ttKeyword, 0)
}

// emitEndKeyword finds the "end" keyword that terminates a branch block.
func emitEndKeyword(tokens *[]rawToken, text string, list, elseList *serverTypes.ListNode) {
	lastList := list
	if elseList != nil {
		lastList = elseList
	}
	if lastList == nil {
		return
	}
	var searchFrom int
	if len(lastList.Nodes) > 0 {
		searchFrom = int(lastList.Nodes[len(lastList.Nodes)-1].End())
	} else {
		searchFrom = int(lastList.Position())
	}
	if searchFrom < 0 || searchFrom >= len(text) {
		return
	}
	m := reEnd.FindStringIndex(text[searchFrom:])
	if m == nil {
		return
	}
	// Emit only the "end" identifier within the delimiter.
	delim := text[searchFrom+m[0] : searchFrom+m[1]]
	idx := strings.Index(delim, "end")
	if idx < 0 {
		return
	}
	emitToken(tokens, searchFrom+m[0]+idx, 3, ttKeyword, 0)
}

func emitToken(tokens *[]rawToken, startByte, length int, tokenType, modifiers uint32) {
	if length <= 0 || startByte < 0 {
		return
	}
	*tokens = append(*tokens, rawToken{startByte, length, tokenType, modifiers})
}

// encodeSemanticTokens converts raw tokens to the LSP relative-encoded format.
// Each token is 5 uint32s: [deltaLine, deltaChar, length, tokenType, tokenModifiers].
func encodeSemanticTokens(tokens []rawToken, text string) []uint32 {
	data := make([]uint32, 0, len(tokens)*5)
	prevLine := uint32(0)
	prevChar := uint32(0)

	for _, tok := range tokens {
		end := tok.startByte + tok.length
		if end > len(text) {
			end = len(text)
		}

		pos := offsetToPosition(text, tok.startByte)
		line := pos.Line
		char := pos.Character

		deltaLine := line - prevLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = char - prevChar
		} else {
			deltaChar = char
		}

		length := uint32(utf16Len(text[tok.startByte:end])) //nolint:gosec

		data = append(data, deltaLine, deltaChar, length, tok.tokenType, tok.modifiers)

		prevLine = line
		prevChar = char
	}

	return data
}
