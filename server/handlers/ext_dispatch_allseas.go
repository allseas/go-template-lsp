//go:build allseas

package handlers

import (
	"text-template-server/types"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// extDefinitionDotScope dispatches go-to-definition for a DotNode whose
// enclosing scope is provided by an extension node (a node type beyond the
// standard text/template grammar, such as TableNode). The per-extension
// predicates live in *_ext_allseas.go files alongside this dispatcher; the
// !allseas counterpart in ext_dispatch.go is a no-op.
func extDefinitionDotScope(n types.Node, uri, text string) (protocol.Location, bool) {
	if pipe, ok := tableDefinitionDotScope(n); ok {
		return protocol.Location{
			URI:   uri,
			Range: nodeRange(pipe, text),
		}, true
	}
	return protocol.Location{}, false
}
