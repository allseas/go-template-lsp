package handlers

import (
	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func definition(_ *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	uri := params.TextDocument.URI
	position := params.Position
	doc, ok := store.Get(uri)
	if !ok {
		log.Error().Str("handler", "definition").Str("uri", uri).Msg("document not found in store")
		return nil, nil
	}

	offset := positionToOffset(doc.text, position)
	node := nodeFind(doc.tree.Root, parse.Pos(offset))

	if node == nil {
		log.Debug().Str("handler", "definition").Any("position", position).Msg("no node found")
		return nil, nil
	}

	if node.Type() == parse.NodeVariable {
		target := node.(*parse.VariableNode)
		varName := target.Ident[0]

		ctx := &Context{Vars: make(map[string]parse.Node)}
		buildPath(doc.tree.Root, node, ctx)

		declPipe, ok := ctx.Vars[varName]
		if !ok {
			// target is the declaration, return itself so the IDE shows references
			return protocol.Location{
				URI:   uri,
				Range: nodeToRange(target, doc.text),
			}, nil
		}

		pipe := declPipe.(*parse.PipeNode)
		for _, decl := range pipe.Decl {
			if decl.Ident[0] == varName {
				return protocol.Location{
					URI:   uri,
					Range: nodeToRange(decl, doc.text),
				}, nil
			}
		}

		return nil, nil
	}

	log.Debug().
		Str("handler", "definition").
		Any("node", node).
		Msg("node at position is not a field or identifier")
	return nil, nil
}
