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
		name:  "slice hint",
		input: "{{/*gotype: []*cg/model/controlmodel.AlarmInstance*/}}",
		wantHints: []TypeHint{
			{Type: typeHintStruct, Text: "[]*cg/model/controlmodel.AlarmInstance", Line: 1},
		},
	},
	{
		name:      "go map type hint uses brackets and is a struct hint",
		input:     "{{/*gotype: map[string]int*/}}",
		wantHints: []TypeHint{{Type: typeHintStruct, Text: "map[string]int", Line: 1}},
	},
	{
		name:  "nested map of slice of pointer hint",
		input: "{{- /*gotype: map[string][]*cg/model/controlmodel.AlarmInstance*/ -}}",
		wantHints: []TypeHint{
			{
				Type: typeHintStruct,
				Text: "map[string][]*cg/model/controlmodel.AlarmInstance",
				Line: 1,
			},
		},
	},
	{
		name:  "dict hint with slice and map values",
		input: `{{/*gotype: map{"subsystem": string, "alarms": []*cg/model/controlmodel.AlarmInstance, "index": map[string]int}*/}}`,
		wantHints: []TypeHint{
			{
				Type: typeHintDict,
				Text: `map{"subsystem": string, "alarms": []*cg/model/controlmodel.AlarmInstance, "index": map[string]int}`,
				Line: 1,
			},
		},
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
			Text: `map{"Order": example.com/m.Order}`,
			Line: 1,
		}},
	},
	{
		name:  "dict hint with multiple entries",
		input: `{{/*gotype: map{"Order": example.com/m.Order, "Address": example.com/m.Address}*/}}`,
		wantHints: []TypeHint{{
			Type: typeHintDict,
			Text: `map{"Order": example.com/m.Order, "Address": example.com/m.Address}`,
			Line: 1,
		}},
	},
	{
		name:  "dict hint tolerates extra whitespace around tokens",
		input: `{{- /* gotype: map{  "A" : pkg.T ,  "B" : other/pkg.U } */ -}}`,
		wantHints: []TypeHint{{
			Type: typeHintDict,
			Text: `map{  "A" : pkg.T ,  "B" : other/pkg.U }`,
			Line: 1,
		}},
	},
	{
		name:  "dict hint on a define block",
		input: "{{- define \"Tpl\" -}}\n" + `{{- /*gotype: map{"K": ex.com/m.K}*/ -}}` + "\n{{- end -}}\n",
		wantHints: []TypeHint{{
			Type: typeHintDict,
			Text: `map{"K": ex.com/m.K}`,
			Line: 2,
		}},
	},
	{
		name:  "generic hint with a dict type argument is a struct hint",
		input: `{{/*gotype: pkg/tmpl.View[map{"g": pkg/other.Gadget}]*/}}`,
		wantHints: []TypeHint{{
			Type: typeHintStruct,
			Text: `pkg/tmpl.View[map{"g": pkg/other.Gadget}]`,
			Line: 1,
		}},
	},
	{
		name:  "dict nested inside a composite is a struct hint",
		input: `{{/*gotype: *[]map{"g": pkg/other.Gadget}*/}}`,
		wantHints: []TypeHint{{
			Type: typeHintStruct,
			Text: `*[]map{"g": pkg/other.Gadget}`,
			Line: 1,
		}},
	},
	{
		name:  "dict with a nested dict value",
		input: `{{/*gotype: map{"outer": map{"inner": string}}*/}}`,
		wantHints: []TypeHint{{
			Type: typeHintDict,
			Text: `map{"outer": map{"inner": string}}`,
			Line: 1,
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

// LoadTypeFromHint test cases

type loadTypeHintTestCase struct {
	name           string
	hint           string
	root           string
	wantErr        bool
	wantTypeName   string
	wantTypeString string
	wantFields     []string
	wantMethods    []string
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
		name:           "loads pointer to Order, unwrapping fields and methods",
		hint:           "*text-template-server/src/model.Order",
		root:           "../../test/resources/typehints-tests",
		wantTypeName:   "Order",
		wantTypeString: "*text-template-server/src/model.Order",
		wantFields:     []string{"ID", "Address", "Items"},
		wantMethods:    []string{"DisplayName", "ItemCount"},
	},
	{
		name:           "loads builtin int",
		hint:           "int",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "int",
	},
	{
		name:           "loads pointer to builtin string",
		hint:           "*string",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "*string",
	},
	{
		name: "loads alias to a builtin (non-named type)",
		hint: "text-template-server/src/model.Fahrenheit",
		root: "../../test/resources/typehints-tests",
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
	// Composite type expressions: slices, arrays, maps, pointers and nesting.
	{
		name:           "loads slice of named type",
		hint:           "[]text-template-server/src/model.Item",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "[]text-template-server/src/model.Item",
	},
	{
		name:           "loads slice of pointer to named type",
		hint:           "[]*text-template-server/src/model.Item",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "[]*text-template-server/src/model.Item",
	},
	{
		name:           "loads slice of builtin",
		hint:           "[]string",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "[]string",
	},
	{
		name:           "loads fixed-size array of builtin",
		hint:           "[3]int",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "[3]int",
	},
	{
		name:           "loads map of builtins",
		hint:           "map[string]int",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "map[string]int",
	},
	{
		name:           "loads map with named value type",
		hint:           "map[string]text-template-server/src/model.Order",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "map[string]text-template-server/src/model.Order",
	},
	{
		name:           "loads nested map of slice of pointer to named type",
		hint:           "map[string][]*text-template-server/src/model.Item",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "map[string][]*text-template-server/src/model.Item",
	},
	{
		name:           "loads pointer to map",
		hint:           "*map[int]text-template-server/src/model.Address",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "*map[int]text-template-server/src/model.Address",
	},
	{
		name:           "loads map keyed by named type",
		hint:           "map[text-template-server/src/model.Address]string",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "map[text-template-server/src/model.Address]string",
	},
	{
		name:           "loads parenthesized builtin",
		hint:           "(int)",
		root:           "../../test/resources/typehints-tests",
		wantTypeString: "int",
	},
	{
		name:    "returns error for bare local type when root has no package",
		hint:    "LocalOnly",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
	{
		name:    "returns error for slice of unresolvable package",
		hint:    "[]nonexistent/package.Foo",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
	{
		name:    "returns error for map value that is not a type",
		hint:    "map[string]fmt.Println",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
	{
		name:    "returns error for a non-type literal",
		hint:    "123",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
	{
		name:    "returns error for a malformed type expression",
		hint:    "map[int]",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
	{
		name:    "returns error for conflicting import path segments",
		hint:    "map[a/model.Address]b/model.Order",
		root:    "../../test/resources/typehints-tests",
		wantErr: true,
	},
}
