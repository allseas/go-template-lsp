package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestSeverityFromName(t *testing.T) {
	cases := map[string]struct {
		want protocol.DiagnosticSeverity
		ok   bool
	}{
		"error":       {protocol.DiagnosticSeverityError, true},
		"WARNING":     {protocol.DiagnosticSeverityWarning, true},
		"warn":        {protocol.DiagnosticSeverityWarning, true},
		"information": {protocol.DiagnosticSeverityInformation, true},
		"info":        {protocol.DiagnosticSeverityInformation, true},
		"hint":        {protocol.DiagnosticSeverityHint, true},
		"bogus":       {0, false},
	}
	for name, tc := range cases {
		got, ok := severityFromName(name)
		assert.Equal(t, tc.ok, ok, name)
		if tc.ok {
			assert.Equal(t, tc.want, got, name)
		}
	}
}

func TestSeverityName(t *testing.T) {
	err := protocol.DiagnosticSeverityError
	warn := protocol.DiagnosticSeverityWarning
	info := protocol.DiagnosticSeverityInformation
	hint := protocol.DiagnosticSeverityHint
	assert.Equal(t, "error", severityName(&err))
	assert.Equal(t, "warning", severityName(&warn))
	assert.Equal(t, "information", severityName(&info))
	assert.Equal(t, "hint", severityName(&hint))
	assert.Equal(t, "unknown", severityName(nil))
}

func TestAtOrAboveThreshold(t *testing.T) {
	err := protocol.DiagnosticSeverityError
	warn := protocol.DiagnosticSeverityWarning
	hint := protocol.DiagnosticSeverityHint

	// Threshold error: only errors count.
	assert.True(t, atOrAboveThreshold(&err, protocol.DiagnosticSeverityError))
	assert.False(t, atOrAboveThreshold(&warn, protocol.DiagnosticSeverityError))
	// Threshold warning: errors and warnings count.
	assert.True(t, atOrAboveThreshold(&warn, protocol.DiagnosticSeverityWarning))
	assert.True(t, atOrAboveThreshold(&err, protocol.DiagnosticSeverityWarning))
	assert.False(t, atOrAboveThreshold(&hint, protocol.DiagnosticSeverityWarning))
	// nil severity never counts.
	assert.False(t, atOrAboveThreshold(nil, protocol.DiagnosticSeverityHint))
}

func TestHasGlobMeta(t *testing.T) {
	assert.True(t, hasGlobMeta("*.tmpl"))
	assert.True(t, hasGlobMeta("a?b"))
	assert.True(t, hasGlobMeta("a[bc]"))
	assert.False(t, hasGlobMeta("plain.tmpl"))
	assert.False(t, hasGlobMeta("-"))
}

func TestExpandPaths(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.tmpl")
	b := filepath.Join(dir, "b.tmpl")
	require.NoError(t, os.WriteFile(a, []byte("{{ .A }}"), 0o600))
	require.NoError(t, os.WriteFile(b, []byte("{{ .B }}"), 0o600))

	got, err := expandPaths([]string{filepath.Join(dir, "*.tmpl"), "-", "literal.tmpl"})
	require.NoError(t, err)
	assert.Contains(t, got, a)
	assert.Contains(t, got, b)
	assert.Contains(t, got, "-")
	assert.Contains(t, got, "literal.tmpl")

	_, err = expandPaths([]string{"["})
	assert.Error(t, err)
}

func TestRenderResultsText(t *testing.T) {
	sev := protocol.DiagnosticSeverityError
	results := []fileDiagnostics{{
		Path: "x.tmpl",
		Diagnostics: []protocol.Diagnostic{{
			Range: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
			},
			Severity: &sev,
			Message:  "boom",
		}},
	}}

	var buf bytes.Buffer
	exceeded := renderResults(&buf, results, "text", protocol.DiagnosticSeverityError)

	assert.True(t, exceeded)
	assert.Equal(t, "x.tmpl:3:5: error: boom\n", buf.String())
}

func TestRenderResultsJSON(t *testing.T) {
	sev := protocol.DiagnosticSeverityWarning
	results := []fileDiagnostics{{
		Path: "y.tmpl",
		Diagnostics: []protocol.Diagnostic{{
			Severity: &sev,
			Message:  "careful",
		}},
	}}

	var buf bytes.Buffer
	exceeded := renderResults(&buf, results, "json", protocol.DiagnosticSeverityWarning)

	assert.True(t, exceeded, "warning is at the warning threshold")

	var decoded []fileDiagnostics
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	require.Len(t, decoded, 1)
	assert.Equal(t, "y.tmpl", decoded[0].Path)
	require.Len(t, decoded[0].Diagnostics, 1)
	assert.Equal(t, "careful", decoded[0].Diagnostics[0].Message)
}

