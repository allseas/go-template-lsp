package main

import (
	"fmt"
	"text-template-server/handlers"
	"text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
	lspuri "go.lsp.dev/uri"
)

var handler protocol.Handler

func uriToPath(uri string) (string, error) {
	u, err := lspuri.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI")
	}
	return u.Filename(), nil
}

// setupHandlers initializes the handler configuration with the given language server name and version. This is separated from server startup to enable testing.
func setupHandlers() {
	handler = protocol.Handler{
		Initialize:                      initialize,
		Initialized:                     initialized,
		Shutdown:                        shutdown,
		TextDocumentCompletion:          handlers.CompletionWithFallback,
		TextDocumentDidOpen:             handlers.DidOpen,
		TextDocumentDidChange:           handlers.DidChange,
		TextDocumentDidClose:            handlers.DidClose,
		TextDocumentDefinition:          handlers.Definition,
		SetTrace:                        handlers.SetTrace,
		WorkspaceDidChangeConfiguration: handlers.ConfigChanged,
		WorkspaceDidChangeWatchedFiles:  handlers.DidChangeWatchedFiles,
		TextDocumentReferences:          handlers.References,
		TextDocumentHover:               handlers.Hover,
	}
}

// Init initializes the LSP server with the provided name and version, sets up the request handlers, and starts the server using standard I/O for communication. It returns an error if the server fails to start.
func Init() error {
	setupHandlers()

	lspServer := server.NewServer(&handler, lsName, false)

	err := lspServer.RunStdio()
	if err != nil {
		log.Error().Err(err).Msg("error starting server")
		return err
	}

	return nil
}

func initialize(_ *glsp.Context, params *protocol.InitializeParams) (any, error) {
	// use RootURI as it is more modern and RootPath is deprecated
	if params.RootURI != nil {
		path, err := uriToPath(*params.RootURI)
		if err != nil {
			return nil, fmt.Errorf("initialize: %w", err)
		}
		handlers.WorkspaceRoot = path
	} else if params.RootPath != nil {
		handlers.WorkspaceRoot = *params.RootPath
	}

	if handlers.WorkspaceRoot != "" {
		if funcs, err := types.LoadGlobalFuncs(handlers.WorkspaceRoot); err != nil {
			log.Warn().Err(err).Msg("failed to load global tmpl:func hints")
		} else {
			types.SetGlobalFuncs(funcs)
			log.Debug().Int("count", len(funcs)).Msg("loaded global tmpl:func hints")
		}
	}
	capabilities := handler.CreateServerCapabilities()

	openClose := true
	changeKind := protocol.TextDocumentSyncKindFull

	capabilities.TextDocumentSync = &protocol.TextDocumentSyncOptions{
		OpenClose: &openClose,
		Change:    &changeKind,
	}

	resolveProvider := false
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"$", "."},
		ResolveProvider:   &resolveProvider,
	}
	v := version
	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &v,
		},
	}, nil
}

func initialized(context *glsp.Context, _ *protocol.InitializedParams) error {
	log.Debug().Msg("initialized")

	// so we don't block the initialized request handler.
	go func(ctx *glsp.Context) {
		if err := handlers.RequestConfig(ctx); err != nil {
			log.Error().Err(err).Msg("failed to request config")
		}
		handlers.RegisterGoFileWatcher(ctx)
	}(context)

	return nil
}

func shutdown(_ *glsp.Context) error {
	log.Debug().Msg("shutting down")

	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}
