package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const typehintsRoot = "../../test/resources/typehints-tests"

func hasErrorSeverity(diags []protocol.Diagnostic) bool {
	for i := range diags {
		if diags[i].Severity != nil &&
			*diags[i].Severity == protocol.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

func TestAnalyzeDocument_ReportsInvalidField(t *testing.T) {
	prevRoot := WorkspaceRoot
	t.Cleanup(func() { WorkspaceRoot = prevRoot })
	SetWorkspaceRoot(typehintsRoot)

	text := "{{- /*gotype: text-template-server/src/model.Order*/ -}}\n" +
		"{{ .CustomerName }} {{ .NoSuchField }}\n"

	diags := AnalyzeDocument("file:///tmp/analyze_bad.tmpl", text)

	require.NotEmpty(t, diags)
	assert.True(t, hasErrorSeverity(diags), "expected an error-severity diagnostic")
}

func TestAnalyzeDocument_CleanTemplate(t *testing.T) {
	prevRoot := WorkspaceRoot
	t.Cleanup(func() { WorkspaceRoot = prevRoot })
	SetWorkspaceRoot(typehintsRoot)

	// No gotype hint: dot is untyped, so a field access is not flagged.
	diags := AnalyzeDocument("file:///tmp/analyze_ok.tmpl", "{{ .Anything }}\n")

	assert.Empty(t, diags)
}

func TestAnalyzeDocument_DiagnosticsDisabled(t *testing.T) {
	prev := GetConfig()
	t.Cleanup(func() { setConfig(prev) })

	cfg := defaultConfig()
	cfg.EnableDiagnostics = false
	setConfig(cfg)

	diags := AnalyzeDocument("file:///tmp/analyze_disabled.tmpl", "{{ end }}")

	assert.Empty(t, diags)
}
