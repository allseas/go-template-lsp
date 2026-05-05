package handlers

import (
	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

var (
	handler protocol.Handler
	version string
	lsName  string
)

func Init(lsName_ string, version_ string) error {
	lsName = lsName_
	version = version_

	handler = protocol.Handler{
		Initialize:                      initialize,
		Initialized:                     initialized,
		Shutdown:                        shutdown,
		TextDocumentCompletion:          completion,
		TextDocumentDidOpen:             didOpen,
		TextDocumentDidChange:           didChange,
		TextDocumentDidClose:            didClose,
		SetTrace:                        SetTrace,
		WorkspaceDidChangeConfiguration: ConfigChanged,
	}

	lspServer := server.NewServer(&handler, lsName, false)

	err := lspServer.RunStdio()
	if err != nil {
		log.Error().Err(err).Msg("error starting server")
		return err
	}

	return nil
}

func initialize(_ *glsp.Context, _ *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()

	openClose := true
	changeKind := protocol.TextDocumentSyncKindFull

	capabilities.TextDocumentSync = &protocol.TextDocumentSyncOptions{
		OpenClose: &openClose,
		Change:    &changeKind,
	}

	capabilities.CompletionProvider = &protocol.CompletionOptions{}
	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func initialized(context *glsp.Context, _ *protocol.InitializedParams) error {
	log.Debug().Msg("initialized")

	// so we don't block the initialized request handler.
	go func(ctx *glsp.Context) {
		if err := RequestConfig(ctx); err != nil {
			log.Error().Err(err).Msg("failed to request config")
		}
	}(context)

	return nil
}

func shutdown(_ *glsp.Context) error {
	log.Debug().Msg("shutting down")

	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}
