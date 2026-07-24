package handlers

import (
	"text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// References finds and outputs all references for a selected variable or function
func References(_ *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || doc.rootTypedTree() == nil {
		log.Debug().Msg("doc or tree is nil")
		return nil, nil
	}

	offset := positionToOffset(doc.text, params.Position)
	tree := doc.typedTreeAtTyped(types.Pos(offset))
	if tree == nil || tree.Root == nil {
		return nil, nil
	}
	target := types.NodeFind(tree.Root, types.Pos(offset))
	if target == nil {
		return nil, nil
	}

	targetKey, ok := nodeKey(target)
	if !ok {
		return nil, nil
	}

	var results []protocol.Location
	includeDecl := params.Context.IncludeDeclaration

	types.Inspect(tree.Root, func(n types.Node) bool {
		key, ok := nodeKey(n)
		if !ok || key != targetKey {
			return true
		}

		if !includeDecl && isVarDecl(n, targetKey) {
			return true
		}

		results = append(results, protocol.Location{
			URI:   params.TextDocument.URI,
			Range: nodeRange(n, doc.text),
		})
		return true
	})

	return results, nil
}

// isVarDecl determines if a node is a definition of a variable for proper reference highlighting
func isVarDecl(n types.Node, targetKey string) bool {
	v, ok := n.(*types.VariableNode)
	if !ok || len(v.Ident) == 0 {
		return false
	}
	return "var:"+v.Ident[0] == targetKey
}

func nodeKey(n types.Node) (string, bool) {
	switch node := n.(type) {
	case *types.VariableNode:
		if len(node.Ident) > 0 {
			return "var:" + node.Ident[0], true
		}
	case *types.FieldNode:
		return "", false
	case *types.ChainNode:
		if v, ok := node.Node.(*types.VariableNode); ok && len(v.Ident) > 0 {
			return "var:" + v.Ident[0], true
		}
	case *types.IdentifierNode:
		return "id:" + node.Ident, true
	}
	return "", false
}

// offsetToPosition converts the integer offset into Position()
func offsetToPosition(text string, offset int) protocol.Position {
	line := uint32(0)
	charUTF16 := uint32(0)

	for i, r := range text {
		if i >= offset {
			break
		}
		if r == '\n' {
			line++
			charUTF16 = 0
			continue
		}

		if r > 0xFFFF {
			charUTF16 += 2
		} else {
			charUTF16++
		}
	}

	return protocol.Position{
		Line:      line,
		Character: charUTF16,
	}
}
