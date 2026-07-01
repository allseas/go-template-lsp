package handlers

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// helpers
func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) // #nosec G304 -- test helper, path is test-controlled
	require.NoError(t, err)
	return string(data)
}

func diagMessages(diags []protocol.Diagnostic) []string {
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	return msgs
}

func findDiagnosticContaining(
	diags []protocol.Diagnostic,
	substr string,
) (*protocol.Diagnostic, bool) {
	for i := range diags {
		if strings.Contains(diags[i].Message, substr) {
			return &diags[i], true
		}
	}
	return nil, false
}

func assertDiagnosticCoversTextRange(
	t *testing.T,
	diag protocol.Diagnostic,
	text string,
	startOffset, endOffset int,
) {
	t.Helper()

	wantStart := offsetToPosition(text, startOffset)
	wantEnd := offsetToPosition(text, endOffset)

	assert.Equal(t, wantStart.Line, diag.Range.Start.Line)
	assert.Equal(t, wantStart.Character, diag.Range.Start.Character)
	assert.Equal(t, wantEnd.Line, diag.Range.End.Line)
	assert.Equal(t, wantEnd.Character, diag.Range.End.Character)
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
		assert.Zero(
			t,
			syntaxErrors,
			"unexpected syntax error in: %s\ndiags: %v",
			tc.text,
			diagMessages(diags),
		)
	}
}

