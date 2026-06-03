package main

import (
	"path/filepath"
	"testing"
	"text-template-server/handlers"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestSetupHandlers(t *testing.T) {
	setupHandlers()

	assert.Equal(t, "goTmpl", lsName, "language server name should be set")
	assert.NotNil(t, version, "version should be set")
	assert.NotNil(t, handler.Initialize, "Initialize handler should be set")
	assert.NotNil(t, handler.TextDocumentCompletion, "Completion handler should be set")
}

func TestInitializeHandler(t *testing.T) {
	setupHandlers()
	result, err := initialize(nil, &protocol.InitializeParams{})

	assert.NoError(t, err, "initialize handler should not return an error")
	assert.NotNil(t, result, "initialize should return a result")

	initResult, ok := result.(protocol.InitializeResult)
	assert.True(t, ok, "result should be InitializeResult")
	assert.NotNil(t, initResult.Capabilities, "capabilities should be set")
	assert.NotNil(t, initResult.ServerInfo, "server info should be set")
	assert.Equal(t, "goTmpl", initResult.ServerInfo.Name, "server name should be set")
	assert.NotNil(t, initResult.ServerInfo.Version, "server version should be set")
}

func TestInitializeSetsWorkspaceRootAndCapabilities(t *testing.T) {
	setupHandlers()
	handlers.WorkspaceRoot = ""
	rootURI := "file:///tmp/project"

	result, err := initialize(nil, &protocol.InitializeParams{RootURI: &rootURI})
	assert.NoError(t, err)

	initResult, ok := result.(protocol.InitializeResult)
	assert.True(t, ok)
	assert.Equal(t, filepath.FromSlash("/tmp/project"), handlers.WorkspaceRoot)

	requireCaps := initResult.Capabilities
	assert.NotNil(t, requireCaps.TextDocumentSync)
	assert.NotNil(t, requireCaps.CompletionProvider)
	assert.Equal(t, []string{"$", "."}, requireCaps.CompletionProvider.TriggerCharacters)
	assert.NotNil(t, requireCaps.CompletionProvider.ResolveProvider)
	assert.False(t, *requireCaps.CompletionProvider.ResolveProvider)
	syncOpts, ok := requireCaps.TextDocumentSync.(*protocol.TextDocumentSyncOptions)
	assert.True(t, ok)
	assert.NotNil(t, syncOpts.OpenClose)
	assert.True(t, *syncOpts.OpenClose)
}

func TestInitializeUsesRootPathWhenRootURIMissing(t *testing.T) {
	setupHandlers()
	handlers.WorkspaceRoot = ""
	rootPath := "C:/repo/server"

	_, err := initialize(nil, &protocol.InitializeParams{RootPath: &rootPath})
	assert.NoError(t, err)
	assert.Equal(t, rootPath, handlers.WorkspaceRoot)
}

func TestURIToPathInvalidURI(t *testing.T) {
	badURI := "%zzzz"
	res, err := uriToPath(badURI)
	assert.Error(t, err)
	assert.Empty(t, res)
}

func TestShutdown(t *testing.T) {
	setupHandlers()
	err := shutdown(nil)
	assert.NoError(t, err, "shutdown handler should not return an error")
}
