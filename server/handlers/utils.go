package handlers

import (
	gotypes "go/types"

	parse "text-template-parser"
	serverTypes "text-template-server/types"
)

// lookupDotContextType returns the current dot type at target's position by
// reading the enclosing list's ValueType from the analysed (typed) tree.
// This reflects context narrowing from range/with blocks.
func lookupDotContextType(doc *document, target parse.Node) gotypes.Type {
	if doc == nil || target == nil {
		return nil
	}
	tt := doc.typedTreeAt(target.Position())
	if tt == nil || tt.Root == nil {
		return nil
	}
	if n := serverTypes.NodeFind(tt.Root, serverTypes.Pos(target.Position())); n != nil {
		if l := serverTypes.EnclosingList(n); l != nil {
			return l.ValueType()
		}
	}
	return nil
}
