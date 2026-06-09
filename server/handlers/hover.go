// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"errors"
	"math"
	"regexp"
	"strings"
	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Hover handles providing the hover message
func Hover(_ *glsp.Context, params *protocol.HoverParams) (hover *protocol.Hover, err error) {
	// Get document content
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || doc.tree == nil {
		err = errors.New("document not found or failed to parse")
		log.Debug().Err(err)
		return
	}

	offset := positionToOffset(doc.text, params.Position)
	tree := doc.treeAt(parse.Pos(offset))
	if tree == nil || tree.Root == nil {
		err = errors.New("no parse tree at offset")
		log.Debug().Err(err)
		return
	}
	target := nodeFind(tree.Root, parse.Pos(offset))
	if target == nil {
		err = errors.New("node not found")
		log.Debug().Err(err)
		return
	}
	// Check for end tag hover
	if branchNode, endRange := endTagHover(
		target,
		params.Position,
		doc.text,
		tree.Root,
	); branchNode != nil {
		// log.Debug().Msg("Hover on end tag of BranchNode")
		hover = &protocol.Hover{
			Range: &endRange,
		}
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageEnd(branchNode, offsetToPosition(doc.text, int(branchNode.Position()))),
		}
		return
	}
	// Check for else tag hover
	if branchNode, elseRange := elseNodeHover(
		target,
		params.Position,
		doc.text,
		tree.Root,
	); branchNode != nil {
		log.Debug().Msg("Hover on else tag of BranchNode")
		hover = &protocol.Hover{
			Range: &elseRange,
		}
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageElse(&branchNode, offsetToPosition(doc.text, int(branchNode.Position()))),
		}
		return
	}

	nodeRange := nodeToRange(target, doc.text)

	hover = &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "",
		},
		Range: &nodeRange,
	}

	// Build hover content based on node type

	buildHoverContent(target, hover, doc)

	return
}

func buildHoverContent(target parse.Node, hover *protocol.Hover, doc *document) {
	switch target := target.(type) {
	case *parse.BranchNode:
		{
			log.Debug().Msg("Hover on BranchNode")
			hover.Contents = protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageBranch(target),
			}
		}
	case *parse.DotNode:
		log.Debug().Msg("Hover on DotNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageDot(target),
		}
	case *parse.FieldNode:
		log.Debug().Msg("Hover on FieldNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageField(target),
		}
	case *parse.IdentifierNode:
		log.Debug().Msg("Hover on IdentifierNode")
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageIdentifier(target),
		}
	case *parse.VariableNode:
		log.Debug().Any("node", target).Msg("Hover on VariableNode")

		tree := doc.treeAt(target.Position())
		if tree == nil {
			tree = doc.tree
		}
		if IsIndexVariable(target, tree.Root) {
			hover.Contents = protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageIndexVariable(target),
			}
		} else {
			varValue, goType := ResolveVarInfo(
				tree.Root,
				target,
				doc.loadedTypeAt(target.Position()),
			)

			hover.Contents = protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: MessageVariable(target, varValue, goType),
			}

		}
	case *parse.NilNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageNil(target),
		}
	case *parse.IfNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageBranch(&target.BranchNode),
		}
	case *parse.RangeNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageBranch(&target.BranchNode),
		}
	case *parse.WithNode:
		log.Debug().Msgf("Hover on %T", target)
		hover.Contents = protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: MessageBranch(&target.BranchNode),
		}
	default:
		log.Debug().Msgf("Hover on unknown node type: %T", target)
	}
}

