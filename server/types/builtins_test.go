package types

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuiltinFuncs(t *testing.T) {
	funcs := BuiltinFuncs()

	assert.Equal(t, 19, len(funcs), "unexpected number of builtin functions")

	// test types of some builtins
	assert.Contains(t, funcs, "len")
	assert.Contains(t, funcs, "and")
	assert.Contains(t, funcs, "or")
	assert.Contains(t, funcs, "eq")
	assert.Contains(t, funcs, "ne")
	assert.Contains(t, funcs, "lt")
	assert.Contains(t, funcs, "gt")

	for name, fn := range funcs {
		if assert.NotNil(t, fn, "builtin %q should have a non-nil *types.Func", name) {
			sig := fn.Type().(*types.Signature)
			assert.NotNil(t, sig, "builtin %q should have a *types.Signature type", name)
			assert.NotNil(t, sig.Params(), "builtin %q should have parameters", name)
			assert.NotNil(t, sig.Results(), "builtin %q should have results", name)
		}
	}

	// test len types
	lenSig := funcs["len"].Type().(*types.Signature)
	assert.Equal(t, 1, lenSig.Params().Len(), "len should have exactly one parameter")
	assert.Equal(
		t,
		"interface{}",
		lenSig.Params().At(0).Type().String(),
		"len parameter should be of type any",
	)
	assert.Equal(t, "int", lenSig.Results().At(0).Type().String(), "len should return int")

	// test and types
	andSig := funcs["and"].Type().(*types.Signature)
	assert.True(t, andSig.Variadic(), "and should be variadic")
	assert.Equal(t, 2, andSig.Params().Len(), "and should have at least two parameters")
	assert.Equal(
		t,
		"interface{}",
		andSig.Params().At(0).Type().String(),
		"and first parameter should be of type any",
	)
	assert.Equal(
		t,
		"[]interface{}",
		andSig.Params().At(1).Type().String(),
		"and variadic parameter should be of type []interface{}",
	)
	assert.Equal(
		t,
		"interface{}",
		andSig.Results().At(0).Type().String(),
		"and should return interface{}",
	)

	// test eq types
	eqSig := funcs["eq"].Type().(*types.Signature)
	assert.True(t, eqSig.Variadic(), "eq should be variadic")
	assert.Equal(t, 2, eqSig.Params().Len(), "eq should have at least two parameters")
	assert.Equal(
		t,
		"interface{}",
		eqSig.Params().At(0).Type().String(),
		"eq first parameter should be of type interface{}",
	)
	assert.Equal(
		t,
		"[]interface{}",
		eqSig.Params().At(1).Type().String(),
		"eq variadic parameter should be of type []interface{}",
	)
	assert.Equal(t, "bool", eqSig.Results().At(0).Type().String(), "eq should return bool")
}
