package types

import (
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const genericFixtureRoot = "../../test/resources/typehints-tests"

// fieldType returns the type of the named field on a (possibly instantiated)
// named struct type, or nil when absent.
func fieldType(t types.Type, name string) types.Type {
	st, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return nil
	}
	s, ok := st.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	for i := 0; i < s.NumFields(); i++ {
		if s.Field(i).Name() == name {
			return s.Field(i).Type()
		}
	}
	return nil
}

func TestLoadGenericInstantiation_SingleParam(t *testing.T) {
	// Bare type argument resolved in the base type's own package.
	lt, err := LoadTypeFromHint(
		"text-template-server/src/model.View[Instance]", genericFixtureRoot,
	)
	require.NoError(t, err)
	require.NotNil(t, lt.DotType)

	named, ok := lt.DotType.(*types.Named)
	require.True(t, ok, "expected an instantiated *types.Named")
	require.Equal(t, 1, named.TypeArgs().Len())
	assert.Equal(t, "Instance", named.TypeArgs().At(0).(*types.Named).Obj().Name())

	// The T-typed field must now be the concrete Instance type, not a type param.
	mt := fieldType(lt.DotType, "Model")
	require.NotNil(t, mt)
	mn, ok := types.Unalias(mt).(*types.Named)
	require.True(t, ok, "Model field should resolve to a concrete named type")
	assert.Equal(t, "Instance", mn.Obj().Name())
}

func TestLoadGenericInstantiation_QualifiedArg(t *testing.T) {
	lt, err := LoadTypeFromHint(
		"text-template-server/src/model.View[text-template-server/src/model.Order]",
		genericFixtureRoot,
	)
	require.NoError(t, err)
	named, ok := lt.DotType.(*types.Named)
	require.True(t, ok)
	require.Equal(t, 1, named.TypeArgs().Len())
	assert.Equal(t, "Order", named.TypeArgs().At(0).(*types.Named).Obj().Name())
}

func TestLoadGenericInstantiation_MultiParam(t *testing.T) {
	lt, err := LoadTypeFromHint(
		"text-template-server/src/model.Pair[Instance, text-template-server/src/model.Order]",
		genericFixtureRoot,
	)
	require.NoError(t, err)
	named, ok := lt.DotType.(*types.Named)
	require.True(t, ok)
	require.Equal(t, 2, named.TypeArgs().Len())
	assert.Equal(t, "Instance", named.TypeArgs().At(0).(*types.Named).Obj().Name())
	assert.Equal(t, "Order", named.TypeArgs().At(1).(*types.Named).Obj().Name())
}

func TestGenericHint_LooksLikeTypeExpr(t *testing.T) {
	assert.True(t, looksLikeTypeExpr("pkg/path.View[Instance]"))
	assert.True(t, looksLikeTypeExpr("pkg/path.Pair[A, pkg/path.B]"))
	assert.True(t, looksLikeTypeExpr("View[Instance]"))
}

// TestLoadGeneric_CrossPackageFsets verifies that a hint whose type argument
// lives in a different package than the generic base records a FileSet for each
// package, so a field reached through the type argument resolves to the correct
// source file (the cross-package go-to-definition case).
func TestLoadGeneric_CrossPackageFsets(t *testing.T) {
	lt, err := LoadTypeFromHint(
		"text-template-server/src/model.View[text-template-server/src/othermodel.Gadget]",
		genericFixtureRoot,
	)
	require.NoError(t, err)
	require.NotNil(t, lt.Fsets)
	// Both the base package (model) and the type-argument package (othermodel)
	// must be represented.
	require.GreaterOrEqual(t, len(lt.Fsets), 2)

	// The T-typed field resolves to Gadget (othermodel).
	modelField := fieldType(lt.DotType, "Model")
	require.NotNil(t, modelField)
	gadget, ok := types.Unalias(modelField).(*types.Named)
	require.True(t, ok)
	require.Equal(t, "Gadget", gadget.Obj().Name())

	// A field of Gadget belongs to othermodel; its position must resolve via the
	// per-package FileSet map to a file in that package.
	serial, _, _ := types.LookupFieldOrMethod(gadget, true, gadget.Obj().Pkg(), "Serial")
	require.NotNil(t, serial)
	fs := lt.Fsets[serial.Pkg()]
	require.NotNil(t, fs, "othermodel package must have a recorded FileSet")
	pos := fs.Position(serial.Pos())
	assert.Contains(t, pos.Filename, "othermodel")
}

func TestHintRefAt(t *testing.T) {
	comment := "/*gotype: cg/template.View[cg/model/controlmodel.Instance] */"
	// Cursor over the base type token.
	baseIdx := strings.Index(comment, "template.View")
	ip, tn, ok := HintRefAt(comment, baseIdx+1)
	require.True(t, ok)
	assert.Equal(t, "cg/template", ip)
	assert.Equal(t, "View", tn)

	// Cursor over the type-argument token.
	argIdx := strings.Index(comment, "controlmodel.Instance")
	ip, tn, ok = HintRefAt(comment, argIdx+1)
	require.True(t, ok)
	assert.Equal(t, "cg/model/controlmodel", ip)
	assert.Equal(t, "Instance", tn)

	// Cursor on whitespace / outside any token.
	_, _, ok = HintRefAt(comment, len(comment)-1)
	assert.False(t, ok)
}

func TestLocateTypeDecl(t *testing.T) {
	pos, ok := LocateTypeDecl("text-template-server/src/othermodel", "Gadget", genericFixtureRoot)
	require.True(t, ok)
	assert.Contains(t, pos.Filename, "othermodel")
	assert.Positive(t, pos.Line)

	_, ok = LocateTypeDecl("text-template-server/src/othermodel", "DoesNotExist", genericFixtureRoot)
	assert.False(t, ok)
}
