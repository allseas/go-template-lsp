package handlers

import (
	"testing"
	"text-template-server/types"

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

func TestDidChangeWatchedFiles_EmptyWorkspaceRoot(t *testing.T) {
	defer func() { WorkspaceRoot = "" }()
	WorkspaceRoot = ""

	ctx := &glsp.Context{
		Call: func(method string, _ any, _ any) {
			t.Errorf("unexpected Call: method=%s", method)
		},
	}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{URI: "file:///test.go", Type: protocol.FileChangeTypeChanged},
		},
	}

	err := DidChangeWatchedFiles(ctx, params)
	assert.NoError(t, err, "should return nil without error when WorkspaceRoot is empty")
}

func TestDidChangeWatchedFiles_NoGoFileChanges(t *testing.T) {
	defer func() { WorkspaceRoot = "" }()
	WorkspaceRoot = "/test"

	ctx := &glsp.Context{
		Call: func(method string, _ any, _ any) {
			t.Errorf("unexpected Call: method=%s", method)
		},
	}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{URI: "file:///test.txt", Type: protocol.FileChangeTypeChanged},
			{URI: "file:///test.md", Type: protocol.FileChangeTypeDeleted},
		},
	}

	err := DidChangeWatchedFiles(ctx, params)
	assert.NoError(t, err, "should return nil without error when no .go files changed")
}

func TestDidChangeWatchedFiles_GoFileChanged(t *testing.T) {
	defer func() { WorkspaceRoot = "" }()
	WorkspaceRoot = t.TempDir()

	ctx := &glsp.Context{
		Notify: func(_ string, _ any) {},
	}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{URI: "file:///test.txt", Type: protocol.FileChangeTypeChanged},
			{URI: "file:///main.go", Type: protocol.FileChangeTypeChanged},
		},
	}

	err := DidChangeWatchedFiles(ctx, params)
	assert.NoError(t, err, "should successfully handle .go file changes")
	assert.NotNil(t, types.GlobalFuncs(), "GlobalFuncs should be updated with new functions")
}

func TestDidChangeWatchedFiles_PercentEncodedGoFile(t *testing.T) {
	defer func() { WorkspaceRoot = "" }()
	WorkspaceRoot = t.TempDir()

	ctx := &glsp.Context{
		Notify: func(_ string, _ any) {},
	}

	// Use a percent-encoded URI—anyGoChange should still detect it as .go
	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{URI: "file:///test%20dir/main.go", Type: protocol.FileChangeTypeChanged},
		},
	}

	err := DidChangeWatchedFiles(ctx, params)
	assert.NoError(t, err, "should detect percent-encoded .go files")
	assert.NotNil(
		t,
		types.GlobalFuncs(),
		"GlobalFuncs should be updated with new functions after percent-encoded .go file change",
	)
}

func TestAnyGoChange(t *testing.T) {
	tests := []struct {
		name     string
		changes  []protocol.FileEvent
		expected bool
	}{
		{
			name:     "no changes",
			changes:  []protocol.FileEvent{},
			expected: false,
		},
		{
			name: "only non-go files",
			changes: []protocol.FileEvent{
				{URI: "file:///test.txt", Type: protocol.FileChangeTypeChanged},
				{URI: "file:///test.md", Type: protocol.FileChangeTypeDeleted},
			},
			expected: false,
		},
		{
			name: "one go file among others",
			changes: []protocol.FileEvent{
				{URI: "file:///test.txt", Type: protocol.FileChangeTypeChanged},
				{URI: "file:///main.go", Type: protocol.FileChangeTypeChanged},
			},
			expected: true,
		},
		{
			name: "multiple go files",
			changes: []protocol.FileEvent{
				{URI: "file:///pkg/main.go", Type: protocol.FileChangeTypeChanged},
				{URI: "file:///pkg/helper.go", Type: protocol.FileChangeTypeCreated},
			},
			expected: true,
		},
		{
			name: "percent-encoded go file",
			changes: []protocol.FileEvent{
				{URI: "file:///test%20dir/main.go", Type: protocol.FileChangeTypeChanged},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := anyGoChange(tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
