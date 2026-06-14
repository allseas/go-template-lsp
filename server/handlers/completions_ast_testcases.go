package handlers

type completionTestCase struct {
	name        string
	src         string
	subStr      string
	occurrence  int
	offsetAdj   int
	isInvoked   bool
	withType    bool
	contains    []string
	notContains []string
}

var chainEditTestCases = []completionTestCase{
	// FieldNode cases
	{
		name:       "FieldNode mid-chain - suggest fields/methods of preceding .Address",
		src:        `{{ .Address..Street }}`,
		subStr:     ".",
		occurrence: 1, // dot between Address and Street
		withType:   true,
		contains:   []string{"Street", "City", "Country", "Zip", "Line", "IsLocal", "ZipCode"},
		notContains: []string{
			"ID",
			"CustomerName",
			"Email",
			"Address",
			"Items",
			"TotalAmount",
			"Paid",
			"DisplayName",
			"ItemCount",
			"IsLargeOrder",
		},
	},
	// ChainNode cases
	{
		name:       "ChainNode mid-chain - suggest fields/methods of preceding (.Address)",
		src:        `{{ (.Address)..Street }}`,
		subStr:     ".",
		occurrence: 1, // dot between ) and Street
		withType:   true,
		contains:   []string{"Street", "City", "Country", "Zip", "Line", "IsLocal", "ZipCode"},
		notContains: []string{
			"ID",
			"CustomerName",
			"Email",
			"Address",
			"Items",
			"TotalAmount",
			"Paid",
			"DisplayName",
			"ItemCount",
			"IsLargeOrder",
		},
	},
	// VariableNode cases
	{
		name:       "VariableNode mid-chain - suggest fields/methods of preceding $t",
		src:        `{{ $t := .Address }}{{ $t..Street }}`,
		subStr:     "$t.",
		occurrence: 0,
		offsetAdj:  2, // skip "$t" to land on the dot before Street
		withType:   true,
		contains:   []string{"Street", "City", "Country", "Zip", "Line", "IsLocal", "ZipCode"},
		notContains: []string{
			"ID",
			"CustomerName",
			"Email",
			"Address",
			"Items",
			"TotalAmount",
			"Paid",
			"DisplayName",
			"ItemCount",
			"IsLargeOrder",
		},
	},
	{
		name:        "VariableNode three segments - only Address fields whose path contains nested .Length",
		src:         `{{ $t := .Address }}{{ $t..Street.Length }}`,
		subStr:      "$t.",
		occurrence:  0,
		offsetAdj:   2, // dot between $t and Street
		withType:    true,
		contains:    []string{"Street"},
		notContains: []string{"Address", "Items"},
	},
	{
		name:       "$. at root scope - root fields and methods",
		src:        `{{ $. }}`,
		subStr:     ".",
		occurrence: 0,
		withType:   true,
		contains: []string{
			"ID", "CustomerName", "Address", "Items",
			"DisplayName", "ItemCount", "IsLargeOrder",
		},
		notContains: []string{"Street", "City", "Line", "ZipCode"},
	},
	{
		name:       "$.Cust mid-typing at root scope - root fields filtered",
		src:        `{{ $.Cust }}`,
		subStr:     ".",
		occurrence: 0,
		withType:   true,
		contains: []string{
			"CustomerName", "ID", "Address",
			"DisplayName",
		},
		notContains: []string{"Street", "City", "Line", "ZipCode"},
	},
	{
		name:       "$.Address. - fields of Address (sub-chain via dot trigger)",
		src:        `{{ $.Address. }}`,
		subStr:     ".",
		occurrence: 1,
		withType:   true,
		contains: []string{
			"Street", "City", "Country", "Zip",
			"Line", "IsLocal", "ZipCode",
		},
		notContains: []string{"ID", "CustomerName", "DisplayName"},
	},
	{
		name:       "$. inside with - root fields, not the rebound dot's fields",
		src:        `{{ with .Address }}{{ $. }}{{ end }}`,
		subStr:     ".",
		occurrence: 1,
		withType:   true,
		contains: []string{
			"ID", "CustomerName", "Address", "Items",
			"DisplayName", "ItemCount",
		},
		notContains: []string{"Street", "City", "Line", "ZipCode"},
	},
	{
		name:       "$.Cust mid-typing inside with - root fields filtered, not Address",
		src:        `{{ with .Address }}{{ $.Cust }}{{ end }}`,
		subStr:     ".",
		occurrence: 1, // the dot of `$.Cust`
		withType:   true,
		contains: []string{
			"CustomerName", "ID", "Address",
			"DisplayName",
		},
		notContains: []string{"Street", "City", "Line", "ZipCode"},
	},
	{
		name:       "$. inside range - root fields, not the iterated element's",
		src:        `{{ range .Items }}{{ $. }}{{ end }}`,
		subStr:     ".",
		occurrence: 1, // the dot of `$.`
		withType:   true,
		contains: []string{
			"ID", "CustomerName", "Address", "Items",
			"DisplayName", "ItemCount",
		},
		notContains: []string{"SKU", "Qty", "UnitPrice", "Label", "Total"},
	},
	{
		name:       "$.Address. inside with - Address fields via root $, not rebound dot",
		src:        `{{ with .Address }}{{ $.Address. }}{{ end }}`,
		subStr:     ".",
		occurrence: 2, // the trailing dot after $.Address
		withType:   true,
		contains:   []string{"Street", "City", "Country", "Zip", "Line", "IsLocal", "ZipCode"},
		notContains: []string{
			"ID", "CustomerName", "DisplayName", "ItemCount",
		},
	},
	{
		name:       "$.Address. inside range - Address fields via root $, not range element",
		src:        `{{ range .Items }}{{ $.Address. }}{{ end }}`,
		subStr:     ".",
		occurrence: 2, // the trailing dot after $.Address
		withType:   true,
		contains:   []string{"Street", "City", "Country", "Zip", "Line", "IsLocal", "ZipCode"},
		notContains: []string{
			"SKU", "Qty", "UnitPrice", "Label", "Total",
			"ID", "CustomerName", "DisplayName",
		},
	},
	{
		name:       "$. inside block — root fields, not the rebound dot's fields",
		src:        `{{ block "csv" .Address }}{{ $. }}{{ end }}`,
		subStr:     ".",
		occurrence: 1, // the dot in $. inside block body
		withType:   true,
		contains: []string{
			"ID", "CustomerName", "Address", "Items",
			"DisplayName", "ItemCount",
		},
		notContains: []string{"Street", "City", "Line", "ZipCode"},
	},
	{
		name:       "$.Address. inside block — Address fields via root $, not block's dot",
		src:        `{{ block "csv" .Items }}{{ $.Address. }}{{ end }}`,
		subStr:     ".",
		occurrence: 2, // the trailing dot after $.Address
		withType:   true,
		contains:   []string{"Street", "City", "Country", "Zip", "Line", "IsLocal", "ZipCode"},
		notContains: []string{
			"ID", "CustomerName", "DisplayName",
		},
	},
}

