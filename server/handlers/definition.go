package handlers

import (
	gotypes "go/types"
	"path/filepath"
	"text-template-server/types"

	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Definition handles finding the definition to jump to
func Definition(_ *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	if !GetConfig().EnableDefinition {
		log.Debug().Msg("definition requested but definition is disabled by config")
		return nil, nil
	}
	uri := params.TextDocument.URI
	position := params.Position
	doc, ok := store.Get(uri)
	if !ok {
		log.Error().Str("handler", "definition").Str("uri", uri).Msg("document not found in store")
		return nil, nil
	}

	offset := positionToOffset(doc.text, position)
	tree := doc.treeAt(parse.Pos(offset))
	if tree == nil || tree.Root == nil {
		log.Debug().
			Str("handler", "definition").
			Any("position", position).
			Msg("no parse tree at offset")
		return nil, nil
	}
	node := nodeFind(tree.Root, parse.Pos(offset))

	if node == nil {
		log.Debug().Str("handler", "definition").Any("position", position).Msg("no node found")
		return nil, nil
	}

	loadedType := doc.loadedTypeAt(parse.Pos(offset))

	switch target := node.(type) {
	case *parse.VariableNode:
		return definitionVariable(target, tree, uri, doc)
	case *parse.DotNode:
		return definitionDot(tree, node, uri, doc)
	case *parse.FieldNode:
		return definitionField(loadedType, target, offset)
	case *parse.IdentifierNode:
		return definitionIdentifier(target)
	}

	log.Debug().
		Str("handler", "definition").
		Any("node", node).
		Msg("node at position is not a field or identifier")
	return nil, nil
}

func definitionVariable(
	target *parse.VariableNode,
	tree *parse.Tree,
	uri protocol.DocumentUri,
	doc *document,
) (any, error) {
	varName := target.Ident[0]

	var results []protocol.Location

	decls := FindVarDeclarations(tree.Root, varName)
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
}

func definitionDot(
	tree *parse.Tree,
	node parse.Node,
	uri protocol.DocumentUri,
	doc *document,
) (any, error) {
	// TODO: decide if that is the correct behaviour to go to the previous range/with?

	ctx := &Context{Vars: make(map[string]parse.Node)}
	buildPath(tree.Root, node, ctx)

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
}

func definitionField(loadedType *types.Tree, target *parse.FieldNode, offset int) (any, error) {
	if loadedType == nil || loadedType.Fset == nil || loadedType.DotType == nil {
		return nil, nil
	}
	if len(target.Ident) == 0 {
		return nil, nil
	}

	targetIdx := getFieldIdentIdx(target, offset)

	// log.Debug().
	// 	Any("dotType", loadedType.DotType).
	// 	Any("Ident", target.Ident).
	// 	Any("target", targetIdx).
	// 	Any("cursorPosition", offset).
	// 	Any("fieldNodePos", target.Pos).
	// 	Msg("definition NodeField")

	var currentType gotypes.Type = loadedType.DotType
	for i := 0; i <= targetIdx; i++ {
		obj, _, _ := gotypes.LookupFieldOrMethod(
			currentType,
			true,
			loadedType.Pkg,
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
			fpos := loadedType.Fset.Position(pos)

			var line uint32
			var char uint32

			if fpos.Line > 0 && fpos.Column > 0 {
				line = uint32(fpos.Line - 1)   //nolint:gosec // disable G115
				char = uint32(fpos.Column - 1) //nolint:gosec // disable G115
			} else {
				log.Debug().Any("fpos", fpos).Msg("Definition: fpos is not > 0")
			}
			filePath := filepath.ToSlash(fpos.Filename)
			if len(filePath) > 0 && filePath[0] != '/' {
				filePath = "/" + filePath
			}

			return protocol.Location{
				URI: "file://" + filePath,
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

func definitionIdentifier(target *parse.IdentifierNode) (any, error) {
	log.Debug().
		Str("handler", "definition").
		Any("identifier", target.Ident).
		Msg("definitionIdentifier")

	entry, ok := types.GetGlobalFuncEntry(target.Ident)
	if !ok {
		return nil, nil
	}
	if entry.Fset == nil {
		return nil, nil
	}

	pos := entry.DefinitionPos()
	if !pos.IsValid() {
		return nil, nil
	}

	fpos := entry.Fset.Position(pos)
	var line uint32
	var char uint32

	if fpos.Line > 0 && fpos.Column > 0 {
		line = uint32(fpos.Line - 1)   //nolint:gosec // disable G115
		char = uint32(fpos.Column - 1) //nolint:gosec // disable G115
	} else {
		log.Debug().Any("fpos", fpos).Msg("Definition: fpos is not > 0")
	}

	filePath := filepath.ToSlash(fpos.Filename)
	if len(filePath) > 0 && filePath[0] != '/' {
		filePath = "/" + filePath
	}

	return protocol.Location{
		URI: "file://" + filePath,
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: char},
			End:   protocol.Position{Line: line, Character: char},
		},
	}, nil
}

func getFieldIdentIdx(target *parse.FieldNode, offset int) int {
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
	return targetIdx
}