func TestCollectDiagnostics_InvalidTokens(t *testing.T) {
	text := `{{ random }}`
	diags := collectDiagnostics(text, "file:///test.tmpl")
	require.NotEmpty(t, diags)

	diag, ok := findDiagnosticContaining(diags, "undefined function")
	require.True(t, ok, "expected undefined function diagnostic, got: %v", diagMessages(diags))

	startIdx := strings.Index(text, "{{")
	endIdx := strings.LastIndex(text, "}}") + 2
	assertDiagnosticCoversTextRange(t, *diag, text, startIdx, endIdx)

	text = `{{ .Items[0] }}`
	diags = collectDiagnostics(text, "file:///test.tmpl")
	require.NotEmpty(t, diags)

	diag, ok = findDiagnosticContaining(diags, "unexpected")
	require.True(
		t,
		ok,
		"expected syntax error diagnostic, got: %v",
		diagMessages(diags),
	)

	startIdx = strings.Index(text, "{{")
	endIdx = strings.LastIndex(text, "}}") + 2
	assertDiagnosticCoversTextRange(t, *diag, text, startIdx, endIdx)

	text = "{{ badOne }}\n{{ badTwo }}"
	diags = collectDiagnostics(text, "file:///test.tmpl")

	count := 0
	for _, d := range diags {
		if strings.Contains(d.Message, "undefined function") {
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

func TestCollectDiagnostics_MalformedVariable(t *testing.T) {
	// Test for issue: malformed variable syntax like {{ $????? should not panic
	text := `SELECT
{{- if .Columns }}
	{{- range $i, $col := .Columns }}
		{{- if gt $i 0 }}, {{ end }}{{ $col }}
		{{ $?????
	{{- end }}
{{- else }}
	*
{{- end }}`
	assert.NotPanics(t, func() {
		collectDiagnostics(text, "file:///test.sql.tmpl")
	})
}

func TestPublishDiagnostics_NilContext(t *testing.T) {
	assert.NotPanics(t, func() {
		publishDiagnostics(nil, "file:///test.tmpl", "{{ .Name }}")
	})
}

func TestPublishDiagnostics_SendsDiagnostics(t *testing.T) {
	var notified bool
	var notifyMethod string
	var notifyParams *protocol.PublishDiagnosticsParams

	ctx := &glsp.Context{
		Notify: func(method string, params any) {
			notified = true
			notifyMethod = method
			if p, ok := params.(*protocol.PublishDiagnosticsParams); ok {
				notifyParams = p
			}
		},
	}

	publishDiagnostics(ctx, "file:///test.tmpl", "{{ random }}")

	assert.True(t, notified)
	assert.Equal(t, protocol.ServerTextDocumentPublishDiagnostics, notifyMethod)
	require.NotNil(t, notifyParams)
	assert.Equal(t, "file:///test.tmpl", notifyParams.URI)
	assert.NotEmpty(t, notifyParams.Diagnostics)
}

func TestPublishDiagnostics_UnknownDiagnosticFallback(t *testing.T) {
	var notifyParams *protocol.PublishDiagnosticsParams

	ctx := &glsp.Context{
		Notify: func(_ string, params any) {
			if p, ok := params.(*protocol.PublishDiagnosticsParams); ok {
				notifyParams = p
			}
		},
	}

	publishDiagnostics(ctx, "file:///test.tmpl", "")
	if notifyParams != nil && len(notifyParams.Diagnostics) > 0 {
		for _, d := range notifyParams.Diagnostics {
			if d.Message == "" {
				assert.Equal(t, "unknown diagnostic", d.Message)
			}
		}
	}
}

func TestCollectDiagnostics_EmptyAction(t *testing.T) {
	text := `{{ }}`
	diags := collectDiagnostics(text, "file:///test.tmpl")
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "missing value")
	// Range should cover the full {{ }} action.
	assert.Equal(t, u32(0), diags[0].Range.Start.Character)
	assert.Equal(t, u32(len(text)), diags[0].Range.End.Character)
}

func TestDiagnostics_OnlyDollar(t *testing.T) {
	text := `{{ $ }}`
	diags := collectDiagnostics(text, "file:///test.tmpl")

	assert.Empty(t, diags)
}

func TestCollectDiagnostics_Comments(t *testing.T) {
	comments := []struct {
		name string
		text string
	}{
		{"simple comment", `{{/* simple comment */}}`},
		{"comment with newlines", `{{/*
multi-line
comment
*/}}`},
		{"comment with trim left", `{{- /* comment */}}`},
		{"comment with trim both", `{{- /* comment */ -}}`},
		{"comment with template syntax", `{{/* {{ .Field }} */}}`},
		{"comment in range", `{{range .Items}}{{/* iteration comment */}}{{end}}`},
		{"comment before field", `{{/* comment */}}{{ .Name }}`},
		{"comment after field", `{{ .Name }}{{/* comment */}}`},
		{"comment with if", `{{if .Cond}}{{/* comment */}}{{end}}`},
		{"long comment", `{{- /* \n\n  # {{ .ProjectName }} \n {{- if .Tagline }} \n */ -}}`},
	}

	for _, tc := range comments {
		t.Run(tc.name, func(t *testing.T) {
			diags := collectDiagnostics(tc.text, "file:///test.tmpl")

			assert.Empty(t, diags)
		})
	}
}

func TestCollectDiagnostics_IncorrectCommentSyntax(t *testing.T) {
	text := `{{/* unclosed comment */   dyguayudsgyaui   }}`
	diags := collectDiagnostics(text, "file:///test.tmpl")
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "comment ends before closing delimiter")
}

// TestCollectDiagnostics_MultiDefines verifies that diagnostics are collected
// across every {{define}} tree in a single document and that a syntax error
// in one define does not suppress diagnostics in another.
func TestCollectDiagnostics_MultiDefines(t *testing.T) {
	t.Run("clean multi-define document yields no diagnostics", func(t *testing.T) {
		src := "{{- define \"A\" -}}\n" +
			"{{- /*gotype: text-template-server/src/model.Order*/ -}}\n" +
			"A: {{ .CustomerName }}\n" +
			"{{- end -}}\n" +
			"{{- define \"B\" -}}\n" +
			"{{ $x := . }}\nB: {{ $x }}\n" +
			"{{- end -}}\n"

		diags := collectDiagnostics(src, "file:///diag-multi-clean.tmpl")
		assert.Empty(t, diags, "expected no diagnostics, got: %v", diagMessages(diags))
	})

	t.Run("error in one define is reported per-tree", func(t *testing.T) {
		src := "{{- define \"A\" -}}\n" +
			"A: {{ random }}\n" + // unsupported function
			"{{- end -}}\n" +
			"{{- define \"B\" -}}\n" +
			"B: {{ .Name }}\n" +
			"{{- end -}}\n"

		diags := collectDiagnostics(src, "file:///diag-multi-err.tmpl")
		_, ok := findDiagnosticContaining(diags, "undefined function")
		require.True(
			t,
			ok,
			"expected undefined-function diagnostic in define A, got: %v",
			diagMessages(diags),
		)
	})
}

// TestCollectDiagnostics_TemplateArgTypeCheck verifies that a diagnostic is
// emitted when a {{template}} call passes an argument whose type doesn't match
// the gotype hint declared on the target {{define}} block, and that no
// diagnostic is emitted for correct or untyped calls.
func TestCollectDiagnostics_TemplateArgTypeCheck(t *testing.T) {
	const resourceDir = "../../test/resources/template-arg-typechecking"

	t.Cleanup(func() { WorkspaceRoot = "" })
	WorkspaceRoot = resourceDir

	uri := func(name string) string { return "file:///" + name }

	t.Run("wrong type emits diagnostic", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/wrong-type-call.tmpl")
		store.Set(uri("wrong-type-call.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("wrong-type-call.tmpl")) })

		diags := collectDiagnostics(src, uri("wrong-type-call.tmpl"))
		_, ok := findDiagnosticContaining(diags, "person-card")
		require.True(
			t,
			ok,
			"expected type-mismatch diagnostic for person-card call, got: %v",
			diagMessages(diags),
		)
	})

	t.Run("correct type yields no type-mismatch diagnostic", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/correct-call.tmpl")
		store.Set(uri("correct-call.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("correct-call.tmpl")) })

		diags := collectDiagnostics(src, uri("correct-call.tmpl"))
		_, ok := findDiagnosticContaining(diags, "person-card")
		require.False(
			t,
			ok,
			"expected no type-mismatch diagnostic for correct call, got: %v",
			diagMessages(diags),
		)
	})

	t.Run("no-arg call yields no type-mismatch diagnostic", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/no-arg-call.tmpl")
		store.Set(uri("no-arg-call.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("no-arg-call.tmpl")) })

		diags := collectDiagnostics(src, uri("no-arg-call.tmpl"))
		_, ok := findDiagnosticContaining(diags, "person-card")
		require.False(t, ok, "expected no diagnostic for no-arg call, got: %v", diagMessages(diags))
	})

	t.Run("unknown target yields no type-mismatch diagnostic", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/unknown-target.tmpl")
		store.Set(uri("unknown-target.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("unknown-target.tmpl")) })

		diags := collectDiagnostics(src, uri("unknown-target.tmpl"))
		_, ok := findDiagnosticContaining(diags, "expects argument of type")
		require.False(
			t,
			ok,
			"expected no type-mismatch diagnostic for untyped target, got: %v",
			diagMessages(diags),
		)
	})

	t.Run("range rebinds dot to element type for template call", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/range-template-call.tmpl")
		store.Set(uri("range-template-call.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("range-template-call.tmpl")) })

		diags := collectDiagnostics(src, uri("range-template-call.tmpl"))
		_, ok := findDiagnosticContaining(diags, "expects argument of type")
		require.False(
			t,
			ok,
			"expected no type-mismatch diagnostic when {{template \"item\" .}} runs inside {{range .Items}}, got: %v",
			diagMessages(diags),
		)
	})

	t.Run("template call outside range emits type-mismatch diagnostic", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/direct-template-call.tmpl")
		store.Set(uri("direct-template-call.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("direct-template-call.tmpl")) })

		diags := collectDiagnostics(src, uri("direct-template-call.tmpl"))
		_, ok := findDiagnosticContaining(diags, "expects argument of type")
		require.True(
			t,
			ok,
			"expected type-mismatch diagnostic when passing Order to template \"item\" outside a range, got: %v",
			diagMessages(diags),
		)
	})
}

func TestCollectDiagnostics_HintLoadFailure(t *testing.T) {
	fileURI := "file:///hint-load-failure.tmpl"
	t.Cleanup(func() { store.Remove(fileURI) })

	t.Run("unresolvable root hint emits warning on comment", func(t *testing.T) {
		src := "{{/*gotype: nonexistent/pkg.Type*/}}\n{{ .Name }}\n"
		store.Set(fileURI, src)

		diags := collectDiagnostics(src, fileURI)
		diag, ok := findDiagnosticContaining(diags, "could not load type")
		require.True(t, ok, "expected hint-load-failure diagnostic, got: %v", diagMessages(diags))

		assert.Equal(t, uint32(0), diag.Range.Start.Line, "diagnostic should be on line 0")
		require.NotNil(t, diag.Severity)
		assert.Equal(t, protocol.DiagnosticSeverityWarning, *diag.Severity)
	})

	t.Run("valid hint emits no hint-load-failure diagnostic", func(t *testing.T) {
		const resourceDir = "../../test/resources/definition-tests-server"
		t.Cleanup(func() { WorkspaceRoot = "" })
		WorkspaceRoot = resourceDir

		src := "{{/*gotype: text-template-server/src/model.Order*/}}\n{{ .CustomerName }}\n"
		store.Set(fileURI, src)

		diags := collectDiagnostics(src, fileURI)
		_, ok := findDiagnosticContaining(diags, "could not load type")
		require.False(
			t,
			ok,
			"expected no hint-load-failure diagnostic for valid hint, got: %v",
			diagMessages(diags),
		)
	})

	t.Run("unresolvable define-block hint emits warning on its comment", func(t *testing.T) {
		src := "{{- define \"myblock\" -}}\n{{/*gotype: nonexistent/pkg.Type*/}}\n{{ .Name }}\n{{- end -}}\n"
		store.Set(fileURI, src)

		diags := collectDiagnostics(src, fileURI)
		diag, ok := findDiagnosticContaining(diags, "could not load type")
		require.True(
			t,
			ok,
			"expected hint-load-failure diagnostic for define block, got: %v",
			diagMessages(diags),
		)

		assert.Equal(
			t,
			uint32(1),
			diag.Range.Start.Line,
			"diagnostic should be on the gotype comment line",
		)
	})
}

// TestCollectDiagnostics_DictHint verifies that dict-shaped gotype hints
// resolve end-to-end through the handler pipeline: valid key access produces
// no diagnostics, unknown keys are flagged, dict shape survives variable
// binding and with-blocks, unresolvable entries emit a hint-load-failure, and
// mixed struct + dict hints in the same document both load successfully.
func TestCollectDiagnostics_DictHint(t *testing.T) {
	const resourceDir = "../../test/resources/dict-typehints"

	t.Cleanup(func() { WorkspaceRoot = "" })
	WorkspaceRoot = resourceDir

	uri := func(name string) string { return "file:///" + name }

	t.Run("valid access resolves through dict", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/dict-valid-access.tmpl")
		store.Set(uri("dict-valid-access.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("dict-valid-access.tmpl")) })

		diags := collectDiagnostics(src, uri("dict-valid-access.tmpl"))
		_, loadFail := findDiagnosticContaining(diags, "could not load type")
		require.False(
			t,
			loadFail,
			"expected no hint-load-failure diagnostic, got: %v",
			diagMessages(diags),
		)
		_, invalid := findDiagnosticContaining(diags, "invalid field")
		require.False(t, invalid, "expected no invalid-field diagnostic, got: %v", diagMessages(diags))
		_, unknownKey := findDiagnosticContaining(diags, "known keys")
		require.False(t, unknownKey, "expected no unknown-key diagnostic, got: %v", diagMessages(diags))
	})

	t.Run("unknown key emits invalid-field diagnostic", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/dict-unknown-key.tmpl")
		store.Set(uri("dict-unknown-key.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("dict-unknown-key.tmpl")) })

		diags := collectDiagnostics(src, uri("dict-unknown-key.tmpl"))
		diag, ok := findDiagnosticContaining(diags, "Unknown")
		require.True(t, ok, "expected diagnostic for Unknown key, got: %v", diagMessages(diags))
		assert.Contains(t, diag.Message, "known keys")
		assert.Contains(t, diag.Message, "Order")
	})

	t.Run("variable binding preserves dict shape", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/dict-var-binding.tmpl")
		store.Set(uri("dict-var-binding.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("dict-var-binding.tmpl")) })

		diags := collectDiagnostics(src, uri("dict-var-binding.tmpl"))
		_, loadFail := findDiagnosticContaining(diags, "could not load type")
		require.False(
			t,
			loadFail,
			"expected no hint-load-failure diagnostic, got: %v",
			diagMessages(diags),
		)
		_, invalid := findDiagnosticContaining(diags, "invalid field")
		require.False(
			t,
			invalid,
			"expected $d.Order.CustomerName to resolve without errors, got: %v",
			diagMessages(diags),
		)
	})

	t.Run("with block preserves dict shape", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/dict-with-block.tmpl")
		store.Set(uri("dict-with-block.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("dict-with-block.tmpl")) })

		diags := collectDiagnostics(src, uri("dict-with-block.tmpl"))
		_, loadFail := findDiagnosticContaining(diags, "could not load type")
		require.False(
			t,
			loadFail,
			"expected no hint-load-failure diagnostic, got: %v",
			diagMessages(diags),
		)
		_, withErr := findDiagnosticContaining(diags, "cannot use type")
		require.False(
			t,
			withErr,
			"expected with-block to accept dict, got: %v",
			diagMessages(diags),
		)
		_, invalid := findDiagnosticContaining(diags, "invalid field")
		require.False(
			t,
			invalid,
			"expected .Order.CustomerName to resolve inside with, got: %v",
			diagMessages(diags),
		)
	})

	t.Run("unresolvable dict entry emits hint-load-failure", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/dict-unresolvable-entry.tmpl")
		store.Set(uri("dict-unresolvable-entry.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("dict-unresolvable-entry.tmpl")) })

		diags := collectDiagnostics(src, uri("dict-unresolvable-entry.tmpl"))
		diag, ok := findDiagnosticContaining(diags, "could not load type")
		require.True(t, ok, "expected hint-load-failure diagnostic, got: %v", diagMessages(diags))
		assert.Contains(t, diag.Message, "Bad", "diagnostic should name the failing key")
		assert.Equal(t, uint32(0), diag.Range.Start.Line, "diagnostic should sit on the hint comment")
		require.NotNil(t, diag.Severity)
		assert.Equal(t, protocol.DiagnosticSeverityWarning, *diag.Severity)
	})

	t.Run("struct hint alongside dict hint still loads", func(t *testing.T) {
		src := readTestFile(t, resourceDir+"/dict-and-struct.tmpl")
		store.Set(uri("dict-and-struct.tmpl"), src)
		t.Cleanup(func() { store.Remove(uri("dict-and-struct.tmpl")) })

		diags := collectDiagnostics(src, uri("dict-and-struct.tmpl"))
		_, ok := findDiagnosticContaining(diags, "could not load type")
		require.False(
			t,
			ok,
			"expected both hints to load; got: %v",
			diagMessages(diags),
		)
	})
}

func TestCollectDiagnostics_IncorrectWith(t *testing.T) {
	text := `{{ with "string" }}Hello{{ end }}`
	diags := collectDiagnostics(text, "file:///test.tmpl")
	require.NotEmpty(t, diags)
	_, ok := findDiagnosticContaining(diags, "cannot use type string in with")
	require.True(t, ok, "expected type error diagnostic for with, got: %v", diagMessages(diags))
}