var completionTestCases = []completionTestCase{
	{
		name:        "chain node (.Address). - Address fields suggested",
		src:         `{{ (.Address). }}`,
		subStr:      ".",
		occurrence:  1,
		withType:    true,
		contains:    []string{"Street", "City", "Country", "Zip"},
		notContains: []string{"ID", "CustomerName", "len", "eq"},
	},
	{
		name:        "chain node (.Address). - Address methods suggested",
		src:         `{{ (.Address). }}`,
		subStr:      ".",
		occurrence:  1,
		withType:    true,
		contains:    []string{"Line", "IsLocal", "ZipCode"},
		notContains: []string{"DisplayName", "ItemCount"},
	},
	{
		name:        "field chain .Address. - Address fields suggested",
		src:         `{{ .Address. }}`,
		subStr:      ".",
		occurrence:  1,
		withType:    true,
		contains:    []string{"Street", "City", "Country", "Zip", "Line", "IsLocal", "ZipCode"},
		notContains: []string{"ID", "CustomerName", "len", "eq"},
	},
	{
		name:        "field chain .Address. - Address methods suggested",
		src:         `{{ .Address. }}`,
		subStr:      ".",
		occurrence:  1,
		withType:    true,
		contains:    []string{"Line", "IsLocal", "ZipCode"},
		notContains: []string{"DisplayName", "ItemCount"},
	},
	{
		name:        "field chain .Address. - no dot prefix on completions",
		src:         `{{ .Address. }}`,
		subStr:      ".",
		occurrence:  1,
		withType:    true,
		notContains: []string{".Street", ".City", ".Line"},
	},
	{
		name:        "field chain on primitive type - no suggestions",
		src:         `{{ .CustomerName. }}`,
		subStr:      ".",
		occurrence:  1,
		withType:    true,
		notContains: []string{"ID", "CustomerName", "DisplayName", "Street", "len"},
	},
	// dot field completions
	{
		name:     "dot triggers Order field completions",
		src:      `{{.}}`,
		subStr:   ".",
		withType: true,
		contains: []string{
			"ID",
			"CustomerName",
			"Email",
			"Address",
			"Items",
			"TotalAmount",
			"Paid",
		},
	},
	{
		name:        "dot does not include builtins",
		src:         `{{.}}`,
		subStr:      ".",
		withType:    true,
		notContains: []string{"len", "eq", "html"},
	},
	// dot method completions
	{
		name:     "dot returns usable method names without dot prefix",
		src:      `{{.}}`,
		subStr:   ".",
		withType: true,
		contains: []string{"DisplayName", "Summary", "ItemCount", "IsLargeOrder", "Format"},
	},
	{
		name:        "dot excludes non-usable methods",
		src:         `{{.}}`,
		subStr:      ".",
		withType:    true,
		notContains: []string{"wrongSecond", "badReturn"},
	},
	{
		name:     "dot returns methods and fields together",
		src:      `{{.}}`,
		subStr:   ".",
		withType: true,
		contains: []string{"ID", "CustomerName", "Paid", "DisplayName", "ItemCount"},
	},
	{
		name:     "general context returns dot-prefixed methods and fields",
		src:      `{{len .}}`,
		subStr:   "l",
		withType: true,
		contains: []string{
			".DisplayName",
			".Summary",
			".ItemCount",
			".IsLargeOrder",
			".Format",
			".ID",
			".CustomerName",
			".Paid",
		},
		notContains: []string{".wrongSecond", ".badReturn"},
	},
	{
		name:        "dot-prefixed completions absent when no loaded type",
		src:         `{{len .}}`,
		subStr:      "l",
		notContains: []string{".DisplayName", ".ItemCount"},
	},
	// pipe filtering - model fields
	{
		name:        "string field piped - string-accepting builtins suggested",
		src:         `{{.CustomerName | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "js", "urlquery", "len"},
		notContains: []string{"not", "and"},
	},
	{
		name:        "bool field piped - bool-accepting builtins suggested",
		src:         `{{.Paid | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"not", "and", "or"},
		notContains: []string{"html", "len"},
	},
	{
		name:      "float field piped - outputUntyped, all builtins shown",
		src:       `{{.TotalAmount | }}`,
		subStr:    "}}",
		offsetAdj: -1,
		isInvoked: true,
		withType:  true,
		contains:  []string{"len", "html", "eq"},
	},
	{
		name:      "struct field piped - outputUntyped, all builtins shown",
		src:       `{{.Address | }}`,
		subStr:    "}}",
		offsetAdj: -1,
		isInvoked: true,
		withType:  true,
		contains:  []string{"len", "html", "and"},
	},
	// pipe filtering - model methods
	{
		name:        "string-returning method piped - string-accepting builtins",
		src:         `{{.DisplayName | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "js", "len"},
		notContains: []string{".Format", "not"},
	},
	{
		name:        "int-returning method piped - int-accepting builtins",
		src:         `{{.ItemCount | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"eq", "lt", "gt"},
		notContains: []string{"html", "not"},
	},
	{
		name:        "bool-returning method piped - bool-accepting builtins",
		src:         `{{.IsLargeOrder | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"not", "and"},
		notContains: []string{"len", "html"},
	},
	{
		name:        "string-returning method with arg piped - string builtins",
		src:         `{{.Format "$" | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "len"},
		notContains: []string{"not"},
	},
	// user method pipe filtering
	{
		name:      "string field piped - no dot-prefixed methods",
		src:       `{{.CustomerName | }}`,
		subStr:    "}}",
		offsetAdj: -1,
		isInvoked: true,
		withType:  true,
		notContains: []string{
			".Format",
			".DisplayName",
			".ItemCount",
			".IsLargeOrder",
			".wrongSecond",
		},
	},
	{
		name:        "bool field piped - no dot-prefixed methods",
		src:         `{{.Paid | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{".Format", ".DisplayName"},
	},
	{
		name:        "identifier in pipe - no dot-prefixed methods",
		src:         `{{.ItemCount | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{".Format", ".DisplayName"},
	},
	{
		name:        "CustomerName pipe - bare method names absent",
		src:         `{{.CustomerName | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{"DisplayName", "Format", "wrongSecond"},
	},
	{
		name:        "Paid pipe - bare method names absent",
		src:         `{{.Paid | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{"DisplayName", "Format", "wrongSecond"},
	},
	{
		name:        "identifier pipe - bare method names absent",
		src:         `{{.ItemCount | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{"DisplayName", "Format", "wrongSecond"},
	},
	// slice field pipe
	{
		name:        "slice field piped - len/index/slice shown, no methods",
		src:         `{{.Items | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"len", "index", "slice"},
		notContains: []string{".DisplayName", ".ItemCount", "DisplayName", "ItemCount"},
	},
	// multi-stage pipe chaining
	{
		name:        "html after string field - string builtins only",
		src:         `{{.CustomerName | html | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"len", "js"},
		notContains: []string{"not", "eq"},
	},
	{
		name:        "not after bool field - bool builtins only",
		src:         `{{.Paid | not | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"and", "or"},
		notContains: []string{"len", "html", ".Oper"},
	},
	// dot piped directly
	{
		name:        "dot piped - all builtins shown, no dot-prefixed fields",
		src:         `{{. | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"len", "html", "and"},
		notContains: []string{".", ".Address", ".Items", ".ID"},
	},
	{
		name:        "struct field piped - dot-prefixed fields excluded",
		src:         `{{.Address | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"len"},
		notContains: []string{".Address", ".Items", ".ID"},
	},
	// builtin chained with model
	{
		name:      "len of items piped - int-accepting builtins",
		src:       `{{.Items | len | }}`,
		subStr:    "}}",
		offsetAdj: -1,
		isInvoked: true,
		withType:  true,
		contains:  []string{"eq", "lt", "print"},
		notContains: []string{
			"html",
			"not",
			".Oper",
			".DisplayName",
			".ItemCount",
			".IsLargeOrder",
		},
	},
	{
		name:        "html of string field piped - string builtins",
		src:         `{{.CustomerName | html | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"len", "js"},
		notContains: []string{"not", "eq"},
	},
	// invoked vs non-invoked
	{
		name:        "invoked after string pipe - string builtins, not bool",
		src:         `{{.CustomerName | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "len"},
		notContains: []string{"not"},
	},
	// scope switch - range
	{
		name:        "inside range - dot trigger returns Item fields not Order fields",
		src:         `{{range .Items}}{{.}}{{end}}`,
		subStr:      ".",
		occurrence:  1,
		withType:    true,
		contains:    []string{"SKU", "Name"},
		notContains: []string{"ID", "CustomerName", "Address", "Items"},
	},
	{
		name:        "inside range - dot-prefixed Item methods, Order methods absent",
		src:         `{{range .Items}}{{len .SKU}}{{end}}`,
		subStr:      "l",
		withType:    true,
		contains:    []string{".IsExpensive", ".Describe", ".Label"},
		notContains: []string{".DisplayName", ".IsLargeOrder", ".wrongSecond"},
	},
	{
		name:       "inside range pipe - Item methods excluded, none accept string",
		src:        `{{range .Items}}{{.Label | }}{{end}}`,
		subStr:     "}}",
		occurrence: 1,
		offsetAdj:  -1,
		isInvoked:  true,
		withType:   true,
		notContains: []string{
			".IsExpensive",
			".Describe",
			".Label",
			".DisplayName",
			".IsLargeOrder",
		},
	},
	{
		name:        "inside range - string Item method piped, string builtins",
		src:         `{{range .Items}}{{.Label | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "len"},
		notContains: []string{"not", "eq"},
	},
	{
		name:        "inside range - bool Item method piped, bool builtins",
		src:         `{{range .Items}}{{IsExpensive | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"not", "and"},
		notContains: []string{"html", "len"},
	},
	// scope switch - with
	{
		name:        "inside with - dot-prefixed Address methods, Order methods absent",
		src:         `{{with .Address}}{{len .Street}}{{end}}`,
		subStr:      "l",
		withType:    true,
		contains:    []string{".Line", ".IsLocal", ".ZipCode"},
		notContains: []string{".DisplayName", ".IsLargeOrder", ".wrongSecond"},
	},
	{
		name:        "inside with pipe - Address methods excluded, none accept string",
		src:         `{{with .Address}}{{Line | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{".Line", ".IsLocal", ".ZipCode", ".DisplayName", ".IsLargeOrder"},
	},
	{
		name:        "inside with - string Address method piped, string builtins",
		src:         `{{with .Address}}{{Line | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "len"},
		notContains: []string{"not", "eq"},
	},
	{
		name:        "inside with - bool Address method piped, bool builtins",
		src:         `{{with .Address}}{{IsLocal | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"not", "and"},
		notContains: []string{"html", "len"},
	},
	// scope switch — block (table)
	{
		name:        "inside block — dot trigger returns Address fields not Order fields",
		src:         `{{block "csv" .Address}}{{.}}{{end}}`,
		subStr:      ".",
		occurrence:  1,
		withType:    true,
		contains:    []string{"Street", "City", "Country", "Zip"},
		notContains: []string{"ID", "CustomerName", "Items"},
	},
	{
		name:        "inside block — dot-prefixed Address methods, Order methods absent",
		src:         `{{block "csv" .Address}}{{len .Street}}{{end}}`,
		subStr:      "l",
		occurrence:  1,
		withType:    true,
		contains:    []string{".Line", ".IsLocal", ".ZipCode"},
		notContains: []string{".DisplayName", ".IsLargeOrder", ".wrongSecond"},
	},
	{
		name:        "inside block pipe — Address methods excluded, none accept string",
		src:         `{{block "csv" .Address}}{{Line | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{".Line", ".IsLocal", ".ZipCode", ".DisplayName", ".IsLargeOrder"},
	},
	{
		name:        "inside block — string Address method piped, string builtins",
		src:         `{{block "csv" .Address}}{{Line | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "len"},
		notContains: []string{"not", "eq"},
	},
	{
		name:        "inside block — bool Address method piped, bool builtins",
		src:         `{{block "csv" .Address}}{{IsLocal | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"not", "and"},
		notContains: []string{"html", "len"},
	},
	// dot suggestions (no type)
	{
		name:        "dot in if condition - no builtins",
		src:         `{{if .}}{{end}}`,
		subStr:      ".",
		notContains: []string{"eq", "len"},
	},
	{
		name:        "dot in range pipeline - no builtins",
		src:         `{{range .}}{{end}}`,
		subStr:      ".",
		notContains: []string{"eq", "len"},
	},
	{
		name:        "dot in with pipeline - no builtins",
		src:         `{{with .}}{{end}}`,
		subStr:      ".",
		notContains: []string{"eq", "len"},
	},
	{
		name:        "sChar dot - dot item returned, not builtins",
		src:         `{{.}}`,
		subStr:      ".",
		notContains: []string{"eq", "len"},
	},
	// variable suggestions
	{
		name:        "dollar sChar - vars returned without sigil",
		src:         `{{$top := .}}{{$}}`,
		subStr:      "$",
		occurrence:  1,
		contains:    []string{"top"},
		notContains: []string{"$top"},
	},
	{
		name:     "non-dollar sChar - full $var label included",
		src:      `{{$top := .}}{{len .}}`,
		subStr:   "l",
		contains: []string{"$top"},
	},
	{
		name:       "variable declared before cursor is visible",
		src:        `{{$x := .}}{{$x}}`,
		subStr:     "$",
		occurrence: 1,
		contains:   []string{"x"},
	},
	{
		name:        "variable declared after cursor is not visible",
		src:         `{{$early := .}}{{$}}{{$late := .}}`,
		subStr:      "$",
		occurrence:  1,
		contains:    []string{"early"},
		notContains: []string{"late", "$late"},
	},
	{
		name:       "range index and value variables visible inside body",
		src:        `{{range $i, $v := .}}{{$}}{{end}}`,
		subStr:     "$",
		occurrence: 2,
		contains:   []string{"i", "v"},
	},
	{
		name:        "range variable not visible after end",
		src:         `{{range $inner := .}}{{end}}{{$}}`,
		subStr:      "$",
		occurrence:  1,
		notContains: []string{"inner", "$inner"},
	},
	{
		name:       "outer variable visible inside nested range",
		src:        `{{$outer := .}}{{range $i := .}}{{range $j := .}}{{$}}{{end}}{{end}}`,
		subStr:     "$",
		occurrence: 3,
		contains:   []string{"outer", "i", "j"},
	},
	{
		name:       "if condition variable visible inside block",
		src:        `{{if $cond := .}}{{$}}{{end}}`,
		subStr:     "$",
		occurrence: 1,
		contains:   []string{"cond"},
	},
	{
		name:       "with variable visible inside block",
		src:        `{{with $w := .}}{{$}}{{end}}`,
		subStr:     "$",
		occurrence: 1,
		contains:   []string{"w"},
	},
	// builtin suggestions
	{
		name:   "builtins appear in general context",
		src:    `{{len .}}`,
		subStr: "l",
		contains: []string{
			"len",
			"eq",
			"ne",
			"and",
			"or",
			"not",
			"print",
			"printf",
			"println",
			"index",
		},
	},
	{
		name:        "builtins absent when sChar is dot",
		src:         `{{.}}`,
		subStr:      ".",
		notContains: []string{"len"},
	},
	{
		name:        "builtins absent when sChar is dollar",
		src:         `{{$x := .}}{{$}}`,
		subStr:      "$",
		occurrence:  1,
		notContains: []string{"len", "range"},
	},
	// pipe filtered suggestions
	{
		name:        "after len pipe - int-accepting builtins only",
		src:         `{{. | len | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"eq", "lt", "print"},
		notContains: []string{"index", "js"},
	},
	{
		name:        "after not pipe - bool-accepting builtins only",
		src:         `{{not . | and . .}}`,
		subStr:      "a",
		withType:    true,
		contains:    []string{"and", "or", "not"},
		notContains: []string{"len", "html"},
	},
	{
		name:        "after html pipe - string-accepting builtins only",
		src:         `{{html . | len .}}`,
		subStr:      "l",
		occurrence:  1,
		withType:    true,
		contains:    []string{"len", "index"},
		notContains: []string{"and", "not"},
	},
	{
		name:     "no preceding pipe - full builtin list",
		src:      `{{len .}}`,
		subStr:   "l",
		contains: []string{"len", "html", "and"},
	},
	// command node position
	{
		name:     "first arg of command - builtins returned",
		src:      `{{len .}}`,
		subStr:   "l",
		contains: []string{"len", "eq"},
	},
}

type nodeFindTestCase struct {
	name     string
	src      string
	pos      int
	isDot    bool
	isIdent  bool
	ident    string
	isVar    bool
	varIdent string
}

var nodeFindTestCases = []nodeFindTestCase{
	{
		name:  "finds dot node at its position",
		src:   `{{.}}`,
		pos:   2,
		isDot: true,
	},
	{
		name:    "finds identifier node",
		src:     `{{len .}}`,
		pos:     2,
		isIdent: true,
		ident:   "len",
	},
	{
		name:     "finds variable node",
		src:      `{{$x := .}}{{$x}}`,
		pos:      20,
		isVar:    true,
		varIdent: "$x",
	},
}

// buildPath scope test cases

type buildPathScopeTestCase struct {
	name        string
	src         string
	dotOccur    int
	varName     string
	wantPresent bool
}

var buildPathScopeTestCases = []buildPathScopeTestCase{
	{
		name:        "vars reset after if branch not taken",
		src:         `{{if .}}{{$inner := .}}{{end}}{{.}}`,
		dotOccur:    2,
		varName:     "$inner",
		wantPresent: false,
	},
	{
		name:        "outer var always in scope",
		src:         `{{$outer := .}}{{if .}}{{end}}{{.}}`,
		dotOccur:    2,
		varName:     "$outer",
		wantPresent: true,
	},
	{
		name:        "vars reset after block body",
		src:         `{{block "csv" .}}{{$inner := .}}{{end}}{{.}}`,
		dotOccur:    2,
		varName:     "$inner",
		wantPresent: false,
	},
	{
		name:        "outer var visible inside block",
		src:         `{{$outer := .}}{{block "csv" .}}{{end}}{{.}}`,
		dotOccur:    2,
		varName:     "$outer",
		wantPresent: true,
	},
}

// completionAst handler test cases

type completionAstTestCase struct {
	name           string
	content        string
	uri            string
	serverDisabled bool
	skipStore      bool
	line           uint32
	character      uint32
	wantNil        bool
	wantLabels     []string
}

var completionAstTestCases = []completionAstTestCase{
	{
		name:           "returns nil when server disabled",
		content:        "{{.}}",
		uri:            "file:///disabled.tmpl",
		serverDisabled: true,
		character:      2,
		wantNil:        true,
	},
	{
		name:      "returns nil when document not in store",
		uri:       "file:///missing.tmpl",
		skipStore: true,
		character: 2,
		wantNil:   true,
	},
	{
		name:      "returns result when tree has broken template",
		content:   "{{invalid template {{{{",
		uri:       "file:///notree.tmpl",
		character: 2,
	},
	{
		name:      "returns nil when cursor outside template block",
		content:   "{{.}}\nplain text",
		uri:       "file:///outside.tmpl",
		line:      1,
		character: 2,
		wantNil:   true,
	},
	{
		name:       "returns CompletionList for valid position",
		content:    "{{.}}",
		uri:        "file:///valid.tmpl",
		character:  2,
		wantLabels: []string{"."},
	},
}

// completionWithFallback handler test cases

type completionFallbackTestCase struct {
	name      string
	content   string
	uri       string
	line      uint32
	character uint32
	wantList  bool
}

var completionFallbackTestCases = []completionFallbackTestCase{
	{
		name:      "returns ast result when ast succeeds",
		content:   "{{.}}",
		uri:       "file:///fallback-ok.tmpl",
		character: 2,
		wantList:  true,
	},
	{
		name:      "falls back to regex when ast returns nil",
		content:   "{{$x := .}}\nplain",
		uri:       "file:///fallback-nil.tmpl",
		line:      1,
		character: 2,
	},
}

// varsItemsT unit test cases

type varsItemsTTestCase struct {
	name        string
	varNames    []string // each entry is Ident[0], e.g. "$x"; nil means pass nil vars
	delSign     bool
	wantNil     bool
	wantLen     int // if > 0, assert exact length
	wantLabels  []string
	notContains []string
	wantFilter  string // if non-empty, assert FilterText of first item equals this
}

var varsItemsTTestCases = []varsItemsTTestCase{
	{
		name:    "nil vars returns nil",
		wantNil: true,
	},
	{
		name:     "empty vars returns nil",
		varNames: []string{},
		wantNil:  true,
	},
	{
		name:       "delSign false keeps full dollar-prefixed label",
		varNames:   []string{"$top"},
		delSign:    false,
		wantLabels: []string{"$top"},
	},
	{
		name:       "delSign true strips sigil from label",
		varNames:   []string{"$top"},
		delSign:    true,
		wantLabels: []string{"top"},
	},
	{
		name:       "delSign true skips bare dollar variable",
		varNames:   []string{"$", "$x"},
		delSign:    true,
		wantLen:    1,
		wantLabels: []string{"x"},
	},
	{
		name:     "deduplicates repeated variable names",
		varNames: []string{"$x", "$x"},
		wantLen:  1,
	},
	{
		name:       "filter text retains sigil when delSign true",
		varNames:   []string{"$val"},
		delSign:    true,
		wantFilter: "$val",
	},
}

// fieldChainItemsT unit test cases

type fieldChainItemsTTestCase struct {
	name        string
	useBasic    bool // pass types.Typ[types.String] as the type
	useOrder    bool // pass the loaded Order *types.Named
	contains    []string
	notContains []string
	wantEmpty   bool
}

var fieldChainItemsTTestCases = []fieldChainItemsTTestCase{
	{
		name:      "nil type returns empty slice",
		wantEmpty: true,
	},
	{
		name:      "basic string type returns empty slice",
		useBasic:  true,
		wantEmpty: true,
	},
	{
		name:        "named struct returns fields without dot prefix",
		useOrder:    true,
		contains:    []string{"ID", "CustomerName", "Address"},
		notContains: []string{".ID"},
	},
	{
		name:        "named struct returns methods without dot prefix",
		useOrder:    true,
		contains:    []string{"DisplayName", "ItemCount"},
		notContains: []string{".DisplayName"},
	},
}

// dotItemsT unit test cases

type dotItemsTTestCase struct {
	name          string
	src           string
	subStr        string
	delSign       bool
	inputIsString bool // set inputType = types.Typ[types.String]
	pipeKind      outputKind
	withType      bool
	contains      []string
	notContains   []string
	wantEmpty     bool
}

var dotItemsTTestCases = []dotItemsTTestCase{
	{
		name:      "non-outputAny pipe kind blocks all items",
		src:       `{{.}}`,
		subStr:    ".",
		pipeKind:  outputBool,
		wantEmpty: true,
	},
	{
		name:          "non-nil inputType blocks all items",
		src:           `{{.}}`,
		subStr:        ".",
		inputIsString: true,
		wantEmpty:     true,
	},
	{
		name:     "delSign false includes dot item",
		src:      `{{.}}`,
		subStr:   ".",
		delSign:  false,
		contains: []string{"."},
	},
	{
		name:        "delSign true omits dot item",
		src:         `{{.}}`,
		subStr:      ".",
		delSign:     true,
		notContains: []string{"."},
	},
	{
		name:        "with loaded type delSign false returns dot-prefixed fields and methods",
		src:         `{{.}}`,
		subStr:      ".",
		withType:    true,
		delSign:     false,
		contains:    []string{".ID", ".CustomerName", ".DisplayName"},
		notContains: []string{"ID"},
	},
	{
		name:        "with loaded type delSign true returns unprefixed fields and methods",
		src:         `{{.}}`,
		subStr:      ".",
		withType:    true,
		delSign:     true,
		contains:    []string{"ID", "DisplayName"},
		notContains: []string{".ID", "."},
	},
}

// pipeFilteredItemsT unit test cases

type pipeFilteredItemsTTestCase struct {
	name          string
	src           string
	subStr        string
	kind          outputKind
	inputIsString bool // set inputType = types.Typ[types.String]
	contains      []string
	notContains   []string
}

var pipeFilteredItemsTTestCases = []pipeFilteredItemsTTestCase{
	{
		name:     "outputAny includes all builtins",
		src:      `{{.}}`,
		subStr:   ".",
		kind:     outputAny,
		contains: []string{"len", "html", "not", "eq"},
	},
	{
		name:        "outputString includes only string-accepting builtins",
		src:         `{{.}}`,
		subStr:      ".",
		kind:        outputString,
		contains:    []string{"html", "len"},
		notContains: []string{"not", "and"},
	},
	{
		name:        "outputBool includes only bool-accepting builtins",
		src:         `{{.}}`,
		subStr:      ".",
		kind:        outputBool,
		contains:    []string{"not", "and"},
		notContains: []string{"html", "len"},
	},
	{
		name:        "outputInt includes only int-accepting builtins",
		src:         `{{.}}`,
		subStr:      ".",
		kind:        outputInt,
		contains:    []string{"eq", "lt"},
		notContains: []string{"html", "not"},
	},
	{
		name:     "outputUntyped includes all builtins",
		src:      `{{.}}`,
		subStr:   ".",
		kind:     outputUntyped,
		contains: []string{"len", "not", "html"},
	},
	{
		name:          "inputType string with outputAny infers string-accepting builtins",
		src:           `{{.}}`,
		subStr:        ".",
		kind:          outputAny,
		inputIsString: true,
		contains:      []string{"html", "len"},
		notContains:   []string{"not"},
	},
}

// pipeOutputKind

type pipeOutputKindTestCase struct {
	name      string
	src       string // template; the test extracts the pipe from the first ActionNode
	nilPipe   bool   // pass ctx with Pipe=nil
	isInvoked bool
	want      outputKind
}

var pipeOutputKindTestCases = []pipeOutputKindTestCase{
	{
		name:    "nil pipe returns outputAny",
		nilPipe: true,
		want:    outputAny,
	},
	{
		name: "single command pipe, non-invoked returns outputAny",
		src:  `{{html .}}`,
		want: outputAny,
	},
	{
		name:      "html preceding when invoked returns outputString",
		src:       `{{html .}}`,
		isInvoked: true,
		want:      outputString,
	},
	{
		name:      "len preceding when invoked returns outputInt",
		src:       `{{len .}}`,
		isInvoked: true,
		want:      outputInt,
	},
	{
		name:      "not preceding when invoked returns outputBool",
		src:       `{{not .}}`,
		isInvoked: true,
		want:      outputBool,
	},
	{
		name: "html piped to x non-invoked returns outputString",
		src:  `{{html . | x}}`,
		want: outputString,
	},
	{
		name:      "non-builtin identifier preceding returns outputAny",
		src:       `{{foo .}}`,
		isInvoked: true,
		want:      outputAny,
	},
	{
		name:      "dot first arg preceding returns outputAny",
		src:       `{{. | html}}`,
		isInvoked: false,
		want:      outputAny,
	},
}

// basicTypeMatchesKind unit test cases

type basicTypeMatchesKindTestCase struct {
	name  string
	basic string
	kind  outputKind
	want  bool
}

var basicTypeMatchesKindTestCases = []basicTypeMatchesKindTestCase{
	{name: "non-basic type returns false", basic: "none", kind: outputString, want: false},
	{name: "string matches outputString", basic: "string", kind: outputString, want: true},
	{name: "string does not match outputInt", basic: "string", kind: outputInt, want: false},
	{name: "int matches outputInt", basic: "int", kind: outputInt, want: true},
	{name: "int does not match outputBool", basic: "int", kind: outputBool, want: false},
	{name: "bool matches outputBool", basic: "bool", kind: outputBool, want: true},
	{name: "bool does not match outputString", basic: "bool", kind: outputString, want: false},
	{name: "string with outputAny returns false", basic: "string", kind: outputAny, want: false},
	{name: "float with outputInt returns false", basic: "float64", kind: outputInt, want: false},
}

// methodAcceptsInput unit test cases

type methodAcceptsInputTestCase struct {
	name          string
	methodName    string // Order method to use
	inputIsString bool
	inputIsInt    bool
	pipeKind      outputKind
	want          bool
}

var methodAcceptsInputTestCases = []methodAcceptsInputTestCase{
	{
		name:       "no input and outputAny returns true",
		methodName: "Format",
		pipeKind:   outputAny,
		want:       true,
	},
	{
		name:       "no input and outputUntyped returns true",
		methodName: "Format",
		pipeKind:   outputUntyped,
		want:       true,
	},
	{
		name:          "string input matches Format(string)",
		methodName:    "Format",
		inputIsString: true,
		want:          true,
	},
	{
		name:       "int input does not match Format(string)",
		methodName: "Format",
		inputIsInt: true,
		want:       false,
	},
	{
		name:       "outputString matches last param of Format(string)",
		methodName: "Format",
		pipeKind:   outputString,
		want:       true,
	},
	{
		name:       "outputInt does not match last param of Format(string)",
		methodName: "Format",
		pipeKind:   outputInt,
		want:       false,
	},
	{
		name:       "outputInt matches last param of Oper(int)",
		methodName: "Oper",
		pipeKind:   outputInt,
		want:       true,
	},
	{
		name:       "outputString does not match last param of Oper(int)",
		methodName: "Oper",
		pipeKind:   outputString,
		want:       false,
	},
}

// methodIsUsable unit test cases

type methodIsUsableTestCase struct {
	name       string
	nilFunc    bool   // construct MethodType with Func=nil
	methodName string // otherwise look up an Order method
	want       bool
}

var methodIsUsableTestCases = []methodIsUsableTestCase{
	{name: "nil func returns false", nilFunc: true, want: false},
	{name: "single return value is usable", methodName: "DisplayName", want: true},
	{name: "second return error is usable", methodName: "Summary", want: true},
}

// toNamed unit test cases

type toNamedTestCase struct {
	name    string
	input   string // "nil"|"named"|"pointer"|"basic"
	wantNil bool
}

var toNamedTestCases = []toNamedTestCase{
	{name: "nil input returns nil", input: "nil", wantNil: true},
	{name: "named type returns the named", input: "named", wantNil: false},
	{name: "pointer to named returns underlying named", input: "pointer", wantNil: false},
	{name: "basic type returns nil", input: "basic", wantNil: true},
}

// resolvePipeDotType unit test cases

type resolvePipeDotTypeTestCase struct {
	name        string
	src         string
	unwrapSlice bool
	nilCtxDot   bool
	wantNilDot  bool
	wantName    string
	wantOrder   bool
}

var resolvePipeDotTypeTestCases = []resolvePipeDotTypeTestCase{
	{
		name:        "range over slice field resolves to element type",
		src:         `{{range .Items}}{{end}}`,
		unwrapSlice: true,
		wantName:    "Item",
	},
	{
		name:     "with over named struct field resolves to that type",
		src:      `{{with .Address}}{{end}}`,
		wantName: "Address",
	},
	{
		name:        "range over non-slice field returns nil DotType",
		src:         `{{range .Address}}{{end}}`,
		unwrapSlice: true,
		wantNilDot:  true,
	},
	{
		name:       "with over slice field returns nil DotType",
		src:        `{{with .Items}}{{end}}`,
		wantNilDot: true,
	},
	{
		name:        "range over unknown field returns original DotType",
		src:         `{{range .Unknown}}{{end}}`,
		unwrapSlice: true,
		wantOrder:   true,
	},
	{
		name:      "with over unknown field returns original DotType",
		src:       `{{with .Unknown}}{{end}}`,
		wantOrder: true,
	},
	{
		name:      "with over non-field arg (dot) returns original DotType",
		src:       `{{with .}}{{end}}`,
		wantOrder: true,
	},
	{
		name:      "with over basic-typed field returns original DotType",
		src:       `{{with .CustomerName}}{{end}}`,
		wantOrder: true,
	},
	{
		name:       "nil ctx.DotType returns nil",
		src:        `{{with .Address}}{{end}}`,
		nilCtxDot:  true,
		wantNilDot: true,
	},
}

// additional buildPathScope cases registered via init to cover range/with/template/chain traversal

func init() {
	buildPathScopeTestCases = append(buildPathScopeTestCases,
		buildPathScopeTestCase{
			name:        "range body variable not visible after end",
			src:         `{{range $r := .}}{{end}}{{.}}`,
			dotOccur:    1,
			varName:     "$r",
			wantPresent: false,
		},
		buildPathScopeTestCase{
			name:        "with body variable not visible after end",
			src:         `{{with $w := .}}{{end}}{{.}}`,
			dotOccur:    1,
			varName:     "$w",
			wantPresent: false,
		},
		buildPathScopeTestCase{
			name:        "template node traversed without panic",
			src:         `{{template "t" .}}{{.}}`,
			dotOccur:    1,
			varName:     "$x",
			wantPresent: false,
		},
		buildPathScopeTestCase{
			name:        "chain node traversed when target is inside",
			src:         `{{(.Address).Street}}{{.}}`,
			dotOccur:    1,
			varName:     "$x",
			wantPresent: false,
		},
		buildPathScopeTestCase{
			name:        "range variable visible inside body",
			src:         `{{range $r := .}}{{.}}{{end}}`,
			dotOccur:    1,
			varName:     "$r",
			wantPresent: true,
		},
		buildPathScopeTestCase{
			name:        "with variable visible inside body",
			src:         `{{with $w := .}}{{.}}{{end}}`,
			dotOccur:    1,
			varName:     "$w",
			wantPresent: true,
		},
	)
}

// completionAstMultiDefineCase covers AST-based completion behaviour when a
// single document contains multiple {{define}} blocks.
type completionAstMultiDefineCase struct {
	name           string
	posSubStr      string // substring whose first byte locates the cursor
	posOccurrence  int
	posCharOffset  int // bytes added to the substring's first byte
	wantContains   []string
	wantNotContain []string
}

var completionAstMultiDefineCases = []completionAstMultiDefineCase{
	{
		name:           "dot completion inside Order define proposes Order fields",
		posSubStr:      "{{ .CustomerName",
		posOccurrence:  0,
		posCharOffset:  4, // sits on the dot before CustomerName
		wantContains:   []string{"CustomerName", "ID", "Address", "Items"},
		wantNotContain: []string{"Street", "City"},
	},
	{
		name:           "dot completion inside Address define proposes Address fields",
		posSubStr:      "{{ .Street",
		posOccurrence:  0,
		posCharOffset:  4, // sits on the dot before Street
		wantContains:   []string{"Street", "City", "Country", "Zip"},
		wantNotContain: []string{"CustomerName", "Items"},
	},
	{
		name:           "dot completion in root template proposes root Address fields",
		posSubStr:      "{{ .Country",
		posOccurrence:  0,
		posCharOffset:  4, // sits on the dot before Country
		wantContains:   []string{"Street", "City", "Country", "Zip"},
		wantNotContain: []string{"CustomerName", "Items"},
	},
}
