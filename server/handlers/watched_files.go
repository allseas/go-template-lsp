package handlers

import (
	"path/filepath"
	"strings"
	"text-template-server/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	lspuri "go.lsp.dev/uri"
)

// WatchedFilesRegistrationID is the unique ID to register the .go file watcher.
const WatchedFilesRegistrationID = "gotmpl-global-funcs-watcher"

// RegisterGoFileWatcher registers reply from client whenever a `.go` file in
// the workspace is created, changed, or deleted, to refresh the cached
// `//tmpl:func "global"` map.
func RegisterGoFileWatcher(ctx *glsp.Context) {
	if ctx == nil {
		return
	}
	params := protocol.RegistrationParams{
		Registrations: []protocol.Registration{
			{
				ID:     WatchedFilesRegistrationID,
				Method: "workspace/didChangeWatchedFiles",
				RegisterOptions: protocol.DidChangeWatchedFilesRegistrationOptions{
					Watchers: []protocol.FileSystemWatcher{
						{GlobPattern: "**/*.go"},
					},
				},
			},
		},
	}
	ctx.Call("client/registerCapability", params, nil)
}

// DidChangeWatchedFiles is the LSP handler for `workspace/didChangeWatchedFiles`.
// It reloads the global tmpl:func cache whenever a relevant `.go` file changes
// and re-analyses every open template document so diagnostics and completions
// reflect the new function set.
func DidChangeWatchedFiles(ctx *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	if WorkspaceRoot == "" {
		return nil
	}
	if !anyGoChange(params.Changes) {
		return nil
	}

	// Invalidate the type-hint cache first so ComputeGlobalFuncs can re-seed
	// it with the packages it type-checks. See RegisterLoadedPackage.
	types.InvalidateTypeHintCache()

	funcs, err := types.ComputeGlobalFuncs(WorkspaceRoot)
	if err != nil {
		log.Warn().Err(err).Msg("failed to reload global tmpl:func hints")
		return nil
	}
	types.SetGlobalFuncs(funcs)
	log.Debug().Int("count", len(funcs)).Msg("reloaded global tmpl:func hints")

	RefreshAllDocuments(ctx)
	return nil
}

func anyGoChange(changes []protocol.FileEvent) bool {
	for _, ev := range changes {
		if isGoURI(ev.URI) {
			return true
		}
	}
	return false
}

func isGoURI(uri string) bool {
	if strings.HasSuffix(uri, ".go") {
		return true
	}
	// Fall back to a strict parse in case the URI is percent-encoded.
	u, err := lspuri.Parse(uri)
	if err != nil {
		return false
	}
	return filepath.Ext(u.Filename()) == ".go"
}
