package handlers

import (
	"go/types"
	"strings"
	"testing"

	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"golang.org/x/tools/go/packages"
)

func TestDefinitionVariable(t *testing.T) {
	src := "{{ $test := 0 }}\n{{ $test }}"
	uri := "file:///def-var.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	assert.Len(t, locations, 1)
	assert.Equal(t, uri, locations[0].URI)
	assert.Equal(t, position(0, 3), locations[0].Range.Start)
}

func TestDefinitionVariableRedeclaration(t *testing.T) {
	src := "{{ $test := 0 }}\n{{ $test }}\n{{ $test := 1 }}\n{{ $test }}"
	uri := "file:///def-redecl.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(3, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	assert.Len(t, locations, 2)
}

func TestDefinitionVariableOnDeclaration(t *testing.T) {
	src := "{{ $test := 0 }}\n{{ $test }}"
	uri := "file:///def-on-decl.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	assert.Len(t, locations, 1)
	assert.Equal(
		t,
		protocol.Location{
			URI: uri,
			Range: protocol.Range{
				Start: protocol.Position{Line: 0x0, Character: 0x3},
				End:   protocol.Position{Line: 0x0, Character: 0x8},
			},
		},
		locations[0],
	)
}

func TestDefinitionDotInRange(t *testing.T) {
	src := "{{- range .Join }}\n{{ . }}\n{{- end }}"
	uri := "file:///def-dot-range.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 3),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.Equal(t, uri, loc.URI)

	assert.Equal(t, uint32(0), loc.Range.Start.Line)
}

func TestDefinitionDotInWith(t *testing.T) {
	src := "{{- with .Obj }}\n{{ . }}\n{{- end }}"
	uri := "file:///def-dot-with.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 3),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.Equal(t, uint32(0), loc.Range.Start.Line)
	assert.Equal(t, uint32(0), loc.Range.End.Line)
}

func TestDefinitionDotOutsideRange(t *testing.T) {
	src := "{{ . }}"
	uri := "file:///def-dot-top.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 3),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinitionField(t *testing.T) {
	src := "{{ .Name }}"
	uri := "file:///def-field.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinitionNoNode(t *testing.T) {
	src := "hello {{ $x := 1 }}"
	uri := "file:///def-nonode.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 1),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinitionNoDefinition(t *testing.T) {
	src := "{{ $variable }}\ntext"
	uri := "file:///def-no-def.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// setDocWithType inserts a document into the store with a pre-loaded type,
// bypassing the type-hint loading mechanism that requires a workspace root.
func setDocWithType(t *testing.T, uri, src string, lt *serverTypes.Tree) {
	t.Helper()
	tree, treeSet, err := tryParse(src)
	require.NoError(t, err)
	typedTrees := make(map[string]*serverTypes.Tree, len(treeSet))
	for name, tr := range treeSet {
		typedTrees[name] = buildTypedTree(tr, lt, nil)
	}
	var typed *serverTypes.Tree
	if tree != nil {
		typed = typedTrees[tree.Name]
	}
	store.mu.Lock()
	store.docs[uri] = &document{
		text:       src,
		tree:       tree,
		trees:      treeSet,
		loadedType: lt,
		typedTree:  typed,
		typedTrees: typedTrees,
	}
	store.mu.Unlock()
}

func definitionOrderType(t *testing.T) *serverTypes.Tree {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedFiles,
		Dir:  "../../test/resources/definition-tests-server",
	}
	pkgs, err := packages.Load(cfg, "text-template-server/src/model")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	pkg := pkgs[0]
	obj := pkg.Types.Scope().Lookup("Order")
	require.NotNil(t, obj)
	named, ok := obj.Type().(*types.Named)
	require.True(t, ok)
	return &serverTypes.Tree{
		DotType: named,
		Pkg:     pkg.Types,
		Fset:    pkg.Fset,
	}
}

func TestDefinitionFieldWithType(t *testing.T) {
	src := "{{ .CustomerName }}"
	uri := "file:///def-field-type.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(70), loc.Range.Start.Line)
}

func TestDefinitionFieldMethodWithType(t *testing.T) {
	src := "{{ .DisplayName }}"
	uri := "file:///def-field-method.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(79), loc.Range.Start.Line)
}

