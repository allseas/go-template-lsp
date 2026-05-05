package main

import (
	"text-template-server/handlers"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

const lsName = "goTmpl"

var (
	version = "0.0.1"
	handler protocol.Handler
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	log.Print("starting server")

	handler = protocol.Handler{
		Initialize:                      initialize,
		Initialized:                     initialized,
		Shutdown:                        shutdown,
		TextDocumentCompletion:          completion,
		TextDocumentDidOpen:             didOpen,
		TextDocumentDidChange:           didChange,
		TextDocumentDidClose:            didClose,
		SetTrace:                        handlers.SetTrace,
		WorkspaceDidChangeConfiguration: handlers.ConfigChanged,
	}

	lspServer := server.NewServer(&handler, lsName, false)

	err := lspServer.RunStdio()
	if err != nil {
		log.Error().Err(err).Msg("error starting server")
		return
	}
}

func initialize(_ *glsp.Context, params *protocol.InitializeParams) (any, error) {
	// log.Debug().Any("params", params).Msg("initializing")

	capabilities := handler.CreateServerCapabilities()

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	log.Debug().Any("params", params).Msg("initialized")

	go handlers.RequestConfig(context)

	return nil
}

func shutdown(_ *glsp.Context) error {
	log.Debug().Msg("shutting down")

	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}
