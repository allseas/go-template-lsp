package types

import (
	"go/types"
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
