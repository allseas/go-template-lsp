package types

import (
	"go/token"
	"go/types"
)

// builtinPkg is a synthetic package used to own the builtin *types.Func objects.
var builtinPkg = types.NewPackage("text/template/builtin", "tmplbuiltin")

// BuiltinFuncs returns a map of all Go text/template builtin function names to
// *types.Func values whose signatures reflect the documented contracts.
//
// Signatures use interface{} ("any") for polymorphic positions. Variadic builtins
// use a variadic signature so that argument-count checking downstream is accurate.
func BuiltinFuncs() map[string]*types.Func {
	anyT := types.NewInterfaceType(nil, nil).Complete() // everything implements the empty interface
	intT := types.Typ[types.Int]
	boolT := types.Typ[types.Bool]
	stringT := types.Typ[types.String]

	// helpers
	v := func(name string, t types.Type) *types.Var {
		return types.NewVar(token.NoPos, nil, name, t)
	}
	ret := func(t types.Type) *types.Tuple {
		return types.NewTuple(types.NewVar(token.NoPos, nil, "", t))
	}
	sig := func(params []*types.Var, result types.Type) *types.Signature {
		return types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), ret(result), false)
	}

	// sigV builds a variadic signature; the last element of params is used as the
	// variadic element type and is automatically wrapped in a slice.
	sigV := func(params []*types.Var, result types.Type) *types.Signature {
		if len(params) == 0 {
			panic("sigV: need at least one parameter")
		}
		last := params[len(params)-1]
		sliceVar := types.NewVar(last.Pos(), last.Pkg(), last.Name(), types.NewSlice(last.Type()))
		params[len(params)-1] = sliceVar
		return types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), ret(result), true)
	}
	fn := func(name string, s *types.Signature) *types.Func {
		return types.NewFunc(token.NoPos, builtinPkg, name, s)
	}

	return map[string]*types.Func{
		// Numeric output
		"len": fn("len", sig([]*types.Var{v("v", anyT)}, intT)),

		// Boolean output
		"not": fn("not", sig([]*types.Var{v("a", anyT)}, boolT)),
		"and": fn("and", sigV([]*types.Var{v("arg0", anyT), v("args", anyT)}, anyT)),
		"or":  fn("or", sigV([]*types.Var{v("arg0", anyT), v("args", anyT)}, anyT)),

		// Comparison (bool output, variadic)
		"eq": fn("eq", sigV([]*types.Var{v("arg1", anyT), v("arg2", anyT)}, boolT)),
		"ne": fn("ne", sigV([]*types.Var{v("arg1", anyT), v("arg2", anyT)}, boolT)),
		"lt": fn("lt", sig([]*types.Var{v("arg1", anyT), v("arg2", anyT)}, boolT)),
		"le": fn("le", sig([]*types.Var{v("arg1", anyT), v("arg2", anyT)}, boolT)),
		"gt": fn("gt", sig([]*types.Var{v("arg1", anyT), v("arg2", anyT)}, boolT)),
		"ge": fn("ge", sig([]*types.Var{v("arg1", anyT), v("arg2", anyT)}, boolT)),

		// String output
		"print":    fn("print", sigV([]*types.Var{v("a", anyT)}, stringT)),
		"printf":   fn("printf", sigV([]*types.Var{v("format", stringT), v("a", anyT)}, stringT)),
		"println":  fn("println", sigV([]*types.Var{v("a", anyT)}, stringT)),
		"html":     fn("html", sigV([]*types.Var{v("s", anyT)}, stringT)),
		"js":       fn("js", sigV([]*types.Var{v("s", anyT)}, stringT)),
		"urlquery": fn("urlquery", sigV([]*types.Var{v("s", anyT)}, stringT)),

		// Dynamic / untyped output
		"call":  fn("call", sigV([]*types.Var{v("fn", anyT), v("args", anyT)}, anyT)),
		"index": fn("index", sigV([]*types.Var{v("item", anyT), v("indices", anyT)}, anyT)),
		"slice": fn("slice", sigV([]*types.Var{v("item", anyT), v("indices", anyT)}, anyT)),
	}
}
