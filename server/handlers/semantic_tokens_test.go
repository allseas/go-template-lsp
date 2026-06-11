package handlers

import (
	"go/types"
	"testing"

	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"golang.org/x/tools/go/packages"
)

// semanticOrderType loads the Order type from the templ-tests model package.
func semanticOrderType(t *testing.T) *serverTypes.Tree {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedFiles,
		Dir:  "../../test/resources/templ-tests",
	}
	pkgs, err := packages.Load(cfg, "cg/model")
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

// setDocWithTypedTree inserts a document with a fully built typed tree into the store,
// bypassing the type-hint loading mechanism that requires a workspace root.
func setDocWithTypedTree(t *testing.T, uri, src string, lt *serverTypes.Tree) {
	t.Helper()
	tree, _, err := tryParse(src)
	require.NoError(t, err)
	typedTree := buildTypedTree(tree, lt)
	doc := &document{text: src, tree: tree, loadedType: lt, typedTree: typedTree}
	if doc.typedTree != nil {
		serverTypes.SetEndsForTree(*doc.typedTree, serverTypes.Pos(len(src)), &doc.text)
	}
	store.mu.Lock()
	store.docs[uri] = doc
	store.mu.Unlock()
}

// TestSemanticTokensFull exercises the full handler path via the document store.
func TestSemanticTokensFull(t *testing.T) {
	lt := semanticOrderType(t)

	tests := []struct {
		name    string
		src     string
		wantLen int // expected number of tokens * 5 (LSP encoded format)
		wantNil bool
	}{
		{
			name:    "dot",
			src:     `{{.}}`,
			wantLen: 5, // one token
		},
		{
			name:    "struct field",
			src:     `{{.CustomerName}}`,
			wantLen: 5,
		},
		{
			name:    "method call",
			src:     `{{.DisplayName}}`,
			wantLen: 5,
		},
		{
			name:    "chained field.field",
			src:     `{{.Address.City}}`,
			wantLen: 10, // two tokens
		},
		{
			name:    "chained field.method",
			src:     `{{.Address.Line}}`,
			wantLen: 10,
		},
		{
			name:    "no typed tree when server disabled",
			src:     `{{.}}`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "file:///sem-" + tt.name + ".tmpl"

			if tt.wantNil {
				// document not in store.
				result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
				require.NoError(t, err)
				assert.Nil(t, result)
				return
			}

			setDocWithTypedTree(t, uri, tt.src, lt)
			defer store.Delete(uri)

			result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Data, tt.wantLen)
		})
	}
}

// TestFieldNodeTokenTypes verifies that struct fields are emitted as ttProperty
// and method calls are emitted as ttFunction.
// The LSP semantic token data is a flat uint32 array with 5 values per token:
func TestFieldNodeTokenTypes(t *testing.T) {
	lt := semanticOrderType(t)

	tests := []struct {
		name           string
		src            string
		noType         bool // when true, don't pass type info
		wantTokenCount int
		// tokenType for each token in order, indexed as result.Data[n*5+3]
		wantTypes []uint32
	}{
		{
			// .CustomerName is a plain struct field (string)
			name:           "struct field emits property token",
			src:            `{{.CustomerName}}`,
			wantTokenCount: 1,
			wantTypes:      []uint32{ttProperty},
		},
		{
			// .DisplayName is a method (func returning string)
			name:           "method emits function token",
			src:            `{{.DisplayName}}`,
			wantTokenCount: 1,
			wantTypes:      []uint32{ttFunction},
		},
		{
			// .Address is a struct field, .City is a struct field on Address
			name:           "chained field.field both emit property tokens",
			src:            `{{.Address.City}}`,
			wantTokenCount: 2,
			wantTypes:      []uint32{ttProperty, ttProperty},
		},
		{
			// .Address is a struct field, .Line is a method on Address
			name:           "chained field.method emits property then function token",
			src:            `{{.Address.Line}}`,
			wantTokenCount: 2,
			wantTypes:      []uint32{ttProperty, ttFunction},
		},
		{
			// .Address.Info.Desc, three-level all-field chain
			name:           "three-level field chain all property tokens",
			src:            `{{.Address.Info.Desc}}`,
			wantTokenCount: 3,
			wantTypes:      []uint32{ttProperty, ttProperty, ttProperty},
		},
		{
			// no type info, all segments fall back to ttProperty
			name:           "without type info field falls back to property",
			src:            `{{.SomeField}}`,
			noType:         true,
			wantTokenCount: 1,
			wantTypes:      []uint32{ttProperty},
		},
		{
			// .IsLargeOrder is a method on Order returning bool
			name:           "bool-returning method emits function token",
			src:            `{{.IsLargeOrder}}`,
			wantTokenCount: 1,
			wantTypes:      []uint32{ttFunction},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "file:///field-tok-" + tt.name + ".tmpl"
			useLt := lt
			if tt.noType {
				useLt = nil
			}
			setDocWithTypedTree(t, uri, tt.src, useLt)
			defer store.Delete(uri)

			result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Data, tt.wantTokenCount*5, "unexpected token count")

			for i, wantType := range tt.wantTypes {
				assert.Equal(t, wantType, result.Data[i*5+3], "token[%d] tokenType", i)
			}
		})
	}
}

