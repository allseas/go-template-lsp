package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCollectDiagnosticsNoErrors(t *testing.T) {
	src := `{{ $x := 1 }}{{ $y := 2 }}`
	uri := "file:///diag_clean.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)
	assert.Empty(t, diags)
}

func TestCollectDiagnosticsSyntaxError(t *testing.T) {
	src := `{{ $x := }}`
	uri := "file:///diag_syntax.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)
	require.Len(t, diags, 1)
	assert.Equal(t, protocol.DiagnosticSeverityError, *diags[0].Severity)
}

func TestCollectDiagnosticsSyntaxErrorUnderlineBlock(t *testing.T) {
	src := `{{ $x := }}`
	uri := "file:///diag_underline.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)
	require.Len(t, diags, 1)
	assert.Equal(t, u32(0), diags[0].Range.Start.Character)
	assert.Equal(t, u32(len(src)), diags[0].Range.End.Character)
}

func TestCollectDiagnosticsDuplicateVariable(t *testing.T) {
	src := "{{ $x := 1 }}\n{{ $x := 2 }}"
	uri := "file:///diag_dup.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)
	require.Len(t, diags, 1)
	assert.Equal(t, protocol.DiagnosticSeverityWarning, *diags[0].Severity)
	assert.Contains(t, diags[0].Message, "$x")
}

func TestCollectDiagnosticsDuplicateVariableRange(t *testing.T) {
	src := "{{ $x := 1 }}\n{{ $x := 2 }}"
	uri := "file:///diag_dup_range.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)
	require.Len(t, diags, 1)
	assert.Equal(t, uint32(1), diags[0].Range.Start.Line)
}

func TestCollectDiagnosticsDuplicateAndSyntaxError(t *testing.T) {
	src := "{{ $x := 1 }}\n{{ $x := 2 }}\n{{ $y := }}"
	uri := "file:///diag_both.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)

	var errors, warnings []protocol.Diagnostic
	for _, d := range diags {
		switch *d.Severity {
		case protocol.DiagnosticSeverityError:
			errors = append(errors, d)
		case protocol.DiagnosticSeverityWarning:
			warnings = append(warnings, d)
		}
	}

	assert.Len(t, errors, 1)
	assert.Len(t, warnings, 1)
}

func TestCollectDiagnosticsRangeFreshScope(t *testing.T) {
	src := "{{ $x := 1 }}\n{{ range .Items }}\n{{ $x := . }}\n{{ end }}"
	uri := "file:///diag_range_scope.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)
	assert.Empty(t, diags)
}

func TestCollectDiagnosticsNoDuplicateAcrossDifferentVars(t *testing.T) {
	src := "{{ $x := 1 }}\n{{ $y := 2 }}\n{{ $z := 3 }}"
	uri := "file:///diag_no_dup.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)
	assert.Empty(t, diags)
}

func TestCollectDiagnosticsUndefinedVariableError(t *testing.T) {
	src := `{{ $x := }}`
	uri := "file:///diag_undef.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	diags := collectDiagnostics(src, uri)
	require.Len(t, diags, 1)
	assert.Equal(t, protocol.DiagnosticSeverityError, *diags[0].Severity)
	assert.Equal(t, u32(0), diags[0].Range.Start.Character)
	assert.Equal(t, u32(len(src)), diags[0].Range.End.Character)
}
