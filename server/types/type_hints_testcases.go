package types

// ParseTypeHints test cases

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
		wantHints: []TypeHint{{Type: typeHintStruct, Text: "MyType", Line: 1}},
	},
	{
		name:      "hint with package path",
		input:     "{{/*gotype: pkg/sub.MyType*/}}",
		wantHints: []TypeHint{{Type: typeHintStruct, Text: "pkg/sub.MyType", Line: 1}},
	},
	{
		name:      "hint with trimming dashes and spaces",
		input:     "{{- /* gotype: Foo */ -}}",
		wantHints: []TypeHint{{Type: typeHintStruct, Text: "Foo", Line: 1}},
	},
	{
		name:      "hint on second line",
		input:     "first line\n{{/*gotype: Bar*/}}",
		wantHints: []TypeHint{{Type: typeHintStruct, Text: "Bar", Line: 2}},
	},
	{
		name:  "multiple hints on separate lines",
		input: "{{/*gotype: Type1*/}}\n{{/*gotype: Type2*/}}",
		wantHints: []TypeHint{
			{Type: typeHintStruct, Text: "Type1", Line: 1},
		},
	},
	{
		name:  "two hints on same line",
		input: "{{/*gotype: A*/}} {{/*gotype: B*/}}",
		wantHints: []TypeHint{
			{Type: typeHintStruct, Text: "A", Line: 1},
		},
	},
	{
		name:      "contains gotype marker but no valid type token",
		input:     "{{/*gotype: 123*/}}",
		wantHints: nil,
	},
	{
		name: "multiple defines each with their own gotype hint",
		input: "{{- define \"OrderTpl\" -}}\n" +
			"{{- /*gotype: example.com/m.Order*/ -}}\n" +
			"body\n" +
			"{{- end -}}\n" +
			"{{- define \"AddressTpl\" -}}\n" +
			"{{- /*gotype: example.com/m.Address*/ -}}\n" +
			"body\n" +
			"{{- end -}}\n",
		wantHints: []TypeHint{
			{Type: typeHintStruct, Text: "example.com/m.Order", Line: 2},
			{Type: typeHintStruct, Text: "example.com/m.Address", Line: 6},
		},
	},
	{
		name: "multiple defines with one missing the gotype hint",
		input: "{{- define \"OrderTpl\" -}}\n" +
			"{{- /*gotype: example.com/m.Order*/ -}}\n" +
			"{{- end -}}\n" +
			"{{- define \"NoHint\" -}}\n" +
			"no hint here\n" +
			"{{- end -}}\n",
		wantHints: []TypeHint{
			{Type: typeHintStruct, Text: "example.com/m.Order", Line: 2},
		},
	},
	{
		name:  "dict hint with a single entry",
		input: `{{/*gotype: map{"Order": example.com/m.Order}*/}}`,
		wantHints: []TypeHint{{
			Type: typeHintDict,
			Text: `"Order": example.com/m.Order`,
			Dict: map[string]string{"Order": "example.com/m.Order"},
			Line: 1,
		}},
	},
	{
		name:  "dict hint with multiple entries",
		input: `{{/*gotype: map{"Order": example.com/m.Order, "Address": example.com/m.Address}*/}}`,
		wantHints: []TypeHint{{
			Type: typeHintDict,
			Text: `"Order": example.com/m.Order, "Address": example.com/m.Address`,
			Dict: map[string]string{
				"Order":   "example.com/m.Order",
				"Address": "example.com/m.Address",
			},
			Line: 1,
		}},
	},
	{
		name:  "dict hint tolerates extra whitespace around tokens",
		input: `{{- /* gotype: map{  "A" : pkg.T ,  "B" : other/pkg.U } */ -}}`,
		wantHints: []TypeHint{{
			Type: typeHintDict,
			Text: `"A" : pkg.T ,  "B" : other/pkg.U`,
			Dict: map[string]string{
				"A": "pkg.T",
				"B": "other/pkg.U",
			},
			Line: 1,
		}},
	},
	{
		name:  "dict hint on a define block",
		input: "{{- define \"Tpl\" -}}\n" + `{{- /*gotype: map{"K": ex.com/m.K}*/ -}}` + "\n{{- end -}}\n",
		wantHints: []TypeHint{{
			Type: typeHintDict,
			Text: `"K": ex.com/m.K`,
			Dict: map[string]string{"K": "ex.com/m.K"},
			Line: 2,
		}},
	},
	{
		name:      "dict hint with an empty body",
		input:     `{{/*gotype: map{}*/}}`,
		wantHints: []TypeHint{{Type: typeHintMalformedDict, Line: 1}},
	},
	{
		name:      "dict hint missing the closing brace is not accepted",
		input:     `{{/*gotype: map{"Order": example.com/m.Order*/}}`,
		wantHints: []TypeHint{{Type: typeHintMalformedDict, Line: 1}},
	},
	{
		name:      "dict hint with an unquoted key is not accepted",
		input:     `{{/*gotype: map{Order: example.com/m.Order}*/}}`,
		wantHints: []TypeHint{{Type: typeHintMalformedDict, Line: 1}},
	},
	{
		name:      "dict hint with a missing colon is not accepted",
		input:     `{{/*gotype: map{"Order" example.com/m.Order}*/}}`,
		wantHints: []TypeHint{{Type: typeHintMalformedDict, Line: 1}},
	},
	{
		name:      "dict hint with a missing type reference is not accepted",
		input:     `{{/*gotype: map{"Order": }*/}}`,
		wantHints: []TypeHint{{Type: typeHintMalformedDict, Line: 1}},
	},
}

// splitTypeHint test cases

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

// LoadTypeFromHint test cases

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
		root:         "../../test/resources/typehints-tests",
		wantTypeName: "Order",
		wantFields: []string{
			"ID",
			"CustomerName",
			"Email",
			"Address",
			"Items",
			"TotalAmount",
			"Paid",
		},
		wantMethods: []string{"DisplayName", "Summary", "ItemCount", "IsLargeOrder", "Format"},
	},
	{
		name:    "returns error for invalid import path",
		hint:    "nonexistent/package.Foo",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
	{
		name:    "returns error when type not found in package",
		hint:    "text-template-server/src/model.NonExistent",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
	{
		name:    "returns error when symbol is not a named type",
		hint:    "fmt.Println",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
}
