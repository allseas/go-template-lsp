package main

import (
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
		Initialize:             initialize,
		Initialized:            initialized,
		Shutdown:               shutdown,
		SetTrace:               setTrace,
		TextDocumentCompletion: completion,
		TextDocumentDidOpen:    didOpen,
		TextDocumentDidChange:  didChange,
		TextDocumentDidClose:   didClose,
		TextDocumentHover:      hover,
	}

	lspServer := server.NewServer(&handler, lsName, false)

	err := lspServer.RunStdio()
	if err != nil {
		log.Error().Err(err).Msg("error starting server")
		return
	}
}

func initialize(_ *glsp.Context, params *protocol.InitializeParams) (any, error) {
	log.Debug().Any("params", params).Msg("initializing")

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

func initialized(_ *glsp.Context, params *protocol.InitializedParams) error {
	log.Debug().Any("params", params).Msg("initialized")

	return nil
}

func shutdown(_ *glsp.Context) error {
	log.Debug().Msg("shutting down")

	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(_ *glsp.Context, params *protocol.SetTraceParams) error {
	log.Debug().Any("params", params).Msg("setting trace")
	protocol.SetTraceValue(params.Value)
	return nil
}