// endTagHover checks if the hover position is within an end tag and returns the corresponding branch node and range if so.
func endTagHover(
	target parse.Node,
	pos protocol.Position,
	text string,
	root parse.Node,
) (resnode parse.Node, endRange protocol.Range) {
	lines := strings.Split(text, "\n")

	targetPos := offsetToPosition(text, int(target.Position()))

	if int(pos.Line) >= len(lines) {
		return
	}

	line := lines[int(pos.Line)]
	if int(pos.Character) > len(line) {
		return
	}
	reg := regexp.MustCompile(`{{-?\s*end\s*-?}}`)
	matches := reg.FindAllStringIndex(line, -1)
	for _, match := range matches {
		if int(pos.Character) >= match[0] && int(pos.Character) <= match[1] {
			// log.Debug().Msg("Position is within an end tag")
			// Scan backwards until target node is found to count end tags and find the corresponding branch node
			cline := int(pos.Line)
			character := int(pos.Character)
			count := 0
			for cline > int(targetPos.Line) || (cline == int(targetPos.Line) && character > int(targetPos.Character)) {
				line := lines[cline]
				matches := reg.FindAllStringIndex(line, -1)
				for j := len(matches) - 1; j >= 0; j-- {
					match := matches[j]
					if cline == int(pos.Line) && (character >= match[0]) {
						continue
					}
					if cline == int(targetPos.Line) && (match[1] >= int(targetPos.Character)) {
						continue
					}
					count++
				}
				cline--

			}
			// Find the corresponding branch node for this end tag
			ctx := &Context{Vars: make(map[string]parse.Node)}
			buildPath(root, target, ctx)
			path := ctx.Path
			for i := len(path) - 1; i >= 0; i-- {
				switch path[i].(type) {
				case *parse.RangeNode, *parse.IfNode, *parse.WithNode, *parse.TemplateNode:
					if count > 0 {
						count--
						continue
					}
					// log.Debug().Msgf("Found branch node of type %T for end tag hover", node)
					resnode = path[i]
					if match[1] < 0 || match[1] > math.MaxUint32 {
						panic("line length overflows uint32??")
					}
					endRange = protocol.Range{
						Start: protocol.Position{
							Line:      pos.Line,
							Character: uint32(match[0]), //nolint:gosec
						},
						End: protocol.Position{
							Line:      pos.Line,
							Character: uint32(match[1]), //nolint:gosec
						},
					}
					return
				}
			}

		}
	}

	return
}

// elseNodeHover checks if the hover position is within an else tag and returns the corresponding branch node and range if so.
func elseNodeHover(
	target parse.Node,
	pos protocol.Position,
	text string,
	root parse.Node,
) (resnode parse.Node, elseRange protocol.Range) {
	lines := strings.Split(text, "\n")

	targetPos := offsetToPosition(text, int(target.Position()))

	if int(pos.Line) >= len(lines) {
		return
	}

	line := lines[int(pos.Line)]
	if int(pos.Character) > len(line) {
		return
	}
	reg := regexp.MustCompile(`{{-?\s*else\b`)
	endReg := regexp.MustCompile(`{{-?\s*end\s*-?}}`)
	matches := reg.FindAllStringIndex(line, -1)
	for _, match := range matches {
		if int(pos.Character) >= match[0] && int(pos.Character) <= match[1] {
			log.Debug().Msg("Position is within an else tag")
			// Scan backwards until target node is found to count else tags and find the corresponding branch node
			cline := int(pos.Line)
			character := int(pos.Character)
			count := 0
			for cline > int(targetPos.Line) || (cline == int(targetPos.Line) && character > int(targetPos.Character)) {
				line := lines[cline]
				matches := endReg.FindAllStringIndex(line, -1)
				for j := len(matches) - 1; j >= 0; j-- {
					match := matches[j]
					if cline == int(pos.Line) && (character >= match[0]) {
						continue
					}
					if cline == int(targetPos.Line) && (match[1] >= int(targetPos.Character)) {
						continue
					}
					count++
				}
				cline--

			}
			// Find the corresponding branch node for this else tag
			ctx := &Context{Vars: make(map[string]parse.Node)}
			buildPath(root, target, ctx)
			path := ctx.Path
			for i := len(path) - 1; i >= 0; i-- {
				switch path[i].(type) {
				case *parse.RangeNode, *parse.IfNode, *parse.WithNode:
					if count > 0 {
						count--
						continue
					}
					// log.Debug().Msgf("Found branch node of type %T for else tag hover", node)
					resnode = path[i]
					if match[1] < 0 || match[1] > math.MaxUint32 {
						panic("line length overflows uint32??")
					}
					elseRange = protocol.Range{
						Start: protocol.Position{
							Line:      pos.Line,
							Character: uint32(match[0]), //nolint:gosec
						},
						End: protocol.Position{
							Line:      pos.Line,
							Character: uint32(match[1]), //nolint:gosec
						},
					}
					return
				}
			}

		}
	}

	return
}