func makeSemanticParams(uri string) *protocol.SemanticTokensParams {
	return &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}
}

func makeDocSymParams(uri string) *protocol.DocumentSymbolParams {
	return &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}
}

// castDocSymbols casts the any result from DocumentSymbols to []protocol.DocumentSymbol.
func castDocSymbols(t *testing.T, result any) []protocol.DocumentSymbol {
	t.Helper()
	if result == nil {
		return nil
	}
	syms, ok := result.([]protocol.DocumentSymbol)
	require.True(t, ok, "expected []protocol.DocumentSymbol, got %T", result)
	return syms
}

// setDocFromSource registers a document in the global store by parsing it so that
// doc.trees is populated (required by DocumentSymbols).
func setDocFromSource(t *testing.T, uri, src string) {
	t.Helper()
	store.Set(uri, src)
}

// TestDocumentSymbolsNoDocument verifies nil is returned when the document is absent.
func TestDocumentSymbolsNoDocument(t *testing.T) {
	result, err := DocumentSymbols(nil, makeDocSymParams("file:///sym-missing.tmpl"))
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestDocumentSymbolsNoDefines verifies that a document with no {{define}} blocks
// produces no symbols (only the root "t" tree exists, which is skipped).
func TestDocumentSymbolsNoDefines(t *testing.T) {
	uri := "file:///sym-nodefines.tmpl"
	setDocFromSource(t, uri, `Hello {{.Name}}`)
	defer store.Delete(uri)

	result, err := DocumentSymbols(nil, makeDocSymParams(uri))
	require.NoError(t, err)
	syms := castDocSymbols(t, result)
	assert.Empty(t, syms)
}

// TestDocumentSymbolsSingleDefine verifies that a single {{define}} block produces
// exactly one symbol with the correct name, kind, and well-formed ranges.
func TestDocumentSymbolsSingleDefine(t *testing.T) {
	uri := "file:///sym-single.tmpl"
	src := `{{define "header"}}Hello World{{end}}`
	setDocFromSource(t, uri, src)
	defer store.Delete(uri)

	result, err := DocumentSymbols(nil, makeDocSymParams(uri))
	require.NoError(t, err)
	syms := castDocSymbols(t, result)
	require.Len(t, syms, 1)

	sym := syms[0]
	assert.Equal(t, "header", sym.Name)
	assert.Equal(t, protocol.SymbolKindFunction, sym.Kind)

	// Full range must span the whole define block.
	assert.LessOrEqual(t, sym.Range.Start.Line, sym.Range.End.Line)
	// Full range must end after it starts.
	assert.True(
		t,
		sym.Range.End.Character > sym.Range.Start.Character ||
			sym.Range.End.Line > sym.Range.Start.Line,
	)
}

// TestDocumentSymbolsMultipleDefines verifies that multiple {{define}} blocks are
// all reported, regardless of iteration order.
func TestDocumentSymbolsMultipleDefines(t *testing.T) {
	uri := "file:///sym-multi.tmpl"
	src := "{{define \"header\"}}top{{end}}\n{{define \"footer\"}}bottom{{end}}"
	setDocFromSource(t, uri, src)
	defer store.Delete(uri)

	result, err := DocumentSymbols(nil, makeDocSymParams(uri))
	require.NoError(t, err)
	syms := castDocSymbols(t, result)
	require.Len(t, syms, 2)

	names := []string{syms[0].Name, syms[1].Name}
	assert.ElementsMatch(t, []string{"header", "footer"}, names)
}

// TestDocumentSymbolsEmptyBody verifies that a {{define}} block with an empty body
// is still reported as a valid symbol.
func TestDocumentSymbolsEmptyBody(t *testing.T) {
	uri := "file:///sym-empty.tmpl"
	src := `{{define "empty"}}{{end}}`
	setDocFromSource(t, uri, src)
	defer store.Delete(uri)

	result, err := DocumentSymbols(nil, makeDocSymParams(uri))
	require.NoError(t, err)
	syms := castDocSymbols(t, result)
	require.Len(t, syms, 1)
	assert.Equal(t, "empty", syms[0].Name)
	assert.Equal(t, protocol.SymbolKindFunction, syms[0].Kind)
}

// TestDocumentSymbolsRangePositions verifies that the full range of a {{define}} block
// starts before (or at) the selection range and ends after (or at) it.
func TestDocumentSymbolsRangePositions(t *testing.T) {
	uri := "file:///sym-ranges.tmpl"
	src := `{{define "myBlock"}}some content{{end}}`
	setDocFromSource(t, uri, src)
	defer store.Delete(uri)

	result, err := DocumentSymbols(nil, makeDocSymParams(uri))
	require.NoError(t, err)
	syms := castDocSymbols(t, result)
	require.Len(t, syms, 1)

	sym := syms[0]
	fullStart := sym.Range.Start.Line*10000 + sym.Range.Start.Character
	fullEnd := sym.Range.End.Line*10000 + sym.Range.End.Character
	selStart := sym.SelectionRange.Start.Line*10000 + sym.SelectionRange.Start.Character
	selEnd := sym.SelectionRange.End.Line*10000 + sym.SelectionRange.End.Character

	assert.LessOrEqual(
		t,
		fullStart,
		selStart,
		"full range should start at or before selection range",
	)
	assert.GreaterOrEqual(t, fullEnd, selEnd, "full range should end at or after selection range")
}

// TestDocumentSymbolsMultilineDefine verifies range positions for a multi-line block.
func TestDocumentSymbolsMultilineDefine(t *testing.T) {
	uri := "file:///sym-multiline.tmpl"
	src := "{{define \"layout\"}}\n  <h1>{{.Title}}</h1>\n{{end}}"
	setDocFromSource(t, uri, src)
	defer store.Delete(uri)

	result, err := DocumentSymbols(nil, makeDocSymParams(uri))
	require.NoError(t, err)
	syms := castDocSymbols(t, result)
	require.Len(t, syms, 1)

	sym := syms[0]
	assert.Equal(t, "layout", sym.Name)
	// The block spans multiple lines so End.Line > Start.Line.
	assert.Greater(t, sym.Range.End.Line, sym.Range.Start.Line)
}

// TestDocumentSymbolsThreeDefines verifies that three defines are all reported.
func TestDocumentSymbolsThreeDefines(t *testing.T) {
	uri := "file:///sym-three.tmpl"
	src := "{{define \"a\"}}A{{end}}\n{{define \"b\"}}B{{end}}\n{{define \"c\"}}C{{end}}"
	setDocFromSource(t, uri, src)
	defer store.Delete(uri)

	result, err := DocumentSymbols(nil, makeDocSymParams(uri))
	require.NoError(t, err)
	syms := castDocSymbols(t, result)
	require.Len(t, syms, 3)

	names := make([]string, len(syms))
	for i, s := range syms {
		names[i] = s.Name
	}
	assert.ElementsMatch(t, []string{"a", "b", "c"}, names)
}

// TestSemanticTokensAdditionalCases covers node kinds not exercised by
// TestSemanticTokenNodeTypes, including else/if branches, pipes with assignment,
// and template calls with a pipeline argument.
func TestSemanticTokensAdditionalCases(t *testing.T) {
	lt := semanticOrderType(t)

	tests := []struct {
		name          string
		src           string
		noType        bool
		wantTypes     []uint32
		wantModifiers []uint32
	}{
		{
			// if/else if/else/end - the parser desugars {{else if}} into a
			// nested IfNode inside ElseList, so the shared {{end}} must appear
			// exactly once (7 tokens total, not 8).
			name:   "if/else-if/else/end keywords",
			src:    `{{if .}}{{else if .}}{{else}}{{end}}`,
			noType: true,
			wantTypes: []uint32{
				ttKeyword,  // if
				ttVariable, // .
				ttKeyword,  // else
				ttKeyword,  // if  (inner)
				ttVariable, // .  (inner condition)
				ttKeyword,  // else (inner)
				ttKeyword,  // end  (shared, emitted once)
			},
		},
		{
			// with/else/end
			name:      "with/else/end",
			src:       `{{with .Address}}{{else}}nothing{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword, ttKeyword},
		},
		{
			// range/else/end
			name:      "range/else/end",
			src:       `{{range .Items}}{{else}}empty{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword, ttKeyword},
		},
		{
			// pipeline with variable assignment: $x, $y := range ...
			name:          "range with two-variable assignment",
			src:           `{{range $i, $v := .Items}}{{end}}`,
			wantTypes:     []uint32{ttKeyword, ttVariable, ttVariable, ttProperty, ttKeyword},
			wantModifiers: []uint32{0, tmDeclaration, tmDeclaration, 0, 0},
		},
		{
			// negative number literal
			name:      "negative number",
			src:       `{{-1}}`,
			noType:    true,
			wantTypes: []uint32{ttNumber},
		},
		{
			// float number literal
			name:      "float number",
			src:       `{{3.14}}`,
			noType:    true,
			wantTypes: []uint32{ttNumber},
		},
		{
			// template call with a pipeline argument
			name:      "template with pipeline arg",
			src:       `{{template "sub" .}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword, ttString, ttVariable},
		},
		{
			// builtin call with two args
			name:          "index builtin with two args",
			src:           `{{index .Items 0}}`,
			wantTypes:     []uint32{ttFunction, ttProperty, ttNumber},
			wantModifiers: []uint32{tmDefaultLibrary, 0, 0},
		},
		{
			// multiple statements in one action (pipe)
			name:          "printf with variable",
			src:           `{{$s := printf "%d" 42}}`,
			noType:        true,
			wantTypes:     []uint32{ttVariable, ttFunction, ttString, ttNumber},
			wantModifiers: []uint32{tmDeclaration, tmDefaultLibrary, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "file:///add-tok-" + tt.name + ".tmpl"
			useLt := lt
			if tt.noType {
				useLt = nil
			}
			setDocWithTypedTree(t, uri, tt.src, useLt)
			defer store.Delete(uri)

			result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Data, len(tt.wantTypes)*5, "unexpected token count")

			for i, wantType := range tt.wantTypes {
				assert.Equal(t, wantType, result.Data[i*5+3], "token[%d] tokenType", i)
			}
			for i, wantMod := range tt.wantModifiers {
				assert.Equal(t, wantMod, result.Data[i*5+4], "token[%d] modifiers", i)
			}
		})
	}
}

// TestSemanticTokenNodeTypes covers each node kind handled by walkSemanticNode.
// Each case specifies the exact token types (result.Data[i*5+3]) and, when relevant,
// the token modifiers (result.Data[i*5+4]) expected in source order.
func TestSemanticTokenNodeTypes(t *testing.T) {
	lt := semanticOrderType(t)

	tests := []struct {
		name          string
		src           string
		noType        bool
		wantTypes     []uint32 // result.Data[i*5+3] for each token i
		wantModifiers []uint32 // result.Data[i*5+4]; nil means don't check
	}{
		{
			name:      "comment",
			src:       `{{/* hello */}}`,
			noType:    true,
			wantTypes: []uint32{ttComment},
		},
		{
			name:      "dot",
			src:       `{{.}}`,
			noType:    true,
			wantTypes: []uint32{ttVariable},
		},
		{
			name:          "variable declaration",
			src:           `{{$x := .}}`,
			noType:        true,
			wantTypes:     []uint32{ttVariable, ttVariable},
			wantModifiers: []uint32{tmDeclaration, 0},
		},
		{
			name:      "variable usage",
			src:       `{{$x := .}}{{$x}}`,
			noType:    true,
			wantTypes: []uint32{ttVariable, ttVariable, ttVariable},
		},
		{
			name:          "builtin identifier",
			src:           `{{len .}}`,
			noType:        true,
			wantTypes:     []uint32{ttFunction, ttVariable},
			wantModifiers: []uint32{tmDefaultLibrary, 0},
		},
		{
			name:      "user function identifier",
			src:       `{{printf "%s" "x"}}`,
			noType:    true,
			wantTypes: []uint32{ttFunction, ttString, ttString},
		},
		{
			name:      "string",
			src:       `{{"hello"}}`,
			noType:    true,
			wantTypes: []uint32{ttString},
		},
		{
			name:      "number",
			src:       `{{42}}`,
			noType:    true,
			wantTypes: []uint32{ttNumber},
		},
		{
			name:      "bool true",
			src:       `{{true}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword},
		},
		{
			name:      "bool false",
			src:       `{{false}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword},
		},
		{
			name:      "nil",
			src:       `{{nil}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword},
		},
		{
			name:      "if/end",
			src:       `{{if .}}{{end}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword, ttVariable, ttKeyword},
		},
		{
			name:      "if/else/end",
			src:       `{{if .}}{{else}}{{end}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword, ttVariable, ttKeyword, ttKeyword},
		},
		{
			// .Items is a []Item field on Order
			name:      "range/end",
			src:       `{{range .Items}}{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword},
		},
		{
			// .Address is a struct field on Order
			name:      "with/end",
			src:       `{{with .Address}}{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword},
		},
		{
			name:      "template",
			src:       `{{template "sub"}}`,
			noType:    true,
			wantTypes: []uint32{ttKeyword, ttString},
		},
		{
			name:      "break",
			src:       `{{range .Items}}{{break}}{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword, ttKeyword},
		},
		{
			name:      "continue",
			src:       `{{range .Items}}{{continue}}{{end}}`,
			wantTypes: []uint32{ttKeyword, ttProperty, ttKeyword, ttKeyword},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "file:///node-tok-" + tt.name + ".tmpl"
			useLt := lt
			if tt.noType {
				useLt = nil
			}
			setDocWithTypedTree(t, uri, tt.src, useLt)
			defer store.Delete(uri)

			result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Data, len(tt.wantTypes)*5, "unexpected token count")

			for i, wantType := range tt.wantTypes {
				assert.Equal(t, wantType, result.Data[i*5+3], "token[%d] tokenType", i)
			}
			for i, wantMod := range tt.wantModifiers {
				assert.Equal(t, wantMod, result.Data[i*5+4], "token[%d] modifiers", i)
			}
		})
	}
}

func TestElseIfNoDuplicateEndToken(t *testing.T) {
	uri := "file:///regression-elseif.tmpl"
	setDocWithTypedTree(t, uri, `{{if .}}{{else if .}}{{else}}{{end}}`, nil)
	defer store.Delete(uri)

	result, err := SemanticTokensFull(nil, makeSemanticParams(uri))
	require.NoError(t, err)
	require.NotNil(t, result)

	// 7 tokens: if . else if . else end  (the {{end}} must appear exactly once)
	require.Len(t, result.Data, 7*5, "{{end}} token must not be duplicated for else-if chains")

	// The last token must be the end keyword.
	assert.Equal(t, ttKeyword, result.Data[6*5+3], "last token must be the 'end' keyword")

	// Verify no two tokens share the same byte offset (no duplicate position).
	positions := make(map[uint32]bool)
	prevLine, prevChar := uint32(0), uint32(0)
	for i := 0; i < 7; i++ {
		deltaLine := result.Data[i*5+0]
		deltaChar := result.Data[i*5+1]
		if deltaLine == 0 {
			prevChar += deltaChar
		} else {
			prevLine += deltaLine
			prevChar = deltaChar
		}
		key := prevLine*100000 + prevChar
		assert.False(t, positions[key], "duplicate token position at token %d", i)
		positions[key] = true
	}
}

// TestDocumentSymbolsBlockStartRegression is a regression test for the bug where
// DocumentSymbols used FindStringIndex (returning the first {{define}} match in the
// file) instead of the last one before bodyStart, causing every define after the
// first to have its Range.Start set to the first define's position.
func TestDocumentSymbolsBlockStartRegression(t *testing.T) {
	uri := "file:///regression-blocksym.tmpl"
	// Three defines on separate lines; without the fix the second and third would
	// both report Range.Start.Line == 0 (the position of "alpha").
	src := "{{define \"alpha\"}}A{{end}}\n{{define \"beta\"}}B{{end}}\n{{define \"gamma\"}}C{{end}}"
	setDocFromSource(t, uri, src)
	defer store.Delete(uri)

	result, err := DocumentSymbols(nil, makeDocSymParams(uri))
	require.NoError(t, err)
	syms := castDocSymbols(t, result)
	require.Len(t, syms, 3)

	// Each symbol must start on a distinct line.
	lines := map[uint32]string{}
	for _, s := range syms {
		line := s.Range.Start.Line
		if prev, exists := lines[line]; exists {
			t.Errorf(
				"symbols %q and %q both have Range.Start.Line=%d (blockStart bug)",
				prev, s.Name, line,
			)
		}
		lines[line] = s.Name
	}
	assert.Len(t, lines, 3, "all three symbols must start on distinct lines")
}
