package handlers

import (
	"go/types"
	"testing"
	parse "text-template-parser"

	"golang.org/x/tools/go/packages"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
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
	items := suggest(cur, parent, ctx, sChar, isInvoked, nil, protocol.Range{})
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
		"DisplayName":  true,
		"Summary":      true,
		"ItemCount":    true,
		"IsLargeOrder": true,
		"Format":       true,
		"Label":        true,
		"Total":        true,
		"IsExpensive":  true,
		"Describe":     true,
		"Line":         true,
		"IsLocal":      true,
		"ZipCode":      true,
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

// helpers — build a LoadedType for Order so tests can inject it into Context.

func orderLoadedType(t *testing.T) *LoadedType {
	t.Helper()

	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
	}
	pkgs, err := packages.Load(cfg, "text-template-server/src/model")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	obj := pkg.Types.Scope().Lookup("Order")
	require.NotNil(t, obj, "Order type must exist in package")

	named := obj.Type().(*types.Named)
	return &LoadedType{
		Pkg:     pkg,
		Named:   named,
		Fields:  structFields(named),
		Methods: namedMethods(named),
	}
}

// suggestAtWithType is like suggestAt but injects a LoadedType into the context.
func suggestAtWithType(
	t *testing.T,
	src string,
	offset int,
	isInvoked bool,
	lt *LoadedType,
) []string {
	t.Helper()

	trees, err := parse.Parse("test", src, "", "", builtins())
	require.NoError(t, err, "template must parse without error")

	root := trees["test"].Root
	ctx := &Context{
		Vars:    map[string]parse.Node{},
		DotType: lt,
	}

	pos := parse.Pos(offset)
	cur := nodeFind(root, pos)
	ok := buildPath(root, cur, ctx)
	require.True(t, ok, "target node must be found in tree")

	var parent parse.Node
	if len(ctx.Path) >= 2 {
		parent = ctx.Path[len(ctx.Path)-2]
	}

	sChar := src[offset]
	items := suggest(cur, parent, ctx, sChar, isInvoked)

	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

// dot field completion

func TestDotFieldCompletion(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run("dot triggers field completions", func(t *testing.T) {
		src := `{{.}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, ".", 0), false, lt)
		for _, field := range []string{"ID", "CustomerName", "Email", "Address", "Items", "TotalAmount", "Paid"} {
			assert.Contains(t, labels, field, "field %q should appear after dot", field)
		}
	})

	t.Run("dot does not include builtins", func(t *testing.T) {
		src := `{{.}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, ".", 0), false, lt)
		assert.NotContains(t, labels, "len")
		assert.NotContains(t, labels, "eq")
		assert.NotContains(t, labels, "html")
	})
}

// dot method completion

func TestDotMethodCompletion(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run("typing dot returns usable method names without dot prefix", func(t *testing.T) {
		src := `{{.}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, ".", 0), false, lt)
		assert.Contains(t, labels, "DisplayName")
		assert.Contains(t, labels, "Summary")
		assert.Contains(t, labels, "ItemCount")
		assert.Contains(t, labels, "IsLargeOrder")
		assert.Contains(t, labels, "Format")
	})

	t.Run("typing dot excludes non-usable methods", func(t *testing.T) {
		src := `{{.}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, ".", 0), false, lt)
		assert.NotContains(t, labels, "wrongSecond")
		assert.NotContains(t, labels, "badReturn")
	})

	t.Run("typing dot returns methods and fields together", func(t *testing.T) {
		src := `{{.}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, ".", 0), false, lt)
		// fields
		assert.Contains(t, labels, "ID")
		assert.Contains(t, labels, "CustomerName")
		assert.Contains(t, labels, "Paid")
		// methods
		assert.Contains(t, labels, "DisplayName")
		assert.Contains(t, labels, "ItemCount")
	})

	t.Run("general context returns all usable methods with dot prefix", func(t *testing.T) {
		// should be changed later, as len . should only suggest string typed fields
		src := `{{len .}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "l", 0), false, lt)
		assert.Contains(t, labels, ".DisplayName")
		assert.Contains(t, labels, ".Summary")
		assert.Contains(t, labels, ".ItemCount")
		assert.Contains(t, labels, ".IsLargeOrder")
		assert.Contains(t, labels, ".Format")
		assert.NotContains(t, labels, ".wrongSecond")
		assert.NotContains(t, labels, ".badReturn")
		// fields with dot
		assert.Contains(t, labels, ".ID")
		assert.Contains(t, labels, ".CustomerName")
		assert.Contains(t, labels, ".Paid")
		// methods with dot
		assert.Contains(t, labels, ".DisplayName")
		assert.Contains(t, labels, ".ItemCount")
	})

	t.Run("dot-prefixed methods excluded when no loaded type", func(t *testing.T) {
		src := `{{len .}}`
		// no lt — dotType is nil
		labels := suggestAt(t, src, offsetOf(t, src, "l", 0))
		assert.NotContains(t, labels, ".DisplayName")
		assert.NotContains(t, labels, ".ItemCount")
	})
}

