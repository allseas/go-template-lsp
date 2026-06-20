package handlers

import (
	"testing"

	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withFuncmapGlobals seeds the global function cache with the language builtins
// merged with the sample workspace globals in test/resources/funcmap-tests
func withFuncmapGlobals(t *testing.T) {
	t.Helper()
	funcs := serverTypes.BuiltinFuncs()
	loaded, err := serverTypes.LoadGlobalFuncs("../../test/resources/funcmap-tests")
	require.NoError(t, err)
	for k, v := range loaded {
		if _, isBuiltin := funcs[k]; !isBuiltin {
			funcs[k] = v
		}
	}
	serverTypes.SetGlobalFuncs(funcs)
	t.Cleanup(func() { serverTypes.SetGlobalFuncs(serverTypes.BuiltinFuncs()) })
}

func TestArgKindCompletions(t *testing.T) {
	withFuncmapGlobals(t)
	original := GetConfig()
	cfg := original
	cfg.PipeChainCompletion = "full"
	setConfig(cfg)
	t.Cleanup(func() { setConfig(original) })
	lt := orderLoadedType(t)
	for _, tc := range argKindCompletionTestCases {
		t.Run(tc.name, func(t *testing.T) {
			offset := offsetOf(t, tc.src, tc.subStr, tc.occurrence) + tc.offsetAdj
			labels := suggestAtWithType(t, tc.src, offset, tc.isInvoked, lt)
			for _, want := range tc.contains {
				assert.Contains(t, labels, want)
			}
			for _, notWant := range tc.notContains {
				assert.NotContains(t, labels, notWant)
			}
		})
	}
}

var argKindCompletionTestCases = []completionTestCase{
	{
		name:        "cyclic test",
		src:         `{{ repeat x }}`,
		subStr:      "x",
		occurrence:  0,
		offsetAdj:   1,
		withType:    true,
		contains:    []string{".Tree.Trr", ".Tree.Left.Trr", ".Tree.Right.Trr"},
		notContains: []string{".Tree.Left.Left.Trr"},
	},
	{
		name:        "sort func test",
		src:         `{{ .CustomerName |  }}`,
		subStr:      "}}",
		occurrence:  0,
		offsetAdj:   -1,
		isInvoked:   true,
		withType:    true,
		contains:    []string{"len", "shout"},
		notContains: []string{".Amt"},
	},
	{
		name:        "int func arg - int methods suggested, string excluded",
		src:         `{{ repeat .Address. }}`,
		subStr:      ".",
		occurrence:  1,
		offsetAdj:   1,
		withType:    true,
		contains:    []string{".CustomerName", ".DisplayName"},
		notContains: []string{".Amt"},
	},
	{
		name:       "pipe into string func - string fields/methods suggested",
		src:        `{{ x | upper }}`,
		subStr:     "x",
		occurrence: 0,
		withType:   true,
		contains:   []string{".CustomerName", ".Email", ".ID", ".DisplayName"},
		notContains: []string{
			".Paid", ".Address", ".Items", ".ItemCount", ".IsLargeOrder",
		},
	},
	{
		name:       "pipe into string func - string user funcs kept, int user func dropped",
		src:        `{{ x | upper }}`,
		subStr:     "x",
		occurrence: 0,
		withType:   true,
		// builtins always offered; upper/lower/repeat/shout return string.
		contains:    []string{"upper", "lower", "repeat", "html", "len", "not", "and"},
		notContains: []string{"wc"},
	},
	{
		name:        "dot piped into string func - string fields/methods, dot consumed",
		src:         `{{ . | upper }}`,
		subStr:      ".",
		occurrence:  0,
		withType:    true,
		contains:    []string{"CustomerName", "Email", "ID", "DisplayName"},
		notContains: []string{"Paid", "Address", "Items", "ItemCount", "IsLargeOrder"},
	},
	{
		name:       "pipe into int func - int methods kept, string excluded",
		src:        `{{ x | wc }}`,
		subStr:     "x",
		occurrence: 0,
		withType:   true,
		// wc accepts a string, so the value flowing in must be a string.
		contains:    []string{".CustomerName", ".DisplayName"},
		notContains: []string{".Paid", ".ItemCount"},
	},
	{
		name:        "string func arg - string fields/methods suggested",
		src:         `{{ upper x }}`,
		subStr:      " x",
		occurrence:  0,
		offsetAdj:   1,
		withType:    true,
		contains:    []string{".CustomerName", ".DisplayName"},
		notContains: []string{".Paid", ".Address", ".ItemCount", ".IsLargeOrder"},
	},
	{
		name:        "string func arg - int user func dropped, builtins kept",
		src:         `{{ upper x }}`,
		subStr:      " x",
		occurrence:  0,
		offsetAdj:   1,
		withType:    true,
		contains:    []string{"upper", "lower", "html", "len", "and"},
		notContains: []string{"wc"},
	},
	{
		name:        "int func arg - int methods suggested, string excluded",
		src:         `{{ repeat "a" x }}`,
		subStr:      " x",
		occurrence:  0,
		offsetAdj:   1,
		withType:    true,
		contains:    []string{".ItemCount", ".Oper"},
		notContains: []string{".CustomerName", ".Paid", ".Address"},
	},
	{
		name:        "int func arg - int user func kept, string user func dropped",
		src:         `{{ repeat "a" x }}`,
		subStr:      " x",
		occurrence:  0,
		offsetAdj:   1,
		withType:    true,
		contains:    []string{"wc", "len", "and"},
		notContains: []string{"upper", "lower"},
	},
	{
		name:       "piped bool into and, explicit arg - dot items visible",
		src:        `{{ .Paid | and x }}`,
		subStr:     " x",
		occurrence: 0,
		offsetAdj:  1,
		withType:   true,
		// anyT params → all fields/funcs offered (pipe value must not hide them)
		contains: []string{".Paid", ".CustomerName", ".ItemCount", "html", "not", "and"},
	},
	{
		name:       "direct and call, second arg - dot items visible",
		src:        `{{ and .Paid x }}`,
		subStr:     " x",
		occurrence: 0,
		offsetAdj:  1,
		withType:   true,
		contains:   []string{".Paid", ".CustomerName", ".ItemCount", "html", "not", "and"},
	},

	// User-defined `repeat(s string, n int) string` — piped value fills n (int).
	// The cursor is the first explicit arg → fills s (string). Suggestions should
	// be string-compatible, not int-compatible and not constrained by the piped
	// string type being an "accepted input" for other functions.
	{
		name:        "piped string into repeat, cursor fills string param - string items",
		src:         `{{ .CustomerName | repeat x }}`,
		subStr:      " x",
		occurrence:  0,
		offsetAdj:   1,
		withType:    true,
		contains:    []string{".CustomerName", ".ID", ".DisplayName", "upper", "lower"},
		notContains: []string{".ItemCount", ".Paid", "wc"},
	},
	{
		name:       "piped string into repeat, cursor fills string param - not int fields",
		src:        `{{ .CustomerName | repeat x }}`,
		subStr:     " x",
		occurrence: 0,
		offsetAdj:  1,
		withType:   true,
		// pipe value (.CustomerName) must not hide dot/field items
		contains:    []string{".CustomerName"},
		notContains: []string{".ItemCount"},
	},

	// User-defined `wc(s string) int` — piped value fills the only param (s).
	// A second explicit arg has no valid param slot; we still give string-flavoured
	// suggestions (clamp to last param) without crashing.
	{
		name:       "piped value into wc, over-specified cursor - graceful string suggestions",
		src:        `{{ .CustomerName | wc x }}`,
		subStr:     " x",
		occurrence: 0,
		offsetAdj:  1,
		withType:   true,
		// param 0 (s string) is the clamped result — string suggestions expected
		contains:    []string{".CustomerName", "upper"},
		notContains: []string{".ItemCount", "wc"},
	},

	// ---- whole-signature per-argument suggestions: repeat(string, int) --------
	// Each explicit argument slot is constrained to its own parameter type, not
	// to the first parameter of the signature.
	{
		name:        "repeat first arg - constrained to string param",
		src:         `{{ repeat x1 x2 }}`,
		subStr:      "x1",
		occurrence:  0,
		withType:    true,
		contains:    []string{".CustomerName", ".DisplayName"},
		notContains: []string{".ItemCount", ".Paid"},
	},
	{
		name:        "repeat second arg - constrained to int param",
		src:         `{{ repeat x1 x2 }}`,
		subStr:      "x2",
		occurrence:  0,
		withType:    true,
		contains:    []string{".ItemCount", ".Oper"},
		notContains: []string{".CustomerName", ".DisplayName"},
	},
	{
		name:        "repeat empty first slot - string suggestions",
		src:         `{{ repeat  }}`,
		subStr:      "}}",
		occurrence:  0,
		offsetAdj:   -1, // land in the empty first argument slot
		isInvoked:   true,
		withType:    true,
		contains:    []string{".CustomerName", ".DisplayName"},
		notContains: []string{".ItemCount", ".Paid"},
	},
	{
		name:        "repeat empty second slot - int suggestions",
		src:         `{{ repeat "a"  }}`,
		subStr:      "}}",
		occurrence:  0,
		offsetAdj:   -1, // land in the empty second argument slot
		isInvoked:   true,
		withType:    true,
		contains:    []string{".ItemCount", ".Oper"},
		notContains: []string{".CustomerName", ".DisplayName"},
	},
	{
		name:        "empty stage after repeat pipe - not constrained to repeat's int param",
		src:         `{{ repeat .Address.City .ItemCount | }}`,
		subStr:      "}}",
		occurrence:  0,
		offsetAdj:   -1, // land in the empty stage after '|'
		isInvoked:   true,
		withType:    true,
		contains:    []string{"upper", "lower"},
		notContains: []string{".ItemCount", ".Oper", ".CustomerName"},
	},
}

// TestChainExpansionEmptyArgSlot verifies that the nested pipe-chain expansion
func TestChainExpansionEmptyArgSlot(t *testing.T) {
	withFuncmapGlobals(t)
	original := GetConfig()
	cfg := original
	cfg.PipeChainCompletion = "full"
	setConfig(cfg)
	t.Cleanup(func() { setConfig(original) })
	lt := orderLoadedType(t)

	cases := []completionTestCase{
		{
			name:        "bare dot in repeat string slot - nested string path (dot consumed)",
			src:         `{{ repeat . }}`,
			subStr:      ".",
			occurrence:  0,
			withType:    true,
			contains:    []string{"Address.City", "Address.Street"},
			notContains: []string{"Address.ZipCode"},
		},
		{
			name:        "empty repeat string slot - nested string path (dot prefixed)",
			src:         `{{ repeat  }}`,
			subStr:      "}}",
			occurrence:  0,
			offsetAdj:   -1,
			isInvoked:   true,
			withType:    true,
			contains:    []string{".Address.City", ".Address.Street"},
			notContains: []string{".Address.ZipCode"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			offset := offsetOf(t, tc.src, tc.subStr, tc.occurrence) + tc.offsetAdj
			labels := suggestAtWithType(t, tc.src, offset, tc.isInvoked, lt)
			for _, want := range tc.contains {
				assert.Contains(t, labels, want)
			}
			for _, notWant := range tc.notContains {
				assert.NotContains(t, labels, notWant)
			}
		})
	}
}
