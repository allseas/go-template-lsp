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
	tt := serverTypes.NewTree(*trees["test"], nil, nil, nil, nil)
	cur := serverTypes.NodeFind(tt.Root, serverTypes.Pos(offset))
	require.NotNil(t, cur)
	items := suggest(cur, src[offset], false, serverTypes.Pos(offset), src, protocol.Range{})
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
	lt *serverTypes.Tree,
) []string {
	t.Helper()
	typ := parse.New("test")
	typ.Mode = parse.ParsePartial | parse.SkipFuncCheck
	treeSet := map[string]*parse.Tree{}
	_, err := typ.Parse(src, "{{", "}}", treeSet, builtins())
	require.NoError(t, err)
	var dotType types.Type
	var pkg *types.Package
	if lt != nil {
		dotType = lt.DotType
		pkg = lt.Pkg
	}
	// Pass GlobalFuncs() so IdentifierNodes carry their function signatures.
	// This ensures cmd.ValueType() is set for function commands in pipes,
	// allowing pipeOutputInfo to rely solely on the typed tree.
	tt := serverTypes.NewTree(*typ, serverTypes.GlobalFuncs(), dotType, pkg, nil)
	if lt != nil {
		tt.DotType = lt.DotType
		tt.Pkg = lt.Pkg
	}
	cur := serverTypes.NodeFind(tt.Root, serverTypes.Pos(offset))
	require.NotNil(t, cur)
	items := suggest(cur, src[offset], isInvoked, serverTypes.Pos(offset), src, protocol.Range{})
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

func orderLoadedType(t *testing.T) *serverTypes.Tree {
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
	tree := serverTypes.Tree{DotType: named, Pkg: pkg.Types}
	return &tree
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

func TestChainEditCompletions(t *testing.T) {
	lt := orderLoadedType(t)
	for _, tc := range chainEditTestCases {
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

func TestCompletionAst(t *testing.T) {
	for _, tc := range completionAstTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.serverDisabled {
				original := GetConfig()
				setConfig(Config{EnableAutocompletion: false})
				t.Cleanup(func() { setConfig(original) })
			} else {
				enableAutocompletion(t)
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
			enableAutocompletion(t)
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

// labelsOf extracts the Label field from a CompletionItem slice.
func labelsOf(items []protocol.CompletionItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

// typedNodeAt builds a typed tree from src (using partial-parse mode) and
// returns the deepest node at the given byte offset.
func typedNodeAt(t *testing.T, src string, offset int, lt *serverTypes.Tree) serverTypes.Node {
	t.Helper()
	typ := parse.New("test")
	typ.Mode = parse.ParsePartial | parse.SkipFuncCheck
	treeSet := map[string]*parse.Tree{}
	_, err := typ.Parse(src, "{{", "}}", treeSet, builtins())
	require.NoError(t, err)
	var dotType types.Type
	var pkg *types.Package
	if lt != nil {
		dotType = lt.DotType
		pkg = lt.Pkg
	}
	tt := serverTypes.NewTree(*typ, nil, dotType, pkg, nil)
	cur := serverTypes.NodeFind(tt.Root, serverTypes.Pos(offset))
	require.NotNil(t, cur)
	return cur
}

func TestVarsItemsT(t *testing.T) {
	wr := protocol.Range{}
	for _, tc := range varsItemsTTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var vars []*serverTypes.VariableNode
			if tc.varNames != nil {
				vars = make([]*serverTypes.VariableNode, len(tc.varNames))
				for i, name := range tc.varNames {
					vars[i] = &serverTypes.VariableNode{
						NodeType: serverTypes.NodeVariable,
						Ident:    []string{name},
					}
				}
			}
			items := varsItemsT(vars, tc.delSign, wr)
			if tc.wantNil {
				assert.Nil(t, items)
				return
			}
			if tc.wantLen > 0 {
				assert.Len(t, items, tc.wantLen)
			}
			labels := labelsOf(items)
			for _, want := range tc.wantLabels {
				assert.Contains(t, labels, want)
			}
			for _, notWant := range tc.notContains {
				assert.NotContains(t, labels, notWant)
			}
			if tc.wantFilter != "" {
				require.NotEmpty(t, items)
				require.NotNil(t, items[0].FilterText)
				assert.Equal(t, tc.wantFilter, *items[0].FilterText)
			}
		})
	}
}

func TestFieldChainItemsT(t *testing.T) {
	wr := protocol.Range{}
	lt := orderLoadedType(t)
	for _, tc := range fieldChainItemsTTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var typ types.Type
			if tc.useBasic {
				typ = types.Typ[types.String]
			} else if tc.useOrder {
				typ = lt.DotType
			}
			items := fieldChainItemsT(typ, outputAny, wr)
			if tc.wantEmpty {
				assert.Empty(t, items)
				return
			}
			labels := labelsOf(items)
			for _, want := range tc.contains {
				assert.Contains(t, labels, want)
			}
			for _, notWant := range tc.notContains {
				assert.NotContains(t, labels, notWant)
			}
		})
	}
}

func TestDotItemsT(t *testing.T) {
	wr := protocol.Range{}
	lt := orderLoadedType(t)
	for _, tc := range dotItemsTTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var loadedType *serverTypes.Tree
			if tc.withType {
				loadedType = lt
			}
			offset := offsetOf(t, tc.src, tc.subStr, 0)
			cur := typedNodeAt(t, tc.src, offset, loadedType)
			var inputType types.Type
			if tc.inputIsString {
				inputType = types.Typ[types.String]
			}
			items := dotItemsT(cur, tc.delSign, inputType, tc.pipeKind, outputAny, wr)
			if tc.wantEmpty {
				assert.Empty(t, items)
				return
			}
			labels := labelsOf(items)
			for _, want := range tc.contains {
				assert.Contains(t, labels, want)
			}
			for _, notWant := range tc.notContains {
				assert.NotContains(t, labels, notWant)
			}
		})
	}
}

