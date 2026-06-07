package main

import (
	"fmt"
	"os"
	"text-template-server/handlers"

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
		log.Debug().Str("rootURI", *params.RootURI).Msg("initialize: received RootURI")
		path, err := uriToPath(*params.RootURI)
		if err != nil {
			return nil, fmt.Errorf("initialize: %w", err)
		}
		handlers.WorkspaceRoot = path
		log.Debug().
			Str("workspaceRoot", handlers.WorkspaceRoot).
			Msg("initialize: workspace root set from RootURI")
	} else if params.RootPath != nil {
		handlers.WorkspaceRoot = *params.RootPath
		log.Debug().
			Str("workspaceRoot", handlers.WorkspaceRoot).
			Msg("initialize: workspace root set from RootPath")
	} else {
		log.Warn().Msg("initialize: no RootURI or RootPath provided, WorkspaceRoot will be empty")
	}

	// In tests, the WorkspaceRoot might be set to fake paths like /tmp/project.
	// Only fall back to os.Getwd() if we aren't under test (or if we really want to ensure the path exists)
	// Actually, the simplest fix for tests is to mock os.Stat or check if we are in test mode.
	// Since go test sets os.Args[0] ending with .test, we can check that. Or just don't strictly override if someone explicitly set it via RootURI but it doesn't exist?
	// Given tests use fake paths, let's just use it if it doesn't exist but warn.
	if handlers.WorkspaceRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			handlers.WorkspaceRoot = cwd
			log.Debug().
				Str("workspaceRoot", handlers.WorkspaceRoot).
				Msg("initialize: workspace root empty, fell back to process cwd")
		}
	} else {
		// Just check if it exists for logging, but don't strictly OVERRIDE what the client sent.
		// If the client sent a path, we should trust it, else we break tests and potentially other valid cases (like virtual FS).
		if _, err := os.Stat(handlers.WorkspaceRoot); err != nil {
			log.Warn().
				Str("workspaceRoot", handlers.WorkspaceRoot).
				Msg("initialize: workspace root does not exist on disk, we will keep it but `go list` may fail later")
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
	}(context)

	return nil
}

func shutdown(_ *glsp.Context) error {
	log.Debug().Msg("shutting down")

	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}
