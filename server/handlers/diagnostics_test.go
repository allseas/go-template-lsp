package handlers

import (
	"math"
	"os"
	"strings"
	"testing"
	parse "text-template-parser"

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

	diag, ok := findDiagnosticContaining(diags, "unsupported function or unregistered command")
	require.True(t, ok, "expected unsupported function diagnostic, got: %v", diagMessages(diags))

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
		if strings.Contains(d.Message, "unsupported function") {
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

// ai generated below
func TestExtractBranchNodes(t *testing.T) {
	ifPipe, ifList, ifElse := &parse.PipeNode{}, &parse.ListNode{}, &parse.ListNode{}
	p, l, e := extractBranchNodes(&parse.IfNode{
		BranchNode: parse.BranchNode{
			Pipe:     ifPipe,
			List:     ifList,
			ElseList: ifElse,
		},
	})
	assert.Equal(t, ifPipe, p)
	assert.Equal(t, ifList, l)
	assert.Equal(t, ifElse, e)

	rangePipe, rangeList, rangeElse := &parse.PipeNode{}, &parse.ListNode{}, &parse.ListNode{}
	p, l, e = extractBranchNodes(&parse.RangeNode{
		BranchNode: parse.BranchNode{
			Pipe:     rangePipe,
			List:     rangeList,
			ElseList: rangeElse,
		},
	})
	assert.Equal(t, rangePipe, p)
	assert.Equal(t, rangeList, l)
	assert.Equal(t, rangeElse, e)

	withPipe, withList, withElse := &parse.PipeNode{}, &parse.ListNode{}, &parse.ListNode{}
	p, l, e = extractBranchNodes(&parse.WithNode{
		BranchNode: parse.BranchNode{
			Pipe:     withPipe,
			List:     withList,
			ElseList: withElse,
		},
	})
	assert.Equal(t, withPipe, p)
	assert.Equal(t, withList, l)
	assert.Equal(t, withElse, e)

	p, l, e = extractBranchNodes(&parse.TextNode{})
	assert.Nil(t, p)
	assert.Nil(t, l)
	assert.Nil(t, e)
}

func TestCollectDeclarations(t *testing.T) {
	ctx := &Context{Vars: map[string]parse.Node{}}
	collectDeclarations(nil, ctx)

	pipeWithNil := &parse.PipeNode{Decl: []*parse.VariableNode{nil}}
	collectDeclarations(pipeWithNil, ctx)

	declVar := &parse.VariableNode{Ident: []string{"$newVar"}}
	pipe := &parse.PipeNode{Decl: []*parse.VariableNode{declVar}}

	collectDeclarations(pipe, ctx)
	assert.NotNil(t, ctx.Vars["$newVar"])

	// Duplicate declarations are reported by the type analysis, not here, so a
	// second declaration registers the variable without emitting diagnostics.
	collectDeclarations(pipe, ctx)
	assert.NotNil(t, ctx.Vars["$newVar"])
}

func TestCheckPipeUsage(t *testing.T) {
	ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
	assert.Nil(t, checkPipeUsage(nil, "", ctx))

	ctx.Vars["$defined"] = &parse.PipeNode{}
	vnode := &parse.VariableNode{Ident: []string{"$defined"}}
	cmd := &parse.CommandNode{Args: []parse.Node{vnode}}
	pipe := &parse.PipeNode{Cmds: []*parse.CommandNode{cmd}}
	assert.Empty(t, checkPipeUsage(pipe, "{{ $defined }}", ctx))

	vnodeUndef := &parse.VariableNode{Ident: []string{"$undef"}}
	cmdUndef := &parse.CommandNode{Args: []parse.Node{vnodeUndef}}
	pipeUndef := &parse.PipeNode{Cmds: []*parse.CommandNode{cmdUndef}}
	diags := checkPipeUsage(pipeUndef, "{{ $undef }}", ctx)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "undeclared variable: $undef")
}

func TestDeclareNode(t *testing.T) {
	actVar := &parse.VariableNode{Ident: []string{"$act"}}
	rngVar := &parse.VariableNode{Ident: []string{"$rng"}}
	ifVar := &parse.VariableNode{Ident: []string{"$if"}}

	tests := []struct {
		name    string
		node    parse.Node
		text    string
		varName string
	}{
		{
			name:    "action node",
			node:    &parse.ActionNode{Pipe: &parse.PipeNode{Decl: []*parse.VariableNode{actVar}}},
			text:    "{{ $act := . }}",
			varName: "$act",
		},
		{
			name: "range node",
			node: &parse.RangeNode{
				BranchNode: parse.BranchNode{
					Pipe: &parse.PipeNode{Decl: []*parse.VariableNode{rngVar}},
				},
			},
			text:    "{{ range $rng := . }}",
			varName: "$rng",
		},
		{
			name: "if node",
			node: &parse.IfNode{
				BranchNode: parse.BranchNode{
					Pipe: &parse.PipeNode{Decl: []*parse.VariableNode{ifVar}},
				},
			},
			text:    "{{ if $if := . }}",
			varName: "$if",
		},
	}

	for _, tc := range tests {
		ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
		declareNode(tc.node, tc.text, ctx)
		assert.NotNil(t, ctx.Vars[tc.varName], tc.name)
	}
}

func TestAnalyzeNode_PipeWrappers(t *testing.T) {
	ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
	vnode := &parse.VariableNode{Ident: []string{"$undef"}}
	cmd := &parse.CommandNode{Args: []parse.Node{vnode}}
	pipe := &parse.PipeNode{Cmds: []*parse.CommandNode{cmd}}

	tests := []struct {
		name string
		node parse.Node
		text string
	}{
		{
			name: "action block",
			node: &parse.ActionNode{Pipe: pipe},
			text: "{{ $undef }}",
		},
		{
			name: "range block",
			node: &parse.RangeNode{BranchNode: parse.BranchNode{Pipe: pipe}},
			text: "{{ range $undef }}",
		},
		{
			name: "if block",
			node: &parse.IfNode{BranchNode: parse.BranchNode{Pipe: pipe}},
			text: "{{ if $undef }}",
		},
		{
			name: "with block",
			node: &parse.WithNode{BranchNode: parse.BranchNode{Pipe: pipe}},
			text: "{{ with $undef }}",
		},
	}

	for _, tc := range tests {
		diags := analyzeNode(tc.node, tc.text, ctx)
		require.Len(t, diags, 1, tc.name)
		assert.Contains(t, diags[0].Message, "undeclared variable: $undef", tc.name)
	}
}

func TestAnalyzeNode_CommandNode(t *testing.T) {
	ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
	ident := &parse.IdentifierNode{Ident: "unregisteredCommand"}
	cmd := &parse.CommandNode{Args: []parse.Node{ident}}

	diags := analyzeNode(cmd, "{{ unregisteredCommand }}", ctx)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "unsupported function or unregistered command")
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

func TestCollectDeclarations_RootIdent(t *testing.T) {
	ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
	declVar := &parse.VariableNode{Ident: []string{"$"}}
	pipe := &parse.PipeNode{Decl: []*parse.VariableNode{declVar}}

	collectDeclarations(pipe, ctx)
	// "$" has no name after trimming the prefix, so it is not registered.
	assert.Nil(t, ctx.Vars["$"])
}

func TestCollectDeclarations_Assignment(t *testing.T) {
	ctx := &Context{Vars: map[string]parse.Node{"$": nil, "$x": &parse.PipeNode{}}}
	declVar := &parse.VariableNode{Ident: []string{"$x"}}
	pipe := &parse.PipeNode{
		Decl:     []*parse.VariableNode{declVar},
		IsAssign: true,
	}
	collectDeclarations(pipe, ctx)
	assert.Equal(t, pipe, ctx.Vars["$x"])
}

func TestCheckPipeUsage_SpecialVariables(t *testing.T) {
	ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
	vnodeRoot := &parse.VariableNode{Ident: []string{"$"}}
	cmdRoot := &parse.CommandNode{Args: []parse.Node{vnodeRoot}}
	pipeRoot := &parse.PipeNode{Cmds: []*parse.CommandNode{cmdRoot}}
	assert.Empty(t, checkPipeUsage(pipeRoot, "{{ $ }}", ctx))
	vnodeEmpty := &parse.VariableNode{Ident: []string{""}}
	cmdEmpty := &parse.CommandNode{Args: []parse.Node{vnodeEmpty}}
	pipeEmpty := &parse.PipeNode{Cmds: []*parse.CommandNode{cmdEmpty}}
	assert.Empty(t, checkPipeUsage(pipeEmpty, "{{}}", ctx))
}

func TestAnalyzeNode_UndefinedNodeEmptyName(t *testing.T) {
	ctx := &Context{Vars: map[string]parse.Node{"$": nil}}
	undefNode := &parse.UndefinedNode{}
	diags := analyzeNode(undefNode, "{{ }}", ctx)
	assert.Empty(t, diags)
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
		{"comment with trim left", `{{- /* comment */ }}`},
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
		_, ok := findDiagnosticContaining(diags, "unsupported function or unregistered command")
		require.True(
			t,
			ok,
			"expected unsupported-function diagnostic in define A, got: %v",
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
