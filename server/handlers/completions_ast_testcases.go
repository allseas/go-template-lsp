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

var completionTestCases = []completionTestCase{
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
	// pipe filtering — model fields
	{
		name:        "string field piped — string-accepting builtins suggested",
		src:         `{{.CustomerName | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "js", "urlquery", "len"},
		notContains: []string{"not", "and"},
	},
	{
		name:        "bool field piped — bool-accepting builtins suggested",
		src:         `{{.Paid | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"not", "and", "or"},
		notContains: []string{"html", "len"},
	},
	{
		name:      "float field piped — outputUntyped, all builtins shown",
		src:       `{{.TotalAmount | }}`,
		subStr:    "}}",
		offsetAdj: -1,
		isInvoked: true,
		withType:  true,
		contains:  []string{"len", "html", "eq"},
	},
	{
		name:      "struct field piped — outputUntyped, all builtins shown",
		src:       `{{.Address | }}`,
		subStr:    "}}",
		offsetAdj: -1,
		isInvoked: true,
		withType:  true,
		contains:  []string{"len", "html", "and"},
	},
	// pipe filtering — model methods
	{
		name:        "string-returning method piped — string-accepting builtins",
		src:         `{{.DisplayName | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "js", "len"},
		notContains: []string{".Format", "not"},
	},
	{
		name:        "int-returning method piped — int-accepting builtins",
		src:         `{{.ItemCount | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"eq", "lt", "gt"},
		notContains: []string{"html", "not"},
	},
	{
		name:        "bool-returning method piped — bool-accepting builtins",
		src:         `{{.IsLargeOrder | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"not", "and"},
		notContains: []string{"len", "html"},
	},
	{
		name:        "string-returning method with arg piped — string builtins",
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
		name:      "string field piped — no dot-prefixed methods",
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
		name:        "bool field piped — no dot-prefixed methods",
		src:         `{{.Paid | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{".Format", ".DisplayName"},
	},
	{
		name:        "identifier in pipe — no dot-prefixed methods",
		src:         `{{.ItemCount | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{".Format", ".DisplayName"},
	},
	{
		name:        "CustomerName pipe — bare method names absent",
		src:         `{{.CustomerName | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{"DisplayName", "Format", "wrongSecond"},
	},
	{
		name:        "Paid pipe — bare method names absent",
		src:         `{{.Paid | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{"DisplayName", "Format", "wrongSecond"},
	},
	{
		name:        "identifier pipe — bare method names absent",
		src:         `{{.ItemCount | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{"DisplayName", "Format", "wrongSecond"},
	},
	// slice field pipe
	{
		name:        "slice field piped — len/index/slice shown, no methods",
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
		name:        "html after string field — string builtins only",
		src:         `{{.CustomerName | html | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"len", "js"},
		notContains: []string{"not", "eq"},
	},
	{
		name:        "not after bool field — bool builtins only",
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
		name:        "dot piped — all builtins shown, no dot-prefixed fields",
		src:         `{{. | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"len", "html", "and"},
		notContains: []string{".", ".Address", ".Items", ".ID"},
	},
	{
		name:        "struct field piped — dot-prefixed fields excluded",
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
		name:      "len of items piped — int-accepting builtins",
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
		name:        "html of string field piped — string builtins",
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
		name:        "invoked after string pipe — string builtins, not bool",
		src:         `{{.CustomerName | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"html", "len"},
		notContains: []string{"not"},
	},
	// scope switch — range
	{
		name:        "inside range — dot-prefixed Item methods, Order methods absent",
		src:         `{{range .Items}}{{len .SKU}}{{end}}`,
		subStr:      "l",
		withType:    true,
		contains:    []string{".IsExpensive", ".Describe", ".Label"},
		notContains: []string{".DisplayName", ".IsLargeOrder", ".wrongSecond"},
	},
	{
		name:       "inside range pipe — Item methods excluded, none accept string",
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
		name:        "inside range — string Item method piped, string builtins",
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
		name:        "inside range — bool Item method piped, bool builtins",
		src:         `{{range .Items}}{{IsExpensive | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"not", "and"},
		notContains: []string{"html", "len"},
	},
	// scope switch — with
	{
		name:        "inside with — dot-prefixed Address methods, Order methods absent",
		src:         `{{with .Address}}{{len .Street}}{{end}}`,
		subStr:      "l",
		withType:    true,
		contains:    []string{".Line", ".IsLocal", ".ZipCode"},
		notContains: []string{".DisplayName", ".IsLargeOrder", ".wrongSecond"},
	},
	{
		name:        "inside with pipe — Address methods excluded, none accept string",
		src:         `{{with .Address}}{{Line | }}{{end}}`,
		subStr:      "}}",
		occurrence:  1,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		notContains: []string{".Line", ".IsLocal", ".ZipCode", ".DisplayName", ".IsLargeOrder"},
	},
	{
		name:        "inside with — string Address method piped, string builtins",
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
		name:        "inside with — bool Address method piped, bool builtins",
		src:         `{{with .Address}}{{IsLocal | }}{{end}}`,
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
		name:        "dot in if condition — no builtins",
		src:         `{{if .}}{{end}}`,
		subStr:      ".",
		notContains: []string{"eq", "len"},
	},
	{
		name:        "dot in range pipeline — no builtins",
		src:         `{{range .}}{{end}}`,
		subStr:      ".",
		notContains: []string{"eq", "len"},
	},
	{
		name:        "dot in with pipeline — no builtins",
		src:         `{{with .}}{{end}}`,
		subStr:      ".",
		notContains: []string{"eq", "len"},
	},
	{
		name:        "sChar dot — dot item returned, not builtins",
		src:         `{{.}}`,
		subStr:      ".",
		notContains: []string{"eq", "len"},
	},
	// variable suggestions
	{
		name:        "dollar sChar — vars returned without sigil",
		src:         `{{$top := .}}{{$}}`,
		subStr:      "$",
		occurrence:  1,
		contains:    []string{"top"},
		notContains: []string{"$top"},
	},
	{
		name:     "non-dollar sChar — full $var label included",
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
		name:        "after len pipe — int-accepting builtins only",
		src:         `{{. | len | }}`,
		subStr:      "}}",
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"eq", "lt", "print"},
		notContains: []string{"index", "js"},
	},
	{
		name:        "after not pipe — bool-accepting builtins only",
		src:         `{{not . | and . .}}`,
		subStr:      "a",
		withType:    true,
		contains:    []string{"and", "or", "not"},
		notContains: []string{"len", "html"},
	},
	{
		name:        "after html pipe — string-accepting builtins only",
		src:         `{{html . | len .}}`,
		subStr:      "l",
		withType:    true,
		contains:    []string{"len", "index"},
		notContains: []string{"and", "not"},
	},
	{
		name:     "no preceding pipe — full builtin list",
		src:      `{{len .}}`,
		subStr:   "l",
		contains: []string{"len", "html", "and"},
	},
	// command node position
	{
		name:     "first arg of command — builtins returned",
		src:      `{{len .}}`,
		subStr:   "l",
		contains: []string{"len", "eq"},
	},
}

// nodeFind test cases

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
