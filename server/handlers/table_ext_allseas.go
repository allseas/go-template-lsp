//go:build allseas

package handlers

import "text-template-server/types"

// tableDefinitionDotScope reports whether n is a TableNode and, if so,
// returns the pipe a DotNode inside the table refers to. It is the sole
// place TableNode is mentioned in the handlers package; everything else
// goes through the generic extension dispatcher.
func tableDefinitionDotScope(n types.Node) (*types.PipeNode, bool) {
	t, ok := n.(*types.TableNode)
	if !ok {
		return nil, false
	}
	return t.Pipe, true
}