func TestDefinitionNestedFieldFirstIdent(t *testing.T) {
	// cursor on "Address"
	src := "{{ .Address.City }}"
	uri := "file:///def-nested-first.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	// "{{ ." is 4 chars, so char 5 is inside "Address"
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(72), loc.Range.Start.Line)
}

func TestDefinitionNestedFieldSecondIdent(t *testing.T) {
	// cursor on "City" (the second identifier)
	src := "{{ .Address.City }}"
	uri := "file:///def-nested-second.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	// "{{ .Address." is 12 chars, so char 13 is inside "City"
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 13),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(7), loc.Range.Start.Line)
}

func TestDefinitionFieldUnknownField(t *testing.T) {
	src := "{{ .NonExistent }}"
	uri := "file:///def-field-unknown.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinitionFieldNoLoadedType(t *testing.T) {
	src := "{{ .CustomerName }}"
	uri := "file:///def-field-notype.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 5),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestFunctionReturn(t *testing.T) {
	src := "{{ .Address.Copy.Copy.Street }}"
	uri := "file:///def-field-func.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 24),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(6), loc.Range.Start.Line)
}

func TestDefinitionFieldInRangeContext(t *testing.T) {
	src := "{{ range .Items }}\n{{ .Name }}\n{{ end }}"
	uri := "file:///def-field-range.tmpl"
	lt := loadModelTypes(t, "Order")["Order"]
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(40), loc.Range.Start.Line)
}

func TestDefinitionFieldInWithContext(t *testing.T) {
	src := "{{ with .Address }}\n{{ .City }}\n{{ end }}"
	uri := "file:///def-field-with.tmpl"
	lt := loadModelTypes(t, "Order")["Order"]
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(t, uint32(7), loc.Range.Start.Line)
}

func TestDefinitionMethodInRangeContext(t *testing.T) {
	src := "{{ range .Items }}\n{{ .Describe }}\n{{ end }}"
	uri := "file:///def-method-range.tmpl"
	lt := loadModelTypes(t, "Order")["Order"]
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	assert.Equal(
		t,
		uint32(61),
		loc.Range.Start.Line,
	)
}

// TestDefinitionMultiDefines tests Definition inside a document with
// multiple {{define}} blocks, each (optionally) preceded by its own gotype hint.
func TestDefinitionMultiDefines(t *testing.T) {
	cases := []struct {
		name          string
		posSubStr     string
		posOccurrence int
		posCharOffset int
		wantFile      string
		wantSameURI   bool
		wantLine      uint32
	}{
		{
			name:          "field inside Order define jumps to Order.CustomerName",
			posSubStr:     "CustomerName",
			posOccurrence: 0,
			posCharOffset: 1,
			wantFile:      "model.go",
			wantLine:      70, // Order.CustomerName at line 71 (0-indexed 70)
		},
		{
			name:          "field inside Address define jumps to Address.Street",
			posSubStr:     "Street",
			posOccurrence: 0,
			posCharOffset: 1,
			wantFile:      "model.go",
			wantLine:      6, // Address.Street at line 7 (0-indexed 6)
		},
		{
			name:          "variable inside no-hint define resolves to its declaration",
			posSubStr:     "$local }}",
			posOccurrence: 0,
			posCharOffset: 1,
			wantSameURI:   true,
			wantLine:      12, // $local := . is on line 13 of multiDefinesTemplate (0-indexed 12)
		},
		{
			name:          "field in root template jumps to Address.Country",
			posSubStr:     ".Country",
			posOccurrence: 0,
			posCharOffset: 1,
			wantFile:      "model.go",
			wantLine:      8, // Address.Country at line 9 (0-indexed 8)
		},
	}

	loaded := loadModelTypes(t, "Order", "Address")
	perTree := map[string]*serverTypes.Tree{
		"t":          loaded["Address"],
		"OrderTpl":   loaded["Order"],
		"AddressTpl": loaded["Address"],
	}

	src := multiDefinesTemplate
	uri := "file:///def-multidefines.tmpl"
	setDocMulti(t, uri, src, perTree)
	t.Cleanup(func() { store.Delete(uri) })

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := posOfSubStr(t, src, tc.posSubStr, tc.posOccurrence)
			pos.Character += uint32(tc.posCharOffset) //nolint:gosec

			params := &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     pos,
				},
			}

			result, err := Definition(nil, params)
			require.NoError(t, err)
			require.NotNil(t, result)

			switch r := result.(type) {
			case protocol.Location:
				if tc.wantFile != "" {
					assert.True(t, strings.HasSuffix(r.URI, tc.wantFile),
						"expected URI to end with %q, got %s", tc.wantFile, r.URI)
				}
				if tc.wantSameURI {
					assert.Equal(t, uri, r.URI)
				}
				assert.Equal(t, tc.wantLine, r.Range.Start.Line)
			case []protocol.Location:
				require.NotEmpty(t, r)
				if tc.wantSameURI {
					assert.Equal(t, uri, r[0].URI)
				}
				assert.Equal(t, tc.wantLine, r[0].Range.Start.Line)
			default:
				t.Fatalf("unexpected result type %T", r)
			}
		})
	}
}

