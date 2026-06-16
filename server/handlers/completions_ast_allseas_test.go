//go:build allseas

package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var completions_ast_tests_allseas = []completionTestCase{
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

func TestCompletionAstAllseas(t *testing.T) {
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
