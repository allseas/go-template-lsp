package handlers

import (
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

type rawToken struct {
	startByte int
	length    int
	tokenType uint32
	modifiers uint32
}

func SemanticTokensFull(
	_ *glsp.Context,
	params *protocol.SemanticTokensParams,
) (*protocol.SemanticTokens, error) {
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || doc.typedTree == nil || doc.typedTree.Root == nil {
		return nil, nil
	}

	serverTypes.SetEndsForTree(*doc.typedTree, serverTypes.Pos(len(doc.text)), &doc.text)

	var tokens []rawToken
	walkSemanticNode(doc.typedTree.Root, doc.text, &tokens)

	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].startByte < tokens[j].startByte
	})

	return &protocol.SemanticTokens{
		Data: encodeSemanticTokens(tokens, doc.text),
	}, nil
}

func DocumentSymbols(_ *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	return nil, nil
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

	case *serverTypes.CommandNode:
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

	case *serverTypes.VariableNode:
		emitToken(tokens, int(n.Position()), len(n.String()), ttVariable, 0)

	case *serverTypes.IdentifierNode:
		// reached outside CommandNode first-arg - emit as function
		emitToken(tokens, int(n.Position()), len(n.Ident), ttFunction, 0)

	case *serverTypes.FieldNode:
		l := 0
		for _, id := range n.Ident {
			l += 1 + len(id) // "." + ident
		}
		emitToken(tokens, int(n.Position()), l, ttProperty, 0)

	case *serverTypes.ChainNode:
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
		emitEndKeyword(tokens, text, n.List, n.ElseList)

	case *serverTypes.RangeNode:
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

	case *serverTypes.WithNode:
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

	case *serverTypes.BreakNode:
		emitKeywordToken(tokens, text, int(n.Position()), int(n.End()), "break")

	case *serverTypes.ContinueNode:
		emitKeywordToken(tokens, text, int(n.Position()), int(n.End()), "continue")

	case *serverTypes.TemplateNode:
		emitKeywordToken(tokens, text, int(n.Position()), int(n.End()), "template")
		if n.Pipe != nil {
			walkSemanticNode(n.Pipe, text, tokens)
		}

	case *serverTypes.UndefinedNode:
		// skip unparseable fragments
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
	searchTo := min(searchFrom+30, len(text))
	idx := strings.Index(text[searchFrom:searchTo], "end")
	if idx < 0 {
		return
	}
	emitToken(tokens, searchFrom+idx, 3, ttKeyword, 0)
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

		length := uint32(utf16Len(text[tok.startByte:end]))

		data = append(data, deltaLine, deltaChar, length, tok.tokenType, tok.modifiers)

		prevLine = line
		prevChar = char
	}

	return data
}