// TestDefinitionVariableChainBaseIdent verifies that clicking on the base variable
// ($item) in a chained expression like {{ $item.IsExpensive }} jumps to the
// variable's declaration (the range pipe), not to the Go source file.
func TestDefinitionVariableChainBaseIdent(t *testing.T) {
	src := "{{ range $item := .Items }}\n{{ $item.IsExpensive }}\n{{ end }}"
	uri := "file:///def-var-chain-base.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 4),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok, "expected []protocol.Location for base variable, got %T", result)
	require.NotEmpty(t, locations)
	assert.Equal(t, uri, locations[0].URI)
	assert.Equal(t, uint32(0), locations[0].Range.Start.Line)
}

func TestDefinitionVariableChainMethodIdent(t *testing.T) {
	src := "{{ range $item := .Items }}\n{{ $item.IsExpensive }}\n{{ end }}"
	uri := "file:///def-var-chain-method.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 9),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok, "expected protocol.Location for chained method, got %T", result)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	// Item.IsExpensive is defined at line 57 in model.go (0-indexed: 56)
	assert.Equal(t, uint32(56), loc.Range.Start.Line)
}

// TestDefinitionVariableChainFieldIdent verifies that clicking on a chained struct
// field (.SKU) in {{ $item.SKU }} jumps to the Go source definition.
func TestDefinitionVariableChainFieldIdent(t *testing.T) {
	src := "{{ range $item := .Items }}\n{{ $item.SKU }}\n{{ end }}"
	uri := "file:///def-var-chain-field.tmpl"
	lt := definitionOrderType(t)
	setDocWithType(t, uri, src, lt)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(1, 9),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok, "expected protocol.Location for chained field, got %T", result)
	assert.True(
		t,
		strings.HasSuffix(loc.URI, "model.go"),
		"expected URI to point to model.go, got %s",
		loc.URI,
	)
	// Item.SKU is defined at line 40 in model.go (0-indexed: 39)
	assert.Equal(t, uint32(39), loc.Range.Start.Line)
}

func TestDefinitionFuncMapNamedFunction(t *testing.T) {
	_, err := serverTypes.LoadGlobalFuncs("../../test/resources/funcmap-tests")
	require.NoError(t, err)
	defer serverTypes.SetGlobalFuncs(nil)

	src := "{{ upper .Name }}"
	uri := "file:///def-func-named.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	// Position cursor on "upper" (offset 3 inside "{{ upper")
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 3),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok, "expected protocol.Location, got %T", result)

	assert.True(t, strings.HasSuffix(loc.URI, "funcs.go"),
		"expected URI to point to funcs.go, got %s", loc.URI)
}

func TestDefinitionFuncMapInlineLiteral(t *testing.T) {
	// "shout" is an inline literal (nil *types.Func). Definition should navigate
	// to the string key in the FuncMap literal.
	_, err := serverTypes.LoadGlobalFuncs("../../test/resources/funcmap-tests")
	require.NoError(t, err)
	defer serverTypes.SetGlobalFuncs(nil)

	src := "{{ shout .Name }}"
	uri := "file:///def-func-inline.tmpl"
	store.Set(uri, src)
	defer store.Delete(uri)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position(0, 3),
		},
	}

	result, err := Definition(nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	loc, ok := result.(protocol.Location)
	require.True(t, ok, "expected protocol.Location, got %T", result)

	assert.True(t, strings.HasSuffix(loc.URI, "funcs.go"),
		"expected URI to point to funcs.go, got %s", loc.URI)
}
