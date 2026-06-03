package types

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
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

func TestNamedMethods(t *testing.T) {
	t.Run("extracts Order methods and params", func(t *testing.T) {
		cfg := &packages.Config{Mode: packages.NeedTypes, Dir: "testdata"}
		pkgs, err := packages.Load(cfg, "text-template-server/src/model")
		require.NoError(t, err)
		require.NotEmpty(t, pkgs)

		obj := pkgs[0].Types.Scope().Lookup("Order")
		require.NotNil(t, obj)

		named, ok := obj.Type().(*types.Named)
		require.True(t, ok)

		methods := NamedMethods(named)
		names := make([]string, len(methods))
		for i, m := range methods {
			names[i] = m.Name
		}

		assert.Contains(t, names, "DisplayName")
		assert.Contains(t, names, "Summary")
		assert.Contains(t, names, "Format")
		assert.Contains(t, names, "Oper")
		assert.NotContains(t, names, "badReturn")
		assert.NotContains(t, names, "wrongSecond")

		var format MethodType
		for _, m := range methods {
			if m.Name == "Format" {
				format = m
				break
			}
		}
		require.Len(t, format.Params, 1)
		assert.Equal(t, "currency", format.Params[0].Name)
		assert.Equal(t, "string", format.Params[0].TypeName)
	})

	t.Run("filters methods with invalid result count", func(t *testing.T) {
		pkg := types.NewPackage("example.test/synth", "synth")
		typeObj := types.NewTypeName(token.NoPos, pkg, "Synth", nil)
		underlying := types.NewStruct(nil, nil)

		zeroResultsSig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(),
			false,
		)
		oneResultSig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])),
			false,
		)
		threeResultsSig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.String]),
				types.NewVar(token.NoPos, pkg, "", types.Typ[types.Int]),
				types.NewVar(token.NoPos, pkg, "", types.Universe.Lookup("error").Type()),
			),
			false,
		)

		named := types.NewNamed(typeObj, underlying, []*types.Func{
			types.NewFunc(token.NoPos, pkg, "NoResult", zeroResultsSig),
			types.NewFunc(token.NoPos, pkg, "Good", oneResultSig),
			types.NewFunc(token.NoPos, pkg, "TooMany", threeResultsSig),
		})

		methods := NamedMethods(named)
		require.Len(t, methods, 1)
		assert.Equal(t, "Good", methods[0].Name)
	})
}

func TestStructFields(t *testing.T) {
	t.Run("returns nil for non-struct named type", func(t *testing.T) {
		pkg := types.NewPackage("example.test/nonstruct", "nonstruct")
		named := types.NewNamed(
			types.NewTypeName(token.NoPos, pkg, "AliasInt", nil),
			types.Typ[types.Int],
			nil,
		)
		assert.Nil(t, StructFields(named))
	})

	t.Run("returns only exported fields and preserves embedded flag", func(t *testing.T) {
		pkg := types.NewPackage("example.test/fields", "fields")
		embeddedType := types.NewNamed(
			types.NewTypeName(token.NoPos, pkg, "Embed", nil),
			types.NewStruct(nil, nil),
			nil,
		)

		st := types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, pkg, "Public", types.Typ[types.String], false),
			types.NewField(token.NoPos, pkg, "private", types.Typ[types.Int], false),
			types.NewField(token.NoPos, pkg, "Embed", embeddedType, true),
		}, nil)

		named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Container", nil), st, nil)
		fields := StructFields(named)

		require.Len(t, fields, 2)
		assert.Equal(t, "Public", fields[0].Name)
		assert.False(t, fields[0].Embedded)
		assert.Equal(t, "Embed", fields[1].Name)
		assert.True(t, fields[1].Embedded)
	})
}