func TestPipeFilteredItemsT(t *testing.T) {
	wr := protocol.Range{}
	for _, tc := range pipeFilteredItemsTTestCases {
		t.Run(tc.name, func(t *testing.T) {
			offset := offsetOf(t, tc.src, tc.subStr, 0)
			cur := typedNodeAt(t, tc.src, offset, nil)
			var inputType types.Type
			if tc.inputIsString {
				inputType = types.Typ[types.String]
			}
			labels := labelsOf(pipeFilteredItemsT(cur, tc.kind, inputType, nil, outputAny, wr))
			for _, want := range tc.contains {
				assert.Contains(t, labels, want)
			}
			for _, notWant := range tc.notContains {
				assert.NotContains(t, labels, notWant)
			}
		})
	}
}

// basicTypeFor returns a types.Type for a string spec used in tests. "none"
// returns a *types.Named so basicTypeMatchesKind exercises its non-basic branch.
func basicTypeFor(t *testing.T, spec string, lt *serverTypes.Tree) types.Type {
	t.Helper()
	switch spec {
	case "string":
		return types.Typ[types.String]
	case "int":
		return types.Typ[types.Int]
	case "bool":
		return types.Typ[types.Bool]
	case "float64":
		return types.Typ[types.Float64]
	case "none":
		return lt.DotType
	}
	t.Fatalf("unknown basic spec %q", spec)
	return nil
}

func TestBasicTypeMatchesKind(t *testing.T) {
	lt := orderLoadedType(t)
	for _, tc := range basicTypeMatchesKindTestCases {
		t.Run(tc.name, func(t *testing.T) {
			typ := basicTypeFor(t, tc.basic, lt)
			got := basicTypeMatchesKind(typ, tc.kind)
			assert.Equal(t, tc.want, got)
		})
	}
}

// findOrderMethod returns the MethodType for the named Order method.
func findOrderMethod(t *testing.T, lt *serverTypes.Tree, name string) serverTypes.MethodType {
	t.Helper()
	for _, m := range serverTypes.NamedMethods(lt.DotType) {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("method %q not found on Order", name)
	return serverTypes.MethodType{}
}

func TestMethodAcceptsInput(t *testing.T) {
	lt := orderLoadedType(t)
	for _, tc := range methodAcceptsInputTestCases {
		t.Run(tc.name, func(t *testing.T) {
			m := findOrderMethod(t, lt, tc.methodName)
			var inputType types.Type
			switch {
			case tc.inputIsString:
				inputType = types.Typ[types.String]
			case tc.inputIsInt:
				inputType = types.Typ[types.Int]
			}
			got := methodAcceptsInput(m, inputType, tc.pipeKind)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMethodIsUsable(t *testing.T) {
	lt := orderLoadedType(t)
	for _, tc := range methodIsUsableTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var m serverTypes.MethodType
			if !tc.nilFunc {
				m = findOrderMethod(t, lt, tc.methodName)
			}
			got := methodIsUsable(m)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestToNamed(t *testing.T) {
	lt := orderLoadedType(t)
	for _, tc := range toNamedTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var typ types.Type
			switch tc.input {
			case "nil":
				typ = nil
			case "named":
				typ = lt.DotType
			case "pointer":
				typ = types.NewPointer(lt.DotType)
			case "basic":
				typ = types.Typ[types.String]
			}
			got := toNamed(typ)
			if tc.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestCompletionAstInvokedDollarPrefixShowsVariables(t *testing.T) {
	enableAutocompletion(t)
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

// TestCompletionAstMultiDefines drives completionAst across multiple
// {{define}} blocks in one document, each carrying its own gotype hint, to
// verify that dot-completion uses the per-tree loaded type.
func TestCompletionAstMultiDefines(t *testing.T) {
	loaded := loadModelTypes(t, "Order", "Address")
	perTree := map[string]*serverTypes.Tree{
		"t":          loaded["Address"],
		"OrderTpl":   loaded["Order"],
		"AddressTpl": loaded["Address"],
	}

	src := multiDefinesTemplate
	uri := "file:///comp-multidefines.tmpl"
	setDocMulti(t, uri, src, perTree)
	t.Cleanup(func() { store.Remove(uri) })

	for _, tc := range completionAstMultiDefineCases {
		t.Run(tc.name, func(t *testing.T) {
			pos := posOfSubStr(t, src, tc.posSubStr, tc.posOccurrence)
			pos.Character += uint32(tc.posCharOffset) //nolint:gosec

			result := completionAst(nil, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     pos,
				},
			})
			require.NotNil(t, result)
			labels := labelsFrom(t, result)
			for _, want := range tc.wantContains {
				assert.Contains(t, labels, want)
			}
			for _, notWant := range tc.wantNotContain {
				assert.NotContains(t, labels, notWant)
			}
		})
	}
}
