package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// enableServer sets EnableServer: true for the duration of the test and restores the original config afterward.
func enableServer(t *testing.T) {
	t.Helper()
	original := GetConfig()
	setConfig(Config{EnableServer: true, Trace: original.Trace})
	t.Cleanup(func() { setConfig(original) })
}

func labelsFrom(t *testing.T, resp any) []string {
	t.Helper()
	list, ok := resp.(protocol.CompletionList)
	require.True(t, ok, "response should be a CompletionList")
	var labels []string
	for _, item := range list.Items {
		labels = append(labels, item.Label)
	}
	return labels
}

func TestCompletionLogic(t *testing.T) {
	t.Run("Scope-aware variable completion", func(t *testing.T) {
		enableServer(t)

		uri := "file:///scope.test.tmpl"
		content := `{{ $top := . }}
		{{ range $i, $v := .Items }}
		{{ $he
		{{ end }}
		{{ $late := . }}`

		store.Set(uri, content)
		t.Cleanup(func() { store.Remove(uri) })

		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position: protocol.Position{
					Line:      2, // 0-indexed: Line 2 is "		{{ $he"
					Character: 5, // After "{{ $he" (including the tab-space)
				},
			},
		}

		resp, err := completion(nil, params)
		require.NoError(t, err)

		labels := labelsFrom(t, resp)

		assert.Contains(t, labels, "$top", "top-level variable should be visible")
		assert.Contains(t, labels, "$i", "index variable should be visible inside range")
		assert.Contains(t, labels, "$v", "value variable should be visible inside range")
		assert.NotContains(
			t,
			labels,
			"$late",
			"$late should be out of scope (defined after cursor)",
		)
		assert.Contains(t, labels, "len", "global functions should be included")
		assert.Contains(t, labels, "range", "keywords should be included")
	})

	t.Run("Server disabled returns nil", func(t *testing.T) {
		original := GetConfig()
		setConfig(Config{EnableServer: false})
		t.Cleanup(func() { setConfig(original) })

		uri := "file:///disabled.tmpl"
		store.Set(uri, "{{ $x := . }}")
		t.Cleanup(func() { store.Remove(uri) })

		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 5},
			},
		}

		resp, err := completion(nil, params)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Document not found returns nil", func(t *testing.T) {
		enableServer(t)

		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.tmpl"},
				Position:     protocol.Position{Line: 0, Character: 5},
			},
		}

		resp, err := completion(nil, params)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Cursor outside template block returns nil", func(t *testing.T) {
		enableServer(t)

		uri := "file:///outside.tmpl"
		// Cursor is on line 1 ("hello"), which is plain text outside any {{ }}
		store.Set(uri, "{{ $x := . }}\nhello")
		t.Cleanup(func() { store.Remove(uri) })

		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 2},
			},
		}

		resp, err := completion(nil, params)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Variables from ended scope not visible", func(t *testing.T) {
		enableServer(t)

		uri := "file:///ended-scope.tmpl"
		// $inner is declared in the range block, which closes before the cursor on line 3
		content := "{{ $outer := . }}\n{{ range $inner := .Items }}\n{{ end }}\n{{ $here"
		store.Set(uri, content)
		t.Cleanup(func() { store.Remove(uri) })

		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 3, Character: 8},
			},
		}

		resp, err := completion(nil, params)
		require.NoError(t, err)

		labels := labelsFrom(t, resp)
		assert.Contains(t, labels, "$outer", "outer variable should still be visible")
		assert.NotContains(t, labels, "$inner", "$inner should not be visible after its scope ends")
	})

	t.Run("Nested range scopes all visible at cursor", func(t *testing.T) {
		enableServer(t)

		uri := "file:///nested-scope.tmpl"
		content := "{{ $a := . }}\n{{ range $i := .Items }}\n{{ range $j := .SubItems }}\n{{ $here"
		store.Set(uri, content)
		t.Cleanup(func() { store.Remove(uri) })

		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 3, Character: 8},
			},
		}

		resp, err := completion(nil, params)
		require.NoError(t, err)

		labels := labelsFrom(t, resp)
		assert.Contains(t, labels, "$a")
		assert.Contains(t, labels, "$i")
		assert.Contains(t, labels, "$j")
	})

	t.Run("If block variable visible inside block", func(t *testing.T) {
		enableServer(t)

		uri := "file:///if-scope.tmpl"
		content := "{{ $x := . }}\n{{ if $cond := .Flag }}\n{{ $here"
		store.Set(uri, content)
		t.Cleanup(func() { store.Remove(uri) })

		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 2, Character: 8},
			},
		}

		resp, err := completion(nil, params)
		require.NoError(t, err)

		labels := labelsFrom(t, resp)
		assert.Contains(t, labels, "$x")
		assert.Contains(t, labels, "$cond")
	})
}

