package handlers

import (
	gotypes "go/types"

	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Definition handles finding the definition to jump to
func Definition(_ *glsp.Context, params *protocol.DefinitionParams) (any, error) {
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

		decls := FindVarDeclarations(doc.tree.Root, varName)
		for _, decl := range decls {
			results = append(results, protocol.Location{
				URI:   uri,
				Range: nodeToRange(decl, doc.text),
			})
		}

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
		if doc.loadedType == nil || doc.loadedType.Fset == nil || doc.loadedType.DotType == nil {
			return nil, nil
		}
		target := node.(*parse.FieldNode)
		if len(target.Ident) == 0 {
			return nil, nil
		}

		// Determine which ident in the chain the cursor is on.
		// A FieldNode at byte Pos covers ".Ident[0].Ident[1]..."
		fieldOffset := int(target.Pos) + 1 // skip the leading '.'
		targetIdx := len(target.Ident) - 1 // default to last ident
		for i, ident := range target.Ident {
			if i > 0 {
				fieldOffset++ // skip the separator '.'
			}
			if offset >= fieldOffset && offset < fieldOffset+len(ident) {
				targetIdx = i
				break
			}
			fieldOffset += len(ident)
		}

		log.Debug().
			Any("dotType", doc.loadedType.DotType).
			Any("Ident", target.Ident).
			Any("target", targetIdx).
			Any("cursorPosition", offset).
			Any("fieldNodePos", target.Pos).
			Msg("definition NodeField")

		// Walk the type chain to find the Go object at targetIdx.
		var currentType gotypes.Type = doc.loadedType.DotType
		for i := 0; i <= targetIdx; i++ {
			obj, _, _ := gotypes.LookupFieldOrMethod(
				currentType,
				true,
				doc.loadedType.Pkg,
				target.Ident[i],
			)
			if obj == nil {
				return nil, nil
			}
			if i == targetIdx {
				pos := obj.Pos()
				if !pos.IsValid() {
					return nil, nil
				}
				fpos := doc.loadedType.Fset.Position(pos)

				var line uint32
				var char uint32

				if fpos.Line > 0 && fpos.Column > 0 {
					line = uint32(fpos.Line - 1)   //nolint:gosec // disable G115
					char = uint32(fpos.Column - 1) //nolint:gosec // disable G115
				} else {
					log.Debug().Any("fpos", fpos).Msg("Definition: fpos is not > 0")
				}
				return protocol.Location{
					URI: "file://" + fpos.Filename,
					Range: protocol.Range{
						Start: protocol.Position{Line: line, Character: char},
						End:   protocol.Position{Line: line, Character: char},
					},
				}, nil
			}
			switch o := obj.(type) {
			case *gotypes.Var:
				currentType = o.Type()
			case *gotypes.Func:
				sig, ok := o.Type().Underlying().(*gotypes.Signature)
				if !ok || sig.Results().Len() == 0 {
					return nil, nil
				}
				currentType = sig.Results().At(0).Type()
			default:
				return nil, nil
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
