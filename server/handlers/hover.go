// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"errors"
	gotypes "go/types"
	"math"
	"regexp"
	"strings"
	"text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mdContent wraps a markdown string in a protocol.MarkupContent.
func mdContent(s string) protocol.MarkupContent {
	return protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: s}
}

// Hover handles providing the hover message.
func Hover(_ *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	if !GetConfig().EnableHover {
		log.Debug().Msg("hover requested but hover is disabled by config")
		return nil, nil
	}
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || doc.typedTree == nil {
		return nil, errors.New("document not found or failed to parse")
	}

	offset := positionToOffset(doc.text, params.Position)
	tree := doc.typedTreeAtTyped(types.Pos(offset))
	if tree == nil || tree.Root == nil {
		return nil, errors.New("no parse tree at offset")
	}
	target := types.NodeFind(tree.Root, types.Pos(offset))
	if target == nil {
		return nil, errors.New("node not found")
	}

	// {{end}} and {{else}} tags don't get their own AST nodes, so handle them
	// at the text level first and bail out with a tag-shaped hover range.
	if bn, r := endTagHover(target, params.Position, doc.text); bn != nil {
		return &protocol.Hover{
			Range:    &r,
			Contents: mdContent(MessageEnd(bn, offsetToPosition(doc.text, int(bn.Position())))),
		}, nil
	}
	if bn, r := elseNodeHover(target, params.Position, doc.text); bn != nil {
		return &protocol.Hover{
			Range:    &r,
			Contents: mdContent(MessageElse(&bn, offsetToPosition(doc.text, int(bn.Position())))),
		}, nil
	}

	r := nodeRange(target, doc.text)
	return &protocol.Hover{
		Range:    &r,
		Contents: mdContent(hoverMessage(target, doc)),
	}, nil
}

// hoverMessage returns the markdown body for a hover on target.
func hoverMessage(target types.Node, _ *document) string {
	log.Debug().Msgf("Hover on %T", target)
	switch t := target.(type) {
	case *types.IfNode:
		return MessageBranch(&t.BranchNode)
	case *types.RangeNode:
		return MessageBranch(&t.BranchNode)
	case *types.WithNode:
		return MessageBranch(&t.BranchNode)
	case *types.DotNode:
		return MessageDot(t, t.ValueType())
	case *types.FieldNode:
		return MessageField(t, t.ValueType())
	case *types.IdentifierNode:
		return MessageIdentifier(t, t.ValueType())
	case *types.NilNode:
		return MessageNil(t)
	case *types.VariableNode:
		if IsIndexVariableTyped(t) {
			return MessageIndexVariable(t)
		}
		var goType gotypes.Type
		if t != nil {
			goType = t.ValueType()
		}
		return MessageVariable(t, nil, goType)
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
	target types.Node,
	pos protocol.Position,
	text string,
	tagRe, nestRe *regexp.Regexp,
	allowTemplate bool,
) (types.Node, protocol.Range) {
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
		// Walk the parent chain (innermost to outermost), skipping `count`
		// branches before claiming one.
		for cur := target; cur != nil; cur = cur.Parent() {
			if !isBranchAncestor(cur, allowTemplate) {
				continue
			}
			if count > 0 {
				count--
				continue
			}
			if match[1] < 0 || match[1] > math.MaxUint32 {
				panic("line length overflows uint32??")
			}
			return cur, protocol.Range{
				Start: protocol.Position{
					Line:      pos.Line,
					Character: uint32(match[0]), //nolint:gosec // bounded above
				},
				End: protocol.Position{
					Line:      pos.Line,
					Character: uint32(match[1]), //nolint:gosec // bounded above
				},
			}
		}
	}
	return nil, zero
}

func isBranchAncestor(n types.Node, allowTemplate bool) bool {
	switch n.(type) {
	case *types.RangeNode, *types.IfNode, *types.WithNode:
		return true
	case *types.TemplateNode:
		return allowTemplate
	}
	return false
}

func endTagHover(
	target types.Node,
	pos protocol.Position,
	text string,
) (types.Node, protocol.Range) {
	return tagHover(target, pos, text, endTagRe, endTagRe, true)
}

func elseNodeHover(
	target types.Node,
	pos protocol.Position,
	text string,
) (types.Node, protocol.Range) {
	return tagHover(target, pos, text, elseTagRe, endTagRe, false)
}
