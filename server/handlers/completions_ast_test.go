package handlers

import (
	"testing"
	"text/template/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// suggestAt parses src, finds the node at offset, builds the path/context,
// and returns the suggestion labels. It is the core helper for all AST tests.
func suggestAt(t *testing.T, src string, offset int) []string {
	t.Helper()

	trees, err := parse.Parse("test", src, "", "", builtins())
	require.NoError(t, err, "template must parse without error")

	root := trees["test"].Root
	ctx := &Context{Vars: map[string]parse.Node{}}

	pos := parse.Pos(offset)
	cur := nodeFind(root, pos)
	ok := buildPath(root, cur, ctx)
	require.True(t, ok, "target node must be found in tree")

	var parent parse.Node
	if len(ctx.Path) >= 2 {
		parent = ctx.Path[len(ctx.Path)-2]
	}

	sChar := src[offset]
	items := suggest(cur, parent, ctx, sChar)

	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

// builtins returns the minimum map required by parse.Parse
func builtins() map[string]any {
	return map[string]any{
		"and": true, "call": true, "html": true, "index": true,
		"slice": true, "js": true, "len": true, "not": true, "or": true,
		"print": true, "printf": true, "println": true, "urlquery": true,
		"eq": true, "ne": true, "lt": true, "le": true, "gt": true, "ge": true,
	}
}

func offsetOf(t *testing.T, s, substr string, n int) int {
	t.Helper()
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			if count == n {
				return i
			}
			count++
		}
	}
	t.Fatalf("substring %q (occurrence %d) not found in %q", substr, n, s)
	return -1
}

// dot tests

func TestDotSuggestions(t *testing.T) {
	t.Run("bare dot action returns dot item", func(t *testing.T) {
		src := `{{.}}`
		// offset 2 is the '.' character
		labels := suggestAt(t, src, 2)
		assert.Contains(t, labels, ".")
	})

	t.Run("dot in if condition", func(t *testing.T) {
		src := `{{if .}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, ".", 0))
		assert.Contains(t, labels, ".")
		assert.NotContains(t, labels, "eq")
		assert.NotContains(t, labels, "len")
	})

	t.Run("dot in range pipeline", func(t *testing.T) {
		src := `{{range .}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, ".", 0))
		assert.Contains(t, labels, ".")
		assert.NotContains(t, labels, "eq")
		assert.NotContains(t, labels, "len")
	})

	t.Run("dot in with pipeline", func(t *testing.T) {
		src := `{{with .}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, ".", 0))
		assert.Contains(t, labels, ".")
		assert.NotContains(t, labels, "eq")
		assert.NotContains(t, labels, "len")
	})

	t.Run("sChar dot returns only dot item, not builtins", func(t *testing.T) {
		src := `{{.}}`
		labels := suggestAt(t, src, 2)
		assert.NotContains(t, labels, "eq")
		assert.NotContains(t, labels, "len")
	})
}

// variables

func TestVariableSuggestions(t *testing.T) {
	t.Run("sChar dollar returns vars without sigil", func(t *testing.T) {
		src := `{{$top := .}}{{$}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1))
		assert.Contains(t, labels, "top")
		assert.NotContains(t, labels, "$top", "should react on $")
	})

	t.Run("sChar non-dollar includes full $var label", func(t *testing.T) {
		src := `{{$top := .}}{{len .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "l", 0))
		assert.Contains(t, labels, "$top")
	})

	t.Run("variable declared before cursor is visible", func(t *testing.T) {
		src := `{{$x := .}}{{$x}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1))
		assert.Contains(t, labels, "x")
	})

	t.Run("variable declared after cursor is not visible", func(t *testing.T) {
		src := `{{$early := .}}{{$}}{{$late := .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1))
		assert.Contains(t, labels, "early")
		assert.NotContains(t, labels, "late")
		assert.NotContains(t, labels, "$late")
	})

	t.Run("range index and value variables visible inside body", func(t *testing.T) {
		src := `{{range $i, $v := .}}{{$}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 2))
		assert.Contains(t, labels, "i")
		assert.Contains(t, labels, "v")
	})

	t.Run("range variable not visible after end", func(t *testing.T) {
		src := `{{range $inner := .}}{{end}}{{$}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1))
		assert.NotContains(t, labels, "inner")
		assert.NotContains(t, labels, "$inner")
	})

	t.Run("outer variable visible inside nested range", func(t *testing.T) {
		src := `{{$outer := .}}{{range $i := .}}{{range $j := .}}{{$}}{{end}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 3))
		assert.Contains(t, labels, "outer")
		assert.Contains(t, labels, "i")
		assert.Contains(t, labels, "j")
	})

	t.Run("if condition variable visible inside block", func(t *testing.T) {
		src := `{{if $cond := .}}{{$}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1))
		assert.Contains(t, labels, "cond")
	})

	t.Run("with variable visible inside block", func(t *testing.T) {
		src := `{{with $w := .}}{{$}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1))
		assert.Contains(t, labels, "w")
	})
}

// builtins suggest

