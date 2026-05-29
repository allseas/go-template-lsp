package handlers

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
	lspuri "go.lsp.dev/uri"
)

var (
	handler       protocol.Handler
	version       string
	lsName        string
	workspaceRoot string
)

func uriToPath(uri string) (string, error) {
	u, err := lspuri.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI")
	}
	return u.Filename(), nil
}

// setupHandlers initializes the handler configuration with the given language server name and version. This is separated from server startup to enable testing.
func setupHandlers(langServerName string, langServerVersion string) {
	lsName = langServerName
	version = langServerVersion

	handler = protocol.Handler{
		Initialize:                      initialize,
		Initialized:                     initialized,
		Shutdown:                        shutdown,
		TextDocumentCompletion:          completionWithFallback,
		TextDocumentDidOpen:             didOpen,
		TextDocumentDidChange:           didChange,
		TextDocumentDidClose:            didClose,
		TextDocumentDefinition:          definition,
		SetTrace:                        SetTrace,
		WorkspaceDidChangeConfiguration: ConfigChanged,
		TextDocumentReferences:          references,
		TextDocumentHover:               hover,
	}
}

// Init initializes the LSP server with the provided name and version, sets up the request handlers, and starts the server using standard I/O for communication. It returns an error if the server fails to start.
func Init(langServerName string, langServerVersion string) error {
	setupHandlers(langServerName, langServerVersion)

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
		workspaceRoot = path
	} else if params.RootPath != nil {
		workspaceRoot = *params.RootPath
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
