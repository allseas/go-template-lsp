//go:build !allseas

package handlers

import (
	"text-template-server/types"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// extDefinitionDotScope is the !allseas no-op counterpart of the dispatcher
// in ext_dispatch_allseas.go. With no extension node types compiled in,
// nothing dispatches.
func extDefinitionDotScope(_ types.Node, _, _ string) (protocol.Location, bool) {
	return protocol.Location{}, false
}

func WelcomeMessage() string {
	return "NOT ALLSEAS"
}
