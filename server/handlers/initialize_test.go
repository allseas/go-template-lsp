package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestSetupHandlers(t *testing.T) {
	setupHandlers("TestServer", "1.0.0")

	assert.Equal(t, "TestServer", lsName, "language server name should be set")
	assert.Equal(t, "1.0.0", version, "version should be set")
	assert.NotNil(t, handler.Initialize, "Initialize handler should be set")
	assert.NotNil(t, handler.TextDocumentCompletion, "Completion handler should be set")
}

func TestInitializeHandler(t *testing.T) {
	setupHandlers("goTmpl", "0.0.1")
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

func TestShutdown(t *testing.T) {
	setupHandlers("goTmpl", "0.0.1")
	err := shutdown(nil)
	assert.NoError(t, err, "shutdown handler should not return an error")
}