func TestBuiltinSuggestions(t *testing.T) {
	t.Run("builtins appear in general context", func(t *testing.T) {
		src := `{{len .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "l", 0))
		for _, fn := range []string{"len", "eq", "ne", "and", "or", "not", "print", "printf", "println", "index"} {
			assert.Contains(t, labels, fn, "builtin %q should be present", fn)
		}
	})

	t.Run("builtins not returned when sChar is dot", func(t *testing.T) {
		src := `{{.}}`
		labels := suggestAt(t, src, 2)
		assert.NotContains(t, labels, "len")
	})

	t.Run("builtins not returned when sChar is dollar", func(t *testing.T) {
		src := `{{$x := .}}{{$}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1))
		assert.NotContains(t, labels, "len")
		assert.NotContains(t, labels, "range")
	})
}

// pipe filtering

func TestPipeFilteredSuggestions(t *testing.T) {
	t.Run("after len pipe, only int-accepting functions suggested", func(t *testing.T) {
		src := `{{len . | eq . .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "e", 0))
		assert.Contains(t, labels, "eq")
		assert.Contains(t, labels, "lt")
		assert.Contains(t, labels, "print")
		// those should get filtered out
		assert.NotContains(t, labels, "html")
		assert.NotContains(t, labels, "js")
	})

	t.Run("after not pipe, only bool-accepting functions suggested", func(t *testing.T) {
		src := `{{not . | and . .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "a", 0))
		assert.Contains(t, labels, "and")
		assert.Contains(t, labels, "or")
		assert.Contains(t, labels, "not")
		assert.NotContains(t, labels, "len")
		assert.NotContains(t, labels, "html")
	})

	t.Run("after html pipe, only string-accepting functions suggested", func(t *testing.T) {
		src := `{{html . | len .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "l", 0))
		assert.Contains(t, labels, "len")
		assert.Contains(t, labels, "index")
		assert.NotContains(t, labels, "and")
		assert.NotContains(t, labels, "not")
	})

	t.Run("no preceding pipe returns full list", func(t *testing.T) {
		src := `{{len .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "l", 0))
		assert.Contains(t, labels, "len")
		assert.Contains(t, labels, "html")
		assert.Contains(t, labels, "and")
	})
}

// command node positions

func TestCommandNodePositionSuggestions(t *testing.T) {
	t.Run("first arg of command returns only builtins", func(t *testing.T) {
		src := `{{len .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "l", 0))
		assert.Contains(t, labels, "len")
		assert.Contains(t, labels, "eq")
	})

	t.Run("second arg of command returns all", func(t *testing.T) {
		src := `{{len .}}`
		labels := suggestAt(t, src, offsetOf(t, src, ".", 0))
		assert.Contains(t, labels, ".")
	})
}

// node find

func TestNodeFind(t *testing.T) {
	t.Run("finds dot node at its position", func(t *testing.T) {
		src := `{{.}}`
		trees, err := parse.Parse("test", src, "", "", builtins())
		require.NoError(t, err)
		root := trees["test"].Root

		node := nodeFind(root, parse.Pos(2))
		_, isDot := node.(*parse.DotNode)
		assert.True(t, isDot, "expected DotNode, got %T", node)
	})

	t.Run("finds identifier node", func(t *testing.T) {
		src := `{{len .}}`
		trees, err := parse.Parse("test", src, "", "", builtins())
		require.NoError(t, err)
		root := trees["test"].Root

		node := nodeFind(root, parse.Pos(2))
		id, isIdent := node.(*parse.IdentifierNode)
		assert.True(t, isIdent, "expected IdentifierNode, got %T", node)
		assert.Equal(t, "len", id.Ident)
	})

	t.Run("finds variable node", func(t *testing.T) {
		src := `{{$x := .}}{{$x}}`
		trees, err := parse.Parse("test", src, "", "", builtins())
		require.NoError(t, err)
		root := trees["test"].Root

		// second $
		node := nodeFind(root, parse.Pos(20))
		v, isVar := node.(*parse.VariableNode)
		assert.True(t, isVar, "expected VariableNode, got %T", node)
		assert.Equal(t, "$x", v.Ident[0])
	})
}

// buildPath correctness check

func TestBuildPathScope(t *testing.T) {
	t.Run("vars reset after if branch not taken", func(t *testing.T) {
		// $inner is declared inside the if-else; cursor is after {{end}},
		// so buildPath should NOT leave $inner in ctx.Vars.
		src := `{{if .}}{{$inner := .}}{{end}}{{.}}`
		trees, err := parse.Parse("test", src, "", "", builtins())
		require.NoError(t, err)
		root := trees["test"].Root

		ctx := &Context{Vars: map[string]parse.Node{}}
		pos := parse.Pos(offsetOf(t, src, ".", 2))
		cur := nodeFind(root, pos)
		buildPath(root, cur, ctx)

		_, hasInner := ctx.Vars["$inner"]
		assert.False(t, hasInner, "$inner should not leak out of if block")
	})

	t.Run("outer var always in scope", func(t *testing.T) {
		src := `{{$outer := .}}{{if .}}{{end}}{{.}}`
		trees, err := parse.Parse("test", src, "", "", builtins())
		require.NoError(t, err)
		root := trees["test"].Root

		ctx := &Context{Vars: map[string]parse.Node{}}
		pos := parse.Pos(offsetOf(t, src, ".", 2))
		cur := nodeFind(root, pos)
		buildPath(root, cur, ctx)

		_, hasOuter := ctx.Vars["$outer"]
		assert.True(t, hasOuter, "$outer must survive if block")
	})
}