func TestIsInsideTemplate(t *testing.T) {
	t.Run("inside unclosed action", func(t *testing.T) {
		assert.True(t, isInsideTemplate("{{ $x", 5))
	})

	t.Run("outside after closing braces", func(t *testing.T) {
		assert.False(t, isInsideTemplate("{{ $x }}", 8))
	})

	t.Run("inside second action after first is closed", func(t *testing.T) {
		assert.True(t, isInsideTemplate("{{ $x }}{{ $y", 13))
	})

	t.Run("comment block returns false", func(t *testing.T) {
		assert.False(t, isInsideTemplate("{{/* comment", 12))
	})

	t.Run("no template markers", func(t *testing.T) {
		assert.False(t, isInsideTemplate("plain text", 5))
	})

	t.Run("empty string", func(t *testing.T) {
		assert.False(t, isInsideTemplate("", 0))
	})

	t.Run("offset right after opening braces", func(t *testing.T) {
		assert.True(t, isInsideTemplate("{{ $x", 2))
	})
}

func TestGetWordAtOffset(t *testing.T) {
	t.Run("returns full variable name", func(t *testing.T) {
		assert.Equal(t, "$foo", getWordAtOffset("{{ $foo", 7))
	})

	t.Run("returns partial variable name", func(t *testing.T) {
		assert.Equal(t, "$fo", getWordAtOffset("{{ $foo", 6))
	})

	t.Run("returns empty string at space boundary", func(t *testing.T) {
		// offset 3 is right before '$', so no word chars precede it
		assert.Equal(t, "", getWordAtOffset("{{ $foo", 3))
	})

	t.Run("returns function name", func(t *testing.T) {
		assert.Equal(t, "len", getWordAtOffset("{{ len", 6))
	})

	t.Run("offset at zero returns empty", func(t *testing.T) {
		assert.Equal(t, "", getWordAtOffset("foo", 0))
	})
}

func TestExtractVariables(t *testing.T) {
	t.Run("simple top-level assignment", func(t *testing.T) {
		text := "{{ $x := . }}"
		assert.Contains(t, extractVariables(text, len(text)), "$x")
	})

	t.Run("range declaration with two variables", func(t *testing.T) {
		// cursor is inside the range block, before {{ end }}
		text := "{{ range $i, $v := .Items }}{{ end }}"
		vars := extractVariables(text, 28) // right at the start of {{ end }}
		assert.Contains(t, vars, "$i")
		assert.Contains(t, vars, "$v")
	})

	t.Run("variable not visible after scope ends", func(t *testing.T) {
		text := "{{ range $inner := .Items }}{{ end }}"
		assert.NotContains(t, extractVariables(text, len(text)), "$inner")
	})

	t.Run("outer variable visible inside inner scope", func(t *testing.T) {
		text := "{{ $outer := . }}{{ range $i := .Items }}"
		vars := extractVariables(text, len(text))
		assert.Contains(t, vars, "$outer")
		assert.Contains(t, vars, "$i")
	})

	t.Run("cursor before any declaration sees nothing", func(t *testing.T) {
		text := "{{ $x := . }}"
		assert.NotContains(t, extractVariables(text, 0), "$x")
	})

	t.Run("if block declares variable", func(t *testing.T) {
		text := "{{ if $cond := .Flag }}"
		assert.Contains(t, extractVariables(text, len(text)), "$cond")
	})
}

func TestDeclaredVars(t *testing.T) {
	t.Run("simple assignment", func(t *testing.T) {
		assert.Equal(t, []string{"$x"}, declaredVars("$x := ."))
	})

	t.Run("range with two variables", func(t *testing.T) {
		vars := declaredVars("range $i, $v := .Items")
		assert.Contains(t, vars, "$i")
		assert.Contains(t, vars, "$v")
	})

	t.Run("no declaration operator returns nil", func(t *testing.T) {
		assert.Nil(t, declaredVars("range .Items"))
	})

	t.Run("multiple variables on left-hand side", func(t *testing.T) {
		assert.Equal(t, []string{"$a", "$b"}, declaredVars("$a, $b := someFunc ."))
	})
}

func TestPositionToOffset(t *testing.T) {
	text := "hello\nworld"

	t.Run("line 0 character 0", func(t *testing.T) {
		assert.Equal(t, 0, positionToOffset(text, protocol.Position{Line: 0, Character: 0}))
	})

	t.Run("line 0 mid", func(t *testing.T) {
		assert.Equal(t, 3, positionToOffset(text, protocol.Position{Line: 0, Character: 3}))
	})

	t.Run("line 1 character 0", func(t *testing.T) {
		assert.Equal(t, 6, positionToOffset(text, protocol.Position{Line: 1, Character: 0}))
	})

	t.Run("line 1 mid", func(t *testing.T) {
		assert.Equal(t, 9, positionToOffset(text, protocol.Position{Line: 1, Character: 3}))
	})

	t.Run("position beyond end of text returns len", func(t *testing.T) {
		assert.Equal(
			t,
			len(text),
			positionToOffset(text, protocol.Position{Line: 10, Character: 0}),
		)
	})
}
