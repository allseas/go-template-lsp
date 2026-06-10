package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestRegisterGoFileWatcher(t *testing.T) {
	var calledMethod string
	var calledParams any
	var calledResult any

	ctx := &glsp.Context{
		Call: func(method string, params any, result any) {
			calledMethod = method
			calledParams = params
			calledResult = result
		},
	}

	RegisterGoFileWatcher(ctx)

	assert.Equal(t, "client/registerCapability", calledMethod)
	assert.Nil(t, calledResult)

	registrationParams, ok := calledParams.(protocol.RegistrationParams)
	require.True(t, ok, "params should be protocol.RegistrationParams")
	require.Len(t, registrationParams.Registrations, 1)

	reg := registrationParams.Registrations[0]
	assert.Equal(t, WatchedFilesRegistrationID, reg.ID)
	assert.Equal(t, "workspace/didChangeWatchedFiles", reg.Method)

	options, ok := reg.RegisterOptions.(protocol.DidChangeWatchedFilesRegistrationOptions)
	require.True(t, ok, "register options should be DidChangeWatchedFilesRegistrationOptions")
	require.Len(t, options.Watchers, 1)
	assert.Equal(t, "**/*.go", options.Watchers[0].GlobPattern)
}
