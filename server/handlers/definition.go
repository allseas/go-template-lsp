package handlers

import (
	gotypes "go/types"
	"text-template-server/types"
	"text-template-server/utils"

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
	tree := doc.typedTreeAtTyped(types.Pos(offset))
	if tree == nil || tree.Root == nil {
		log.Debug().
			Str("handler", "definition").
			Any("position", position).
			Msg("no parse tree at offset")
		return nil, nil
	}
	node := types.NodeFind(tree.Root, types.Pos(offset))

	if node == nil {
		log.Debug().Str("handler", "definition").Any("position", position).Msg("no node found")
		return nil, nil
	}

	loadedType := doc.loadedTypeAtTyped(types.Pos(offset))

	switch target := node.(type) {
	case *types.VariableNode:
		varName := target.Ident[0]
		identIdx := getVariableIdentIdx(target, offset)

		// Cursor is on a chained field/method (e.g. ".IsExpensive" in "$item.IsExpensive"):
		// resolve through the variable's type and navigate to the Go source.
		if identIdx > 0 {
			decls := FindVarDeclarationsTyped(tree.Root, varName)
			for _, decl := range decls {
				if decl.ValueType() != nil {
					return resolveFieldChainDefinition(
						loadedType,
						decl.ValueType(),
						target.Ident[1:],
						identIdx-1,
					)
				}
			}
			return nil, nil
		}

		// Cursor is on the base variable: jump to its declaration.
		var results []protocol.Location
		decls := FindVarDeclarationsTyped(tree.Root, varName)
		for _, decl := range decls {
			results = append(results, protocol.Location{
				URI:   uri,
				Range: nodeRange(decl, doc.text),
			})
		}

		if len(results) == 0 {
			return nil, nil
		}
		return results, nil
	case *types.DotNode:
		// TODO: decide if that is the correct behaviour to go to the previous range/with?
		for cur := node.Parent(); cur != nil; cur = cur.Parent() {
			switch branch := cur.(type) {
			case *types.RangeNode:
				return protocol.Location{
					URI:   uri,
					Range: nodeRange(branch.Pipe, doc.text),
				}, nil
			case *types.WithNode:
				return protocol.Location{
					URI:   uri,
					Range: nodeRange(branch.Pipe, doc.text),
				}, nil
			default:
				if loc, ok := extDefinitionDotScope(cur, uri, doc.text); ok {
					return loc, nil
				}
			}
		}
		return nil, nil
	case *types.FieldNode:
		return fieldNodeDefinition(loadedType, dotTypeAt(target), target, offset)
	case *types.IdentifierNode:
		return definitionIdentifier(target)
	case *types.TemplateNode:
		return templateDefinition(target.Name, doc.typedTrees, uri, doc.text)
	}

	log.Debug().
		Str("handler", "definition").
		Any("node", node).
		Msg("node at position is not a field or identifier")
	return nil, nil
}

func templateDefinition(
	templateName string,
	typedTrees map[string]*types.Tree,
	uri protocol.DocumentUri,
	docText string,
) (any, error) {
	if tree, ok := typedTrees[templateName]; ok && tree != nil && tree.Root != nil {
		return protocol.Location{
			URI:   uri,
			Range: nodeRange(tree.Root, docText),
		}, nil
	}
	return nil, nil
}

func fieldNodeDefinition(
	loadedType *types.Tree,
	dotType gotypes.Type,
	target *types.FieldNode,
	offset int,
) (any, error) {
	if loadedType == nil || loadedType.Fset == nil {
		return nil, nil
	}
	if dotType == nil {
		dotType = loadedType.DotType
	}
	if dotType == nil {
		return nil, nil
	}
	if len(target.Ident) == 0 {
		return nil, nil
	}

	targetIdx := getFieldIdentIdx(target, offset)
	return resolveFieldChainDefinition(loadedType, dotType, target.Ident, targetIdx)
}

// resolveFieldChainDefinition walks idents from baseType up to and including targetIdx,
// then returns the LSP Location pointing at that field or method in the Go source.
func resolveFieldChainDefinition(
	loadedType *types.Tree,
	baseType gotypes.Type,
	idents []string,
	targetIdx int,
) (any, error) {
	if loadedType == nil || loadedType.Fset == nil || baseType == nil || len(idents) == 0 {
		return nil, nil
	}

	currentType := baseType
	for i := 0; i <= targetIdx; i++ {
		if d, ok := currentType.(*types.DictType); ok {
			valueTyp, keyOk := d.LookupDictKey(idents[i])
			if !keyOk {
				return nil, nil
			}
			if i == targetIdx {
				named := toNamed(valueTyp)
				if named == nil {
					return nil, nil
				}
				return namedTypeLocation(loadedType, named)
			}
			currentType = valueTyp
			continue
		}
		obj, _, _ := gotypes.LookupFieldOrMethod(
			currentType,
			true,
			loadedType.Pkg,
			idents[i],
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
			return protocol.Location{
				URI: utils.FilePathToURI(fpos.Filename),
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

// namedTypeLocation returns the LSP Location of a *types.Named type declaration.
func namedTypeLocation(loadedType *types.Tree, named *gotypes.Named) (any, error) {
	pos := named.Obj().Pos()
	if !pos.IsValid() {
		return nil, nil
	}
	fpos := loadedType.Fset.Position(pos)
	var line, char uint32
	if fpos.Line > 0 && fpos.Column > 0 {
		line = uint32(fpos.Line - 1)   //nolint:gosec // disable G115
		char = uint32(fpos.Column - 1) //nolint:gosec // disable G115
	}
	return protocol.Location{
		URI: utils.FilePathToURI(fpos.Filename),
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: char},
			End:   protocol.Position{Line: line, Character: char},
		},
	}, nil
}

func definitionIdentifier(target *types.IdentifierNode) (any, error) {
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

	return protocol.Location{
		URI: utils.FilePathToURI(fpos.Filename),
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: char},
			End:   protocol.Position{Line: line, Character: char},
		},
	}, nil
}

func getFieldIdentIdx(target *types.FieldNode, offset int) int {
	fieldOffset := int(target.Pos) + 1 // skip the leading '.'
	targetIdx := len(target.Ident) - 1 // default to last ident
	for i, ident := range target.Ident {
		if i > 0 {
			fieldOffset++ // skip the separator '.'
		}
		if offset >= fieldOffset && offset <= fieldOffset+len(ident) {
			targetIdx = i
			break
		}
		fieldOffset += len(ident)
	}
	return targetIdx
}

// getVariableIdentIdx returns the index into target.Ident that the cursor (offset) falls on.
// Ident[0] is the base variable (e.g. "$item"); Ident[1:] are chained fields/methods.
// Returns 0 when the cursor is on the base variable or the position is ambiguous.
func getVariableIdentIdx(target *types.VariableNode, offset int) int {
	pos := int(target.Pos)
	for i, ident := range target.Ident {
		end := pos + len(ident)
		if offset >= pos && offset <= end {
			return i
		}
		pos = end + 1 // +1 for the '.' separator
	}
	return 0
}
