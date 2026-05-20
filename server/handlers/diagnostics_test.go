package handlers

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// helpers
func diagMessages(diags []protocol.Diagnostic) []string {
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	return msgs
}

func findDiagnosticContaining(diags []protocol.Diagnostic, substr string) (*protocol.Diagnostic, bool) {
	for i := range diags {
		if strings.Contains(diags[i].Message, substr) {
			return &diags[i], true
		}
	}
	return nil, false
}

func assertDiagnosticCoversTextRange(t *testing.T, diag protocol.Diagnostic, text string, startOffset, endOffset int) {
	t.Helper()

	wantStart := offsetToPosition(text, startOffset)
	wantEnd := offsetToPosition(text, endOffset)

	assert.Equal(t, wantStart.Line, diag.Range.Start.Line)
	assert.Equal(t, wantStart.Character, diag.Range.Start.Character)
	assert.Equal(t, wantEnd.Line, diag.Range.End.Line)
	assert.Equal(t, wantEnd.Character, diag.Range.End.Character)
}

func TestIsUnparsedText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"variable with dollar", "$myVar", false},
		{"dot accessor", ".Field", false},
		{"pipe separator", "funcA | funcB", false},
		{"backtick string", "`raw string`", false},
		{"colon-equals assign", "$x := 1", false},
		{"pure number", "42", false},
		{"pure spaces", "   ", false},
		{"plain word", "something", true},
		{"alpha only", "abc", true},
		{"mixed non-special", "hello world", true},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, isUnparsedText(tc.input), tc.name)
	}
}

func u32(v int) uint32 {
	if v > math.MaxUint32 {
		return math.MaxUint32
	}
	if v >= 0 {
		return uint32(v)
	}
	return 0
}

func TestExpandToFullBracketsFromOffset(t *testing.T) {
	text := `hello {{ .Name }} world`
	pos := strings.Index(text, ".Name")
	rng := expandToFullBracketsFromOffset(pos, text)

	openIdx := strings.Index(text, "{{")
	closeIdx := strings.Index(text, "}}") + 2

	assert.Equal(t, u32(openIdx), rng.Start.Character)
	assert.Equal(t, u32(closeIdx), rng.End.Character)

	text = "line1\n{{ .Name \nline3"
	pos = strings.Index(text, ".Name")
	rng = expandToFullBracketsFromOffset(pos, text)

	endPos := offsetToPosition(text, pos+len(".Name "))
	assert.LessOrEqual(t, rng.End.Line, endPos.Line+1)

	text = "{{ .X }}"
	rng = expandToFullBracketsFromOffset(len(text), text)
	assert.LessOrEqual(t, rng.Start.Character, rng.End.Character+1)

	text = "{{ foo }}"
	for i := 0; i < len(text); i++ {
		rng = expandToFullBracketsFromOffset(i, text)
		startOff := positionToOffset(text, rng.Start)
		endOff := positionToOffset(text, rng.End)
		assert.LessOrEqualf(t, startOff, endOff, "at pos %d", i)
	}
}

func TestHasExactDiagnosticAtRange(t *testing.T) {
	text := "line one\n{{ .Bad }}\nline three"

	openIdx := strings.Index(text, "{{")
	closeIdx := strings.Index(text, "}}") + 2

	existingDiag := protocol.Diagnostic{
		Range: protocol.Range{
			Start: offsetToPosition(text, openIdx),
			End:   offsetToPosition(text, closeIdx),
		},
		Message: "already reported",
	}

	found := hasExactDiagnosticAtRange(
		[]protocol.Diagnostic{existingDiag},
		openIdx,
		closeIdx,
		text,
	)
	assert.True(t, found)

	found = hasExactDiagnosticAtRange(nil, openIdx, closeIdx, text)
	assert.False(t, found)

	otherStart := strings.Index(text, "line three")
	found = hasExactDiagnosticAtRange(
		[]protocol.Diagnostic{existingDiag},
		otherStart,
		otherStart+5,
		text,
	)
	assert.False(t, found)
}

func TestExtractBranchNodes(t *testing.T) {
	pipe, list, elseList := extractBranchNodes(nil)
	assert.Nil(t, pipe)
	assert.Nil(t, list)
	assert.Nil(t, elseList)
}

func TestCollectDiagnostics_EmptyAndTrivial(t *testing.T) {
	diags := collectDiagnostics("", "file:///test.tmpl")
	assert.Empty(t, diags)

	diags = collectDiagnostics("Hello, world!", "file:///test.tmpl")
	assert.Empty(t, diags)
}

func TestCollectDiagnostics_ValidTemplateBlocks(t *testing.T) {
	valid := []struct {
		name string
		text string
	}{
		{"dot field", `Hello {{ .Name }}`},
		{"variable assign", `{{ $x := .Value }}`},
		{"end keyword", `{{ end }}`},
		{"else keyword", `{{ else }}`},
		{"comment block", `{{/* a comment */}}`},
		{"trimmed braces", `{{- .Name -}}`},
		{"pipe expression", `{{ .Items | len }}`},
	}

	for _, tc := range valid {
		diags := collectDiagnostics(tc.text, "file:///test.tmpl")
		syntaxErrors := 0
		for _, d := range diags {
			if strings.Contains(d.Message, "unexpected token or unparseable") {
				syntaxErrors++
			}
		}
		assert.Zero(t, syntaxErrors, "unexpected syntax error in: %s\ndiags: %v", tc.text, diagMessages(diags))
	}
}

func TestCollectDiagnostics_InvalidTokens(t *testing.T) {
	text := `{{ random }}`
	diags := collectDiagnostics(text, "file:///test.tmpl")
	require.NotEmpty(t, diags)

	diag, ok := findDiagnosticContaining(diags, "unexpected token or unparseable")
	require.True(t, ok, "expected syntax error diagnostic, got: %v", diagMessages(diags))

	startIdx := strings.Index(text, "{{")
	endIdx := strings.LastIndex(text, "}}") + 2
	assertDiagnosticCoversTextRange(t, *diag, text, startIdx, endIdx)

	text = `{{ .Items[0] }}`
	diags = collectDiagnostics(text, "file:///test.tmpl")
	require.NotEmpty(t, diags)

	diag, ok = findDiagnosticContaining(diags, "unexpected token or unparseable")
	require.True(t, ok, "expected syntax error diagnostic, got: %v", diagMessages(diags))

	startIdx = strings.Index(text, "{{")
	endIdx = strings.LastIndex(text, "}}") + 2
	assertDiagnosticCoversTextRange(t, *diag, text, startIdx, endIdx)

	text = "{{ badOne }}\n{{ badTwo }}"
	diags = collectDiagnostics(text, "file:///test.tmpl")

	count := 0
	for _, d := range diags {
		if strings.Contains(d.Message, "unexpected token or unparseable") {
			count++
		}
	}
	assert.GreaterOrEqual(t, count, 2)
}

func TestCollectDiagnostics_MalformedMatch(t *testing.T) {
	text := "{{}}"
	assert.NotPanics(t, func() {
		collectDiagnostics(text, "file:///test.tmpl")
	})
}

func TestPublishDiagnostics_NilContext(t *testing.T) {
	assert.NotPanics(t, func() {
		publishDiagnostics(nil, "file:///test.tmpl", "{{ .Name }}")
	})
}
