package handlers

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func SemanticTokensFull(_ *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	doc, ok := store.Get(params.TextDocument.URI)

	if !ok || doc.loadedType == nil {
		return nil, nil
	}

	return nil, nil
}

func DocumentSymbols(_ *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	return nil, nil
}
