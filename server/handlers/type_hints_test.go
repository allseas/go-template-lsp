package handlers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTypeHints(t *testing.T) {
	for _, tc := range parseTypeHintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			hints := ParseTypeHints(strings.NewReader(tc.input))
			assert.Equal(t, tc.wantHints, hints)
		})
	}
}

func TestSplitTypeHint(t *testing.T) {
	for _, tc := range splitTypeHintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			importPath, typeName := splitTypeHint(tc.hint)
			assert.Equal(t, tc.wantImport, importPath)
			assert.Equal(t, tc.wantType, typeName)
		})
	}
}

func TestLoadTypeFromHint(t *testing.T) {
	for _, tc := range loadTypeHintTestCases {
		t.Run(tc.name, func(t *testing.T) {
			lt, err := LoadTypeFromHint(tc.hint, tc.root)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, lt)

			if tc.wantTypeName != "" {
				assert.Equal(t, tc.wantTypeName, lt.Named.Obj().Name())
			}

			fieldNames := make([]string, len(lt.Fields))
			for i, f := range lt.Fields {
				fieldNames[i] = f.Name
			}
			for _, want := range tc.wantFields {
				assert.Contains(t, fieldNames, want)
			}

			methodNames := make([]string, len(lt.Methods))
			for i, m := range lt.Methods {
				methodNames[i] = m.Name
			}
			for _, want := range tc.wantMethods {
				assert.Contains(t, methodNames, want)
			}
		})
	}
}