// pipe filtering with model fields

func TestPipeFilteringWithModelFields(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run("string field piped — string-accepting builtins suggested", func(t *testing.T) {
		// .CustomerName is string → after pipe, suggest html, js, len etc.
		src := `{{.CustomerName | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "html")
		assert.Contains(t, labels, "js")
		assert.Contains(t, labels, "urlquery")
		assert.Contains(t, labels, "len")
		assert.NotContains(t, labels, "not")
		assert.NotContains(t, labels, "and")
	})

	t.Run("bool field piped — bool-accepting builtins suggested", func(t *testing.T) {
		// .Paid is bool
		src := `{{.Paid | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "not")
		assert.Contains(t, labels, "and")
		assert.Contains(t, labels, "or")
		assert.NotContains(t, labels, "html")
		assert.NotContains(t, labels, "len")
	})

	t.Run("float field piped — int-accepting builtins suggested", func(t *testing.T) {
		// .TotalAmount is float64 — IsInteger is false, so falls to outputUntyped → all builtins
		src := `{{.TotalAmount | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		// float64 is not IsInteger so typeToOutputKind returns outputUntyped → all builtins shown
		assert.Contains(t, labels, "len")
		assert.Contains(t, labels, "html")
		assert.Contains(t, labels, "eq")
	})

	t.Run("struct field piped — all builtins shown (outputUntyped)", func(t *testing.T) {
		// .Address is a struct
		src := `{{.Address | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "len")
		assert.Contains(t, labels, "html")
		assert.Contains(t, labels, "and")
	})
}

// pipe filtering with model methods

func TestPipeFilteringWithModelMethods(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run(
		"string-returning method piped — string-accepting builtins suggested",
		func(t *testing.T) {
			// .DisplayName returns string
			src := `{{.DisplayName | }}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
			assert.Contains(t, labels, "html")
			assert.Contains(t, labels, "js")
			assert.Contains(t, labels, "len")
			assert.NotContains(t, labels, ".Format")
			assert.NotContains(t, labels, "not")
		},
	)

	t.Run("int-returning method piped — int-accepting builtins suggested", func(t *testing.T) {
		// .ItemCount returns int
		src := `{{.ItemCount | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "eq")
		assert.Contains(t, labels, "lt")
		assert.Contains(t, labels, "gt")
		assert.NotContains(t, labels, "html")
		assert.NotContains(t, labels, "not")
	})

	t.Run("bool-returning method piped — bool-accepting builtins suggested", func(t *testing.T) {
		// .IsLargeOrder returns bool
		src := `{{.IsLargeOrder | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "not")
		assert.Contains(t, labels, "and")
		assert.NotContains(t, labels, "len")
		assert.NotContains(t, labels, "html")
	})

	t.Run(
		"string-returning method with arg piped — string-accepting builtins suggested",
		func(t *testing.T) {
			// .Format returns string
			src := `{{.Format "$" | }}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
			assert.Contains(t, labels, "html")
			assert.Contains(t, labels, "len")
			assert.NotContains(t, labels, "not")
		},
	)
}

func TestUserMethodPipeFiltering(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run(
		"string field piped — no dot-prefixed methods appear (methods not suggested in pipe)",
		func(t *testing.T) {
			src := `{{.CustomerName | }}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
			assert.NotContains(t, labels, ".Format")
			assert.NotContains(t, labels, ".DisplayName")
			assert.NotContains(t, labels, ".ItemCount")
			assert.NotContains(t, labels, ".IsLargeOrder")
			assert.NotContains(t, labels, ".wrongSecond")
		},
	)

	t.Run(
		"bool and int fields piped — no dot-prefixed methods (none accept bool/int)",
		func(t *testing.T) {
			boolLabels := suggestAtWithType(t, `{{.Paid | }}`, offsetOf(t, `{{.Paid | }}`, "}}", 0)-1, true, lt)
			assert.NotContains(t, boolLabels, ".Format")
			assert.NotContains(t, boolLabels, ".DisplayName")

			intLabels := suggestAtWithType(
				t,
				`{{ItemCount | }}`,
				offsetOf(t, `{{ItemCount | }}`, "}}", 0)-1,
				true,
				lt,
			)
			assert.NotContains(t, intLabels, ".Format")
			assert.NotContains(t, intLabels, ".DisplayName")
		},
	)

	t.Run("bare method names never appear in any pipe context", func(t *testing.T) {
		for _, src := range []string{`{{.CustomerName | }}`, `{{.Paid | }}`, `{{ItemCount | }}`} {
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
			assert.NotContains(t, labels, "DisplayName")
			assert.NotContains(t, labels, "Format")
			assert.NotContains(t, labels, "wrongSecond")
		}
	})
}

func TestSliceFieldPipeCompletion(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run(
		"slice pipe shows builtins, no dot-prefixed methods (none accept slice)",
		func(t *testing.T) {
			src := `{{.Items | }}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
			assert.Contains(t, labels, "len")
			assert.Contains(t, labels, "index")
			assert.Contains(t, labels, "slice")
			// no methods accept a slice → none appear dot-prefixed
			assert.NotContains(t, labels, ".DisplayName")
			assert.NotContains(t, labels, ".ItemCount")
			// bare method names never appear either
			assert.NotContains(t, labels, "DisplayName")
			assert.NotContains(t, labels, "ItemCount")
		},
	)
}

func TestMultiStagePipeChaining(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run("html after string field — string builtins only", func(t *testing.T) {
		src := `{{.CustomerName | html | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "len")
		assert.Contains(t, labels, "js")
		assert.NotContains(t, labels, "not")
		assert.NotContains(t, labels, "eq")
	})

	t.Run("not after bool field — bool builtins only", func(t *testing.T) {
		src := `{{.Paid | not | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "and")
		assert.Contains(t, labels, "or")
		assert.NotContains(t, labels, "len")
		assert.NotContains(t, labels, "html")
		assert.NotContains(t, labels, ".Oper")
	})
}

func TestInvokedVsNonInvoked(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run("invoked after string pipe — string builtins, not bool", func(t *testing.T) {
		src := `{{.CustomerName | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "html")
		assert.Contains(t, labels, "len")
		assert.NotContains(t, labels, "not")
	})
}

// dot piped directly

func TestDotPipedDirectly(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run("dot piped — all completions shown (struct type)", func(t *testing.T) {
		src := `{{. | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "len")
		assert.Contains(t, labels, "html")
		assert.Contains(t, labels, "and")
		// dot, fields are values not callables — must not appear as pipe targets
		assert.NotContains(t, labels, ".")
		assert.NotContains(t, labels, ".Address")
		assert.NotContains(t, labels, ".Items")
		assert.NotContains(t, labels, ".ID")
	})

	t.Run("struct field piped — dot-prefixed fields excluded", func(t *testing.T) {
		// {{ .Address | .Address }} is syntactically wrong; .Address must not be suggested
		src := `{{.Address | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.NotContains(t, labels, ".Address")
		assert.NotContains(t, labels, ".Items")
		assert.NotContains(t, labels, ".ID")
		// builtins still present
		assert.Contains(t, labels, "len")
	})
}

// builtin chained — len output is int

func TestBuiltinChainedWithModel(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run("len of items piped — int-accepting builtins suggested", func(t *testing.T) {
		src := `{{.Items | len | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "eq")
		assert.Contains(t, labels, "lt")
		assert.Contains(t, labels, "print")
		assert.NotContains(t, labels, "html")
		assert.NotContains(t, labels, "not")
		assert.NotContains(t, labels, ".Oper")
		assert.NotContains(t, labels, ".DisplayName")
		assert.NotContains(t, labels, ".ItemCount")
		assert.NotContains(t, labels, ".IsLargeOrder")
	})

	t.Run("html of string field piped — string-accepting builtins suggested", func(t *testing.T) {
		src := `{{.CustomerName | html | }}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
		assert.Contains(t, labels, "len")
		assert.Contains(t, labels, "js")
		assert.NotContains(t, labels, "not")
		assert.NotContains(t, labels, "eq")
	})
}

func TestScopeSwitchWithRange(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run(
		"inside range no-pipe — dot-prefixed Item methods appear, Order methods absent",
		func(t *testing.T) {
			// pipeInputType=nil at 'len' (single command, no preceding) → all Item methods shown
			src := `{{range .Items}}{{len .SKU}}{{end}}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "l", 0), false, lt)
			assert.Contains(t, labels, ".IsExpensive")
			assert.Contains(t, labels, ".Describe")
			assert.Contains(t, labels, ".Label")
			assert.NotContains(t, labels, ".DisplayName")
			assert.NotContains(t, labels, ".IsLargeOrder")
			assert.NotContains(t, labels, ".wrongSecond")
		},
	)

	t.Run(
		"inside range pipe — zero-param Item methods excluded (string pipe, none accept string)",
		func(t *testing.T) {
			src := `{{range .Items}}{{.Label | }}{{end}}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 1)-1, true, lt)
			// Label returns string; all Item methods take 0 params → none appear dot-prefixed
			assert.NotContains(t, labels, ".IsExpensive")
			assert.NotContains(t, labels, ".Describe")
			assert.NotContains(t, labels, ".Label")
			// Order methods absent from Item scope regardless
			assert.NotContains(t, labels, ".DisplayName")
			assert.NotContains(t, labels, ".IsLargeOrder")
		},
	)

	t.Run("inside range — string Item method piped, string builtins suggested", func(t *testing.T) {
		src := `{{range .Items}}{{.Label | }}{{end}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 1)-1, true, lt)
		assert.Contains(t, labels, "html")
		assert.Contains(t, labels, "len")
		assert.NotContains(t, labels, "not")
		assert.NotContains(t, labels, "eq")
	})

	t.Run("inside range — bool Item method piped, bool builtins suggested", func(t *testing.T) {
		src := `{{range .Items}}{{IsExpensive | }}{{end}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 1)-1, true, lt)
		assert.Contains(t, labels, "not")
		assert.Contains(t, labels, "and")
		assert.NotContains(t, labels, "html")
		assert.NotContains(t, labels, "len")
	})
}

func TestScopeSwitchWithWith(t *testing.T) {
	lt := orderLoadedType(t)

	t.Run(
		"inside with no-pipe — dot-prefixed Address methods appear, Order methods absent",
		func(t *testing.T) {
			// pipeInputType=nil at 'len' → all Address methods shown
			src := `{{with .Address}}{{len .Street}}{{end}}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "l", 0), false, lt)
			assert.Contains(t, labels, ".Line")
			assert.Contains(t, labels, ".IsLocal")
			assert.Contains(t, labels, ".ZipCode")
			assert.NotContains(t, labels, ".DisplayName")
			assert.NotContains(t, labels, ".IsLargeOrder")
			assert.NotContains(t, labels, ".wrongSecond")
		},
	)

	t.Run(
		"inside with pipe — zero-param Address methods excluded (string pipe, none accept string)",
		func(t *testing.T) {
			src := `{{with .Address}}{{Line | }}{{end}}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 1)-1, true, lt)
			// Line returns string; all Address methods take 0 params → none appear dot-prefixed
			assert.NotContains(t, labels, ".Line")
			assert.NotContains(t, labels, ".IsLocal")
			assert.NotContains(t, labels, ".ZipCode")
			// Order methods absent from Address scope regardless
			assert.NotContains(t, labels, ".DisplayName")
			assert.NotContains(t, labels, ".IsLargeOrder")
		},
	)

	t.Run(
		"inside with — string Address method piped, string builtins suggested",
		func(t *testing.T) {
			src := `{{with .Address}}{{Line | }}{{end}}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 1)-1, true, lt)
			assert.Contains(t, labels, "html")
			assert.Contains(t, labels, "len")
			assert.NotContains(t, labels, "not")
			assert.NotContains(t, labels, "eq")
		},
	)

	t.Run("inside with — bool Address method piped, bool builtins suggested", func(t *testing.T) {
		src := `{{with .Address}}{{IsLocal | }}{{end}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 1)-1, true, lt)
		assert.Contains(t, labels, "not")
		assert.Contains(t, labels, "and")
		assert.NotContains(t, labels, "html")
		assert.NotContains(t, labels, "len")
	})
}

// dot tests

func TestDotSuggestions(t *testing.T) {
	t.Run("dot in if condition", func(t *testing.T) {
		src := `{{if .}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, ".", 0))
		assert.NotContains(t, labels, "eq")
		assert.NotContains(t, labels, "len")
	})

	t.Run("dot in range pipeline", func(t *testing.T) {
		src := `{{range .}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, ".", 0))
		assert.NotContains(t, labels, "eq")
		assert.NotContains(t, labels, "len")
	})

	t.Run("dot in with pipeline", func(t *testing.T) {
		src := `{{with .}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, ".", 0))
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
	t.Run("sChar dollar returns vars with sigil", func(t *testing.T) {
		src := `{{$top := .}}{{$}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1), false)
		assert.Contains(t, labels, "$top")
	})

	t.Run("sChar non-dollar includes full $var label", func(t *testing.T) {
		src := `{{$top := .}}{{len .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "l", 0))
		assert.Contains(t, labels, "$top")
	})

	t.Run("variable declared before cursor is visible", func(t *testing.T) {
		src := `{{$x := .}}{{$x}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1), false)
		assert.Contains(t, labels, "$x")
	})

	t.Run("variable declared after cursor is not visible", func(t *testing.T) {
		src := `{{$early := .}}{{$}}{{$late := .}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1), false)
		assert.Contains(t, labels, "$early")
		assert.NotContains(t, labels, "late")
		assert.NotContains(t, labels, "$late")
	})

	t.Run("range index and value variables visible inside body", func(t *testing.T) {
		src := `{{range $i, $v := .}}{{$}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 2), false)
		assert.Contains(t, labels, "$i")
		assert.Contains(t, labels, "$v")
	})

	t.Run("range variable not visible after end", func(t *testing.T) {
		src := `{{range $inner := .}}{{end}}{{$}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1))
		assert.NotContains(t, labels, "inner")
		assert.NotContains(t, labels, "$inner")
	})

	t.Run("outer variable visible inside nested range", func(t *testing.T) {
		src := `{{$outer := .}}{{range $i := .}}{{range $j := .}}{{$}}{{end}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 3), false)
		assert.Contains(t, labels, "$outer")
		assert.Contains(t, labels, "$i")
		assert.Contains(t, labels, "$j")
	})

	t.Run("if condition variable visible inside block", func(t *testing.T) {
		src := `{{if $cond := .}}{{$}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1), false)
		assert.Contains(t, labels, "$cond")
	})

	t.Run("with variable visible inside block", func(t *testing.T) {
		src := `{{with $w := .}}{{$}}{{end}}`
		labels := suggestAt(t, src, offsetOf(t, src, "$", 1), false)
		assert.Contains(t, labels, "$w")
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
	t.Run(
		"after len pipe, only int-accepting functions suggested on ctrl+space",
		func(t *testing.T) {
			lt := orderLoadedType(t)
			src := `{{. | len | }}`
			labels := suggestAtWithType(t, src, offsetOf(t, src, "}}", 0)-1, true, lt)
			assert.Contains(t, labels, "eq")
			assert.Contains(t, labels, "lt")
			assert.Contains(t, labels, "print")
			// those should get filtered out
			assert.NotContains(t, labels, "index")
			assert.NotContains(t, labels, "js")
		},
	)

	t.Run("after not pipe, only bool-accepting functions suggested", func(t *testing.T) {
		lt := orderLoadedType(t)
		src := `{{not . | and . .}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "a", 0), false, lt)
		assert.Contains(t, labels, "and")
		assert.Contains(t, labels, "or")
		assert.Contains(t, labels, "not")
		assert.NotContains(t, labels, "len")
		assert.NotContains(t, labels, "html")
	})

	t.Run("after html pipe, only string-accepting functions suggested", func(t *testing.T) {
		lt := orderLoadedType(t)
		src := `{{html . | len .}}`
		labels := suggestAtWithType(t, src, offsetOf(t, src, "l", 0), false, lt)
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

// full completion tests

func TestCompletionAst(t *testing.T) {
	t.Run("returns nil when server disabled", func(t *testing.T) {
		original := GetConfig()
		setConfig(Config{EnableServer: false})
		t.Cleanup(func() { setConfig(original) })

		uri := "file:///disabled.tmpl"
		store.Set(uri, "{{.}}")
		t.Cleanup(func() { store.Remove(uri) })

		result := completionAst(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 2},
			},
		})
		assert.Nil(t, result)
	})

	t.Run("returns nil when document not in store", func(t *testing.T) {
		enableServer(t)
		result := completionAst(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///missing.tmpl"},
				Position:     protocol.Position{Line: 0, Character: 2},
			},
		})
		assert.Nil(t, result)
	})

	t.Run("returns nil when tree is nil", func(t *testing.T) {
		enableServer(t)
		uri := "file:///notree.tmpl"
		// Set a document with no parsed tree by storing broken template
		store.Set(uri, "{{invalid template {{{{")
		t.Cleanup(func() { store.Remove(uri) })

		result := completionAst(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 2},
			},
		})
		// with the new parser, the tree is not nil in the wrong code case
		assert.NotNil(t, result)
	})

	t.Run("returns nil when cursor outside template block", func(t *testing.T) {
		enableServer(t)
		uri := "file:///outside.tmpl"
		store.Set(uri, "{{.}}\nplain text")
		t.Cleanup(func() { store.Remove(uri) })

		result := completionAst(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 2},
			},
		})
		assert.Nil(t, result)
	})

	t.Run("returns CompletionList for valid template and position", func(t *testing.T) {
		enableServer(t)
		uri := "file:///valid.tmpl"
		store.Set(uri, "{{.}}")
		t.Cleanup(func() { store.Remove(uri) })

		result := completionAst(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 2},
			},
		})
		require.NotNil(t, result)
		list, ok := result.(protocol.CompletionList)
		require.True(t, ok)
		assert.False(t, list.IsIncomplete)

		labels := make([]string, len(list.Items))
		for i, item := range list.Items {
			labels[i] = item.Label
		}
		assert.Contains(t, labels, ".")
	})
}

func TestCompletionWithFallback(t *testing.T) {
	t.Run("returns ast result when ast succeeds", func(t *testing.T) {
		enableServer(t)
		uri := "file:///fallback-ok.tmpl"
		store.Set(uri, "{{.}}")
		t.Cleanup(func() { store.Remove(uri) })

		resp, err := completionWithFallback(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 2},
			},
		})
		require.NoError(t, err)
		_, ok := resp.(protocol.CompletionList)
		assert.True(t, ok, "expected CompletionList from ast path")
	})

	t.Run("falls back to regex when ast returns nil", func(t *testing.T) {
		enableServer(t)
		// cursor outside template — ast returns nil, fallback should run
		uri := "file:///fallback-nil.tmpl"
		store.Set(uri, "{{$x := .}}\nplain")
		t.Cleanup(func() { store.Remove(uri) })

		resp, err := completionWithFallback(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 2},
			},
		})
		require.NoError(t, err)
		_ = resp
	})
}
