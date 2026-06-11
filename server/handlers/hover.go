// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"errors"
	gotypes "go/types"
	"math"
	"regexp"
	"strings"
	parse "text-template-parser"
	serverTypes "text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mdContent wraps a markdown string in a protocol.MarkupContent.
func mdContent(s string) protocol.MarkupContent {
	return protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: s}
}

// lookupTypedNodeType returns the resolved Go type of the analysed node at
// the same source position as target, or nil if none is known.
func lookupTypedNodeType(doc *document, target parse.Node) gotypes.Type {
	if doc == nil || target == nil {
		return nil
	}
	tt := doc.typedTreeAt(target.Position())
	if tt == nil || tt.Root == nil {
		return nil
	}
	if n := serverTypes.NodeFind(tt.Root, serverTypes.Pos(target.Position())); n != nil {
		return n.ValueType()
	}
	return nil
}

// Hover handles providing the hover message.
func Hover(_ *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || doc.tree == nil {
		return nil, errors.New("document not found or failed to parse")
	}

	offset := positionToOffset(doc.text, params.Position)
	tree := doc.treeAt(parse.Pos(offset))
	if tree == nil || tree.Root == nil {
		return nil, errors.New("no parse tree at offset")
	}
	target := nodeFind(tree.Root, parse.Pos(offset))
	if target == nil {
		return nil, errors.New("node not found")
	}

	// {{end}} and {{else}} tags don't get their own AST nodes, so handle them
	// at the text level first and bail out with a tag-shaped hover range.
	if bn, r := endTagHover(target, params.Position, doc.text, tree.Root); bn != nil {
		return &protocol.Hover{
			Range:    &r,
			Contents: mdContent(MessageEnd(bn, offsetToPosition(doc.text, int(bn.Position())))),
		}, nil
	}
	if bn, r := elseNodeHover(target, params.Position, doc.text, tree.Root); bn != nil {
		return &protocol.Hover{
			Range:    &r,
			Contents: mdContent(MessageElse(&bn, offsetToPosition(doc.text, int(bn.Position())))),
		}, nil
	}

	r := nodeToRange(target, doc.text)
	return &protocol.Hover{
		Range:    &r,
		Contents: mdContent(hoverMessage(target, doc)),
	}, nil
}

// hoverMessage returns the markdown body for a hover on target.
func hoverMessage(target parse.Node, doc *document) string {
	log.Debug().Msgf("Hover on %T", target)
	switch t := target.(type) {
	case *parse.IfNode:
		return MessageBranch(&t.BranchNode)
	case *parse.RangeNode:
		return MessageBranch(&t.BranchNode)
	case *parse.WithNode:
		return MessageBranch(&t.BranchNode)
	case *parse.DotNode:
		return MessageDot(t, lookupTypedNodeType(doc, t))
	case *parse.FieldNode:
		return MessageField(t, lookupTypedNodeType(doc, t))
	case *parse.IdentifierNode:
		return MessageIdentifier(t)
	case *parse.NilNode:
		return MessageNil(t)
	case *parse.VariableNode:
		tree := doc.treeAt(t.Position())
		if tree == nil {
			tree = doc.tree
		}
		if IsIndexVariable(t, tree.Root) {
			return MessageIndexVariable(t)
		}
		val, typ := ResolveVarInfo(tree.Root, t, doc.typedTreeAt(t.Position()))
		return MessageVariable(t, val, typ)
	}
	return ""
}

var (
	endTagRe  = regexp.MustCompile(`{{-?\s*end\s*-?}}`)
	elseTagRe = regexp.MustCompile(`{{-?\s*else\b`)
)

// tagHover finds a synthetic tag (end or else) at pos and walks the ancestor
// path of target to locate the branch node it belongs to. nestRe counts the
// nesting levels above the cursor; allowTemplate decides whether a
// TemplateNode in the ancestor chain consumes a nesting level.
func tagHover(
	target parse.Node,
	pos protocol.Position,
	text string,
	root parse.Node,
	tagRe, nestRe *regexp.Regexp,
	allowTemplate bool,
) (parse.Node, protocol.Range) {
	var zero protocol.Range
	lines := strings.Split(text, "\n")
	if int(pos.Line) >= len(lines) {
		return nil, zero
	}
	line := lines[int(pos.Line)]
	if int(pos.Character) > len(line) {
		return nil, zero
	}
	targetPos := offsetToPosition(text, int(target.Position()))

	for _, match := range tagRe.FindAllStringIndex(line, -1) {
		if int(pos.Character) < match[0] || int(pos.Character) > match[1] {
			continue
		}
		// Count nesting tags between the cursor and the target node.
		count := 0
		for cline := int(pos.Line); cline > int(targetPos.Line) ||
			(cline == int(targetPos.Line) && int(pos.Character) > int(targetPos.Character)); cline-- {
			for _, m := range nestRe.FindAllStringIndex(lines[cline], -1) {
				if cline == int(pos.Line) && int(pos.Character) >= m[0] {
					continue
				}
				if cline == int(targetPos.Line) && m[1] >= int(targetPos.Character) {
					continue
				}
				count++
			}
		}
		// Walk the ancestor path, skipping `count` branches before claiming one.
		ctx := &Context{Vars: make(map[string]parse.Node)}
		buildPath(root, target, ctx)
		for i := len(ctx.Path) - 1; i >= 0; i-- {
			if !isBranchAncestor(ctx.Path[i], allowTemplate) {
				continue
			}
			if count > 0 {
				count--
				continue
			}
			if match[1] < 0 || match[1] > math.MaxUint32 {
				panic("line length overflows uint32??")
			}
			return ctx.Path[i], protocol.Range{
				Start: protocol.Position{
					Line:      pos.Line,
					Character: uint32(match[0]),
				}, //nolint:gosec
				End: protocol.Position{
					Line:      pos.Line,
					Character: uint32(match[1]),
				}, //nolint:gosec
			}
		}
	}
	return nil, zero
}

func isBranchAncestor(n parse.Node, allowTemplate bool) bool {
	switch n.(type) {
	case *parse.RangeNode, *parse.IfNode, *parse.WithNode:
		return true
	case *parse.TemplateNode:
		return allowTemplate
	}
	return false
}

func endTagHover(
	target parse.Node,
	pos protocol.Position,
	text string,
	root parse.Node,
) (parse.Node, protocol.Range) {
	return tagHover(target, pos, text, root, endTagRe, endTagRe, true)
}

func elseNodeHover(
	target parse.Node,
	pos protocol.Position,
	text string,
	root parse.Node,
) (parse.Node, protocol.Range) {
	return tagHover(target, pos, text, root, elseTagRe, endTagRe, false)
}
