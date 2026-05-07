// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func hover(ctx *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	// Get document content
	// Analyze position
	// Build hover content

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "Not Implemented Yet",
		},
		Range: &protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 3},
		},
	}, nil
}