// TestRenderResultsFiltersBelowThreshold verifies that diagnostics below the
// threshold are omitted from the output in both text and JSON formats.
func TestRenderResultsFiltersBelowThreshold(t *testing.T) {
	warn := protocol.DiagnosticSeverityWarning
	err := protocol.DiagnosticSeverityError
	results := []fileDiagnostics{{
		Path: "z.tmpl",
		Diagnostics: []protocol.Diagnostic{
			{Severity: &warn, Message: "just a warning"},
			{Severity: &err, Message: "a real error"},
		},
	}}

	var textBuf bytes.Buffer
	exceeded := renderResults(&textBuf, results, "text", protocol.DiagnosticSeverityError)
	assert.True(t, exceeded)
	assert.Contains(t, textBuf.String(), "a real error")
	assert.NotContains(t, textBuf.String(), "just a warning")

	var jsonBuf bytes.Buffer
	renderResults(&jsonBuf, results, "json", protocol.DiagnosticSeverityError)
	var decoded []fileDiagnostics
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &decoded))
	require.Len(t, decoded, 1)
	require.Len(t, decoded[0].Diagnostics, 1)
	assert.Equal(t, "a real error", decoded[0].Diagnostics[0].Message)
}

func TestRenderResultsJSON_EmptyIsArray(t *testing.T) {
	var buf bytes.Buffer
	renderResults(&buf, nil, "json", protocol.DiagnosticSeverityError)
	assert.Equal(t, "[]", strings.TrimSpace(buf.String()))
}

func TestRunCheck_NoFiles(t *testing.T) {
	var out, errb bytes.Buffer
	code := runCheck(nil, &out, &errb, strings.NewReader(""))
	assert.Equal(t, 2, code)
	assert.Contains(t, errb.String(), "no files given")
}

func TestRunCheck_InvalidFlags(t *testing.T) {
	var out, errb bytes.Buffer
	assert.Equal(t, 2, runCheck(
		[]string{"--min-severity", "nope", "x"}, &out, &errb, strings.NewReader(""),
	))

	out.Reset()
	errb.Reset()
	assert.Equal(t, 2, runCheck(
		[]string{"--format", "nope", "x"}, &out, &errb, strings.NewReader(""),
	))
}

func TestRunCheck_SyntaxErrorFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.tmpl")
	require.NoError(t, os.WriteFile(f, []byte("{{ end }}"), 0o600))

	var out, errb bytes.Buffer
	code := runCheck([]string{f}, &out, &errb, strings.NewReader(""))

	assert.Equal(t, 1, code)
	assert.Contains(t, out.String(), "error:")
}

func TestRunCheck_CleanFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "ok.tmpl")
	require.NoError(t, os.WriteFile(f, []byte("{{ .Anything }}\n"), 0o600))

	var out, errb bytes.Buffer
	code := runCheck([]string{f}, &out, &errb, strings.NewReader(""))

	assert.Equal(t, 0, code)
	assert.Empty(t, strings.TrimSpace(out.String()))
}

func TestRunCheck_Stdin(t *testing.T) {
	var out, errb bytes.Buffer
	code := runCheck([]string{"-"}, &out, &errb, strings.NewReader("{{ end }}"))

	assert.Equal(t, 1, code)
	assert.Contains(t, out.String(), "<stdin>")
}

// TestRunCheck_SeedsGlobalFuncs verifies that runCheck seeds the global funcs
// cache from the workspace, so a workspace-defined //tmpl:func "global"
// function (and builtins) are not flagged as undefined in batch mode.
func TestRunCheck_SeedsGlobalFuncs(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "funcs.tmpl")
	// "upper" is defined by the fixture's global FuncMap; "len" is a builtin.
	require.NoError(t, os.WriteFile(f, []byte(`{{ upper "x" }} {{ len "y" }}`+"\n"), 0o600))

	var out, errb bytes.Buffer
	code := runCheck(
		[]string{"--root", "../test/resources/funcmap-tests", f},
		&out, &errb, strings.NewReader(""),
	)

	assert.Equal(t, 0, code)
	assert.NotContains(t, out.String(), "undefined function")
	assert.Empty(t, strings.TrimSpace(out.String()))
}

func TestRunCheck_ReadError(t *testing.T) {
	var out, errb bytes.Buffer
	code := runCheck(
		[]string{filepath.Join(t.TempDir(), "missing.tmpl")},
		&out, &errb, strings.NewReader(""),
	)
	assert.Equal(t, 2, code)
}

func TestRunCheck_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.tmpl")
	require.NoError(t, os.WriteFile(f, []byte("{{ end }}"), 0o600))

	var out, errb bytes.Buffer
	code := runCheck([]string{"--format", "json", f}, &out, &errb, strings.NewReader(""))

	assert.Equal(t, 1, code)
	var decoded []fileDiagnostics
	require.NoError(t, json.Unmarshal(out.Bytes(), &decoded))
	require.Len(t, decoded, 1)
	assert.NotEmpty(t, decoded[0].Diagnostics)
}
