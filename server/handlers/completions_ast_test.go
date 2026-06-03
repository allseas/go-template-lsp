package handlers

import (
	"go/types"
	"testing"
	parse "text-template-parser"
	serverTypes "text-template-server/types"

	"golang.org/x/tools/go/packages"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func suggestAt(t *testing.T, src string, offset int) []string {
	t.Helper()
	trees, err := parse.Parse("test", src, "", "", builtins())
	require.NoError(t, err)
	root := trees["test"].Root
	ctx := &Context{Vars: map[string]parse.Node{}}
	cur := nodeFind(root, parse.Pos(offset))
	ok := buildPath(root, cur, ctx)
	require.True(t, ok)
	var parent parse.Node
	if len(ctx.Path) >= 2 {
		parent = ctx.Path[len(ctx.Path)-2]
	}
	items := suggest(parent, ctx, src[offset], false, protocol.Range{})
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

func suggestAtWithType(
	t *testing.T,
	src string,
	offset int,
	isInvoked bool,
	lt *serverTypes.LoadedType,
) []string {
	t.Helper()
	typ := parse.New("test")
	typ.Mode = parse.ParsePartial | parse.SkipFuncCheck
	treeSet := map[string]*parse.Tree{}
	_, err := typ.Parse(src, "{{", "}}", treeSet, builtins())
	require.NoError(t, err)
	root := typ.Root
	ctx := &Context{Vars: map[string]parse.Node{}, DotType: lt}
	cur := nodeFind(root, parse.Pos(offset))
	ok := buildPath(root, cur, ctx)
	require.True(t, ok)
	var parent parse.Node
	if len(ctx.Path) >= 2 {
		parent = ctx.Path[len(ctx.Path)-2]
	}
	items := suggest(parent, ctx, src[offset], isInvoked, protocol.Range{})
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

func builtins() map[string]any {
	return map[string]any{
		"and": true, "call": true, "html": true, "index": true,
		"slice": true, "js": true, "len": true, "not": true, "or": true,
		"print": true, "printf": true, "println": true, "urlquery": true,
		"eq": true, "ne": true, "lt": true, "le": true, "gt": true, "ge": true,
		"DisplayName": true, "Summary": true, "ItemCount": true,
		"IsLargeOrder": true, "Format": true, "Label": true, "Total": true,
		"IsExpensive": true, "Describe": true, "Line": true, "IsLocal": true,
		"ZipCode": true,
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

func orderLoadedType(t *testing.T) *serverTypes.LoadedType {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  "../../test/resources/typehints-tests",
	}
	pkgs, err := packages.Load(cfg, "text-template-server/src/model")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	pkg := pkgs[0]
	obj := pkg.Types.Scope().Lookup("Order")
	require.NotNil(t, obj)
	named := obj.Type().(*types.Named)
	return &serverTypes.LoadedType{
		Pkg:     pkg,
		Named:   named,
		Fields:  serverTypes.StructFields(named),
		Methods: serverTypes.NamedMethods(named),
	}
}

func TestCompletionSuggestions(t *testing.T) {
	lt := orderLoadedType(t)
	for _, tc := range completionTestCases {
		t.Run(tc.name, func(t *testing.T) {
			offset := offsetOf(t, tc.src, tc.subStr, tc.occurrence) + tc.offsetAdj
			var labels []string
			if tc.withType {
				labels = suggestAtWithType(t, tc.src, offset, tc.isInvoked, lt)
			} else {
				labels = suggestAt(t, tc.src, offset)
			}
			for _, want := range tc.contains {
				assert.Contains(t, labels, want)
			}
			for _, notWant := range tc.notContains {
				assert.NotContains(t, labels, notWant)
			}
		})
	}
}

func TestNodeFind(t *testing.T) {
	for _, tc := range nodeFindTestCases {
		t.Run(tc.name, func(t *testing.T) {
			trees, err := parse.Parse("test", tc.src, "", "", builtins())
			require.NoError(t, err)
			node := nodeFind(trees["test"].Root, parse.Pos(tc.pos))
			if tc.isDot {
				_, ok := node.(*parse.DotNode)
				assert.True(t, ok, "expected DotNode, got %T", node)
			}
			if tc.isIdent {
				id, ok := node.(*parse.IdentifierNode)
				require.True(t, ok, "expected IdentifierNode, got %T", node)
				assert.Equal(t, tc.ident, id.Ident)
			}
			if tc.isVar {
				v, ok := node.(*parse.VariableNode)
				require.True(t, ok, "expected VariableNode, got %T", node)
				assert.Equal(t, tc.varIdent, v.Ident[0])
			}
		})
	}
}

func TestBuildPathScope(t *testing.T) {
	for _, tc := range buildPathScopeTestCases {
		t.Run(tc.name, func(t *testing.T) {
			trees, err := parse.Parse("test", tc.src, "", "", builtins())
			require.NoError(t, err)
			root := trees["test"].Root
			ctx := &Context{Vars: map[string]parse.Node{}}
			pos := parse.Pos(offsetOf(t, tc.src, ".", tc.dotOccur))
			buildPath(root, nodeFind(root, pos), ctx)
			_, present := ctx.Vars[tc.varName]
			assert.Equal(t, tc.wantPresent, present)
		})
	}
}

func TestCompletionAst(t *testing.T) {
	for _, tc := range completionAstTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.serverDisabled {
				original := GetConfig()
				setConfig(Config{EnableServer: false})
				t.Cleanup(func() { setConfig(original) })
			} else {
				enableServer(t)
			}
			if !tc.skipStore {
				store.Set(tc.uri, tc.content)
				t.Cleanup(func() { store.Remove(tc.uri) })
			}
			result := completionAst(nil, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: tc.uri},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			})
			if tc.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			if len(tc.wantLabels) > 0 {
				list, ok := result.(protocol.CompletionList)
				require.True(t, ok)
				assert.False(t, list.IsIncomplete)
				assert.Contains(t, labelsFrom(t, result), tc.wantLabels[0])
			}
		})
	}
}

func TestCompletionWithFallback(t *testing.T) {
	for _, tc := range completionFallbackTestCases {
		t.Run(tc.name, func(t *testing.T) {
			enableServer(t)
			store.Set(tc.uri, tc.content)
			t.Cleanup(func() { store.Remove(tc.uri) })
			resp, err := CompletionWithFallback(nil, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: tc.uri},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			})
			require.NoError(t, err)
			if tc.wantList {
				_, ok := resp.(protocol.CompletionList)
				assert.True(t, ok, "expected CompletionList")
			}
		})
	}
}

func TestCompletionAstInvokedDollarPrefixShowsVariables(t *testing.T) {
	enableServer(t)
	uri := "file:///invoked-dollar.tmpl"
	content := `{{$top := .}}{{$}}`
	store.Set(uri, content)
	t.Cleanup(func() { store.Remove(uri) })

	const pos protocol.UInteger = 16
	result := completionAst(nil, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: pos},
		},
		Context: &protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
	})

	require.NotNil(t, result)
	labels := labelsFrom(t, result)
	assert.Contains(t, labels, "top")
	assert.NotContains(t, labels, "$top")
}
