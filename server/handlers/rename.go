package handlers

import (
	gotypes "go/types"
	"strings"
	"text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Rename renames the symbol under the cursor across the relevant template.
func Rename(_ *glsp.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || doc.typedTree == nil {
		log.Debug().Msg("rename: doc or tree is nil")
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

	// Cursor on a chained field/method segment
	if obj, isField := fieldRenameTarget(target, offset); isField {
		if obj == nil {
			return nil, nil
		}
		return renameFieldObject(tree.Root, obj, params, doc.text)
	}

	// Cursor on a base variable or function identifier.
	return renameSymbol(tree.Root, target, params, doc.text)
}

// renameSymbol renames a base variable or function identifier by matching the
// target's nodeKey against every node in the tree.
func renameSymbol(
	root types.Node,
	target types.Node,
	params *protocol.RenameParams,
	text string,
) (*protocol.WorkspaceEdit, error) {
	targetKey, ok := nodeKey(target)
	if !ok {
		return nil, nil
	}

	newName := normalizeRenameName(targetKey, params.NewName)
	if newName == "" {
		return nil, nil
	}

	var edits []protocol.TextEdit
	types.Inspect(root, func(n types.Node) bool {
		key, ok := nodeKey(n)
		if !ok || key != targetKey {
			return true
		}
		edit, ok := renameEdit(n, newName, text)
		if !ok {
			return true
		}
		edits = append(edits, edit)
		return true
	})

	if len(edits) == 0 {
		return nil, nil
	}
	return workspaceEdit(params.TextDocument.URI, edits), nil
}

// renameFieldObject renames every field/method segment in the tree that resolves
// to target. Only the matching segment of each chain is edited; preceding and
// following segments are left intact.
func renameFieldObject(
	root types.Node,
	target gotypes.Object,
	params *protocol.RenameParams,
	text string,
) (*protocol.WorkspaceEdit, error) {
	newName := strings.TrimSpace(params.NewName)
	if newName == "" {
		return nil, nil
	}

	var edits []protocol.TextEdit
	types.Inspect(root, func(n types.Node) bool {
		switch node := n.(type) {
		case *types.FieldNode:
			for i, name := range node.Ident {
				owner := walkChainType(node.DotType(), node.Ident[:i])
				if obj, ok := lookupField(owner, name); ok && obj == target {
					start := fieldSegmentStart(node, i)
					edits = append(edits, makeRenameEdit(start, len(name), newName, text))
				}
			}
		case *types.VariableNode:
			// Ident[0] is the base variable; chained fields start at index 1.
			for i := 1; i < len(node.Ident); i++ {
				owner := walkChainType(node.Base, node.Ident[1:i])
				if obj, ok := lookupField(owner, node.Ident[i]); ok && obj == target {
					start := variableSegmentStart(node, i)
					edits = append(edits, makeRenameEdit(start, len(node.Ident[i]), newName, text))
				}
			}
		}
		return true
	})

	if len(edits) == 0 {
		return nil, nil
	}
	return workspaceEdit(params.TextDocument.URI, edits), nil
}

// fieldRenameTarget reports whether the cursor sits on a chained field/method
// segment and, if so, resolves that segment to its Go object. isField is true
// for any field/method context even when the object cannot be resolved (obj is
// nil), so the caller can avoid an incorrect fallback to base-variable rename.
func fieldRenameTarget(target types.Node, offset int) (obj gotypes.Object, isField bool) {
	switch node := target.(type) {
	case *types.FieldNode:
		if len(node.Ident) == 0 {
			return nil, false
		}
		idx := getFieldIdentIdx(node, offset)
		owner := walkChainType(node.DotType(), node.Ident[:idx])
		o, _ := lookupField(owner, node.Ident[idx])
		return o, true
	case *types.VariableNode:
		idx := getVariableIdentIdx(node, offset)
		if idx == 0 {
			return nil, false // cursor is on the base variable, not a field
		}
		owner := walkChainType(node.Base, node.Ident[1:idx])
		o, _ := lookupField(owner, node.Ident[idx])
		return o, true
	}
	return nil, false
}

// lookupField resolves name as a field or method on owner.
func lookupField(owner gotypes.Type, name string) (gotypes.Object, bool) {
	if owner == nil {
		return nil, false
	}
	obj, _, _ := gotypes.LookupFieldOrMethod(owner, true, nil, name)
	if obj == nil {
		return nil, false
	}
	return obj, true
}

// fieldSegmentStart returns the byte offset of the idx-th identifier in a field
// chain (e.g. ".Address.City"), accounting for the leading and separating dots.
func fieldSegmentStart(f *types.FieldNode, idx int) int {
	pos := int(f.Position()) + 1 // skip the leading '.'
	for j := 0; j < idx; j++ {
		pos += len(f.Ident[j]) + 1 // identifier + separating '.'
	}
	return pos
}

// variableSegmentStart returns the byte offset of the idx-th identifier in a
// variable chain (e.g. "$o.Address.City"), where idx 0 is the base variable.
func variableSegmentStart(v *types.VariableNode, idx int) int {
	pos := int(v.Position())
	for j := 0; j < idx; j++ {
		pos += len(v.Ident[j]) + 1 // identifier + separating '.'
	}
	return pos
}

// normalizeRenameName trims surrounding whitespace and, for variables, ensures
// the new name retains the leading '$' that the template syntax requires.
func normalizeRenameName(targetKey, newName string) string {
	name := strings.TrimSpace(newName)
	if name == "" {
		return ""
	}
	if strings.HasPrefix(targetKey, "var:") && !strings.HasPrefix(name, "$") {
		name = "$" + name
	}
	return name
}

// renameEdit produces a TextEdit that replaces only the renamable segment of a
// node: the base variable identifier for variable/chain nodes, or the whole
// identifier for identifier nodes. Chained field accesses are left untouched.
func renameEdit(n types.Node, newName, text string) (protocol.TextEdit, bool) {
	switch node := n.(type) {
	case *types.VariableNode:
		if len(node.Ident) == 0 {
			return protocol.TextEdit{}, false
		}
		return makeRenameEdit(int(node.Position()), len(node.Ident[0]), newName, text), true
	case *types.ChainNode:
		if v, ok := node.Node.(*types.VariableNode); ok && len(v.Ident) > 0 {
			return makeRenameEdit(int(v.Position()), len(v.Ident[0]), newName, text), true
		}
	case *types.IdentifierNode:
		return makeRenameEdit(int(node.Position()), len(node.Ident), newName, text), true
	}
	return protocol.TextEdit{}, false
}

func makeRenameEdit(start, length int, newText, text string) protocol.TextEdit {
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: offsetToPosition(text, start),
			End:   offsetToPosition(text, start+length),
		},
		NewText: newText,
	}
}

func workspaceEdit(uri protocol.DocumentUri, edits []protocol.TextEdit) *protocol.WorkspaceEdit {
	return &protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentUri][]protocol.TextEdit{
			uri: edits,
		},
	}
}
