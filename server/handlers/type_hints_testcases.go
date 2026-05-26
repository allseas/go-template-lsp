package handlers

// ── ParseTypeHints test cases ────────────────────────────────────────────────

type parseTypeHintTestCase struct {
	name      string
	input     string
	wantHints []TypeHint
}

var parseTypeHintTestCases = []parseTypeHintTestCase{
	{
		name:      "empty input",
		input:     "",
		wantHints: nil,
	},
	{
		name:      "no gotype comment",
		input:     "plain text\n{{.Name}}",
		wantHints: nil,
	},
	{
		name:      "single hint",
		input:     "{{/*gotype: MyType*/}}",
		wantHints: []TypeHint{{Line: 1, Type: "MyType"}},
	},
	{
		name:      "hint with package path",
		input:     "{{/*gotype: pkg/sub.MyType*/}}",
		wantHints: []TypeHint{{Line: 1, Type: "pkg/sub.MyType"}},
	},
	{
		name:      "hint with trimming dashes and spaces",
		input:     "{{- /* gotype: Foo */ -}}",
		wantHints: []TypeHint{{Line: 1, Type: "Foo"}},
	},
	{
		name:      "hint on second line",
		input:     "first line\n{{/*gotype: Bar*/}}",
		wantHints: []TypeHint{{Line: 2, Type: "Bar"}},
	},
	{
		name:  "multiple hints on separate lines",
		input: "{{/*gotype: Type1*/}}\n{{/*gotype: Type2*/}}",
		wantHints: []TypeHint{
			{Line: 1, Type: "Type1"},
			{Line: 2, Type: "Type2"},
		},
	},
	{
		name:  "two hints on same line",
		input: "{{/*gotype: A*/}} {{/*gotype: B*/}}",
		wantHints: []TypeHint{
			{Line: 1, Type: "A"},
			{Line: 1, Type: "B"},
		},
	},
}

// ── splitTypeHint test cases ─────────────────────────────────────────────────

type splitTypeHintTestCase struct {
	name       string
	hint       string
	wantImport string
	wantType   string
}

var splitTypeHintTestCases = []splitTypeHintTestCase{
	{
		name:       "no package prefix",
		hint:       "MyType",
		wantImport: ".",
		wantType:   "MyType",
	},
	{
		name:       "simple package.Type",
		hint:       "pkg.MyType",
		wantImport: "pkg",
		wantType:   "MyType",
	},
	{
		name:       "full import path",
		hint:       "example.com/pkg/sub.MyType",
		wantImport: "example.com/pkg/sub",
		wantType:   "MyType",
	},
	{
		name:       "dot in package segment with slash after",
		hint:       "a.b/c.Type",
		wantImport: "a.b/c",
		wantType:   "Type",
	},
	{
		name:       "multiple dots, last wins",
		hint:       "a.b.Type",
		wantImport: "a.b",
		wantType:   "Type",
	},
}

// ── LoadTypeFromHint test cases ──────────────────────────────────────────────

type loadTypeHintTestCase struct {
	name         string
	hint         string
	root         string
	wantErr      bool
	wantTypeName string
	wantFields   []string
	wantMethods  []string
}

var loadTypeHintTestCases = []loadTypeHintTestCase{
	{
		name:         "loads Order type with fields and methods",
		hint:         "text-template-server/src/model.Order",
		root:         "",
		wantTypeName: "Order",
		wantFields:   []string{"ID", "CustomerName", "Email", "Address", "Items", "TotalAmount", "Paid"},
		wantMethods:  []string{"DisplayName", "Summary", "ItemCount", "IsLargeOrder", "Format"},
	},
	{
		name:    "returns error for invalid import path",
		hint:    "nonexistent/package.Foo",
		root:    "",
		wantErr: true,
	},
	{
		name:    "returns error when type not found in package",
		hint:    "text-template-server/src/model.NonExistent",
		root:    "",
		wantErr: true,
	},
}
