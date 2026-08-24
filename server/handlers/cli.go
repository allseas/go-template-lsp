package handlers

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// AnalyzeDocument runs the full parse and type-analysis pipeline for a single
// document and returns its diagnostics, without requiring an LSP connection.
// It is the entry point used by the batch CLI (see server/cli.go): the same
// code path that powers editor diagnostics is reused, so results are identical
// to what an editor would show.
//
// The caller controls type-hint resolution by setting the package-level
// WorkspaceRoot before invoking this function; per-file resolution falls back
// to the directory containing the document when the root does not apply.
func AnalyzeDocument(uri, text string) []protocol.Diagnostic {
	if !GetConfig().EnableDiagnostics {
		return []protocol.Diagnostic{}
	}

	store.Set(uri, text)

	diagnostics := collectDiagnostics(text, uri)

	// Batch analysis treats every file independently. Root parse trees all
	// share the name "t", so leaving the document in the store would let one
	// file's inferred input types bleed into the next. Removing it also clears
	// this URI's entries from the shared template-input-type registry.
	store.Remove(uri)

	if diagnostics == nil {
		return []protocol.Diagnostic{}
	}
	for i := range diagnostics {
		if strings.TrimSpace(diagnostics[i].Message) == "" {
			diagnostics[i].Message = "unknown diagnostic"
		}
	}
	return diagnostics
}

// SetWorkspaceRoot sets the workspace root used for gotype-hint type
// resolution in batch mode. It mirrors what the LSP `initialize` handler does
// from the client's RootURI.
func SetWorkspaceRoot(root string) {
	WorkspaceRoot = root
}
