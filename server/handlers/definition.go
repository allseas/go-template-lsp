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

	switch node.Type() {
	case parse.NodeVariable:
		target := node.(*parse.VariableNode)
		varName := target.Ident[0]

		var results []protocol.Location

		// this goes over the tree and finds declarations (inside PipeNode) of varName
		inspect(doc.tree.Root, func(n parse.Node) bool {
			pipe, ok := n.(*parse.PipeNode)
			if !ok {
				return true
			}
			for _, decl := range pipe.Decl {
				if decl.Ident[0] == varName {
					results = append(results, protocol.Location{
						URI:   uri,
						Range: nodeToRange(decl, doc.text),
					})
				}
			}
			return true
		})

		if len(results) == 0 {
			return nil, nil
		}
		return results, nil
	case parse.NodeDot:
		// TODO: decide if that is the correct behaviour to go to the previous range/with?
		ctx := &Context{Vars: make(map[string]parse.Node)}
		buildPath(doc.tree.Root, node, ctx)

		for i := len(ctx.Path) - 1; i >= 0; i-- {
			switch branch := ctx.Path[i].(type) {
			case *parse.RangeNode:
				return protocol.Location{
					URI:   uri,
					Range: nodeToRange(branch.Pipe, doc.text),
				}, nil
			case *parse.WithNode:
				return protocol.Location{
					URI:   uri,
					Range: nodeToRange(branch.Pipe, doc.text),
				}, nil
			}
		}
		return nil, nil
	case parse.NodeField:
		// TODO: go to the definition in the go files
		return nil, nil
	}

	log.Debug().
		Str("handler", "definition").
		Any("node", node).
		Msg("node at position is not a field or identifier")
	return nil, nil
}
