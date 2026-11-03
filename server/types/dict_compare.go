package types

import "go/types"

// The go/types predicates (Identical, AssignableTo, ConvertibleTo, Implements)
// panic("unreachable") when the type they switch on is foreign to go/types,
// which *DictType is. A dict can sit anywhere inside a type: `[]map{...}`,
// `*map{...}`, `View[map{...}]`. Every comparison that may see a hint-derived
// type must therefore go through the DictSafe* functions below; forbidigo
// (see .golangci.yml) rejects direct calls outside this file.

// DictSafeIdentical is types.Identical that tolerates *DictType. Two dicts are
// identical when they have the same keys with identical value types.
func DictSafeIdentical(x, y types.Type) bool {
	if x == y {
		return true
	}
	if !containsDict(x) && !containsDict(y) {
		return types.Identical(x, y)
	}
	if x == nil || y == nil {
		return false
	}
	x, y = types.Unalias(x), types.Unalias(y)
	xd, xIsDict := x.(*DictType)
	yd, yIsDict := y.(*DictType)
	if xIsDict || yIsDict {
		return xIsDict && yIsDict && len(xd.Fields) == len(yd.Fields) &&
			dictCovers(xd, yd, DictSafeIdentical)
	}
	return sameShape(x, y, DictSafeIdentical)
}

// DictSafeAssignableTo is types.AssignableTo that tolerates *DictType.
func DictSafeAssignableTo(got, want types.Type) bool {
	if !containsDict(got) && !containsDict(want) {
		return types.AssignableTo(got, want)
	}
	if got == nil || want == nil {
		return false
	}
	if iface, ok := want.Underlying().(*types.Interface); ok {
		return iface.Empty() || DictSafeImplements(got, iface)
	}
	return dictCompatible(got, want)
}

// DictSafeConvertibleTo is types.ConvertibleTo that tolerates *DictType. Go has
// no conversions involving maps beyond assignability, so for dict-bearing types
// it is the same relation as DictSafeAssignableTo.
func DictSafeConvertibleTo(got, want types.Type) bool {
	if !containsDict(got) && !containsDict(want) {
		return types.ConvertibleTo(got, want)
	}
	return DictSafeAssignableTo(got, want)
}

// DictSafeImplements is types.Implements that tolerates *DictType. For
// dict-bearing types each interface method is looked up on got and its
// signature compared with DictSafeIdentical. A dict itself has no methods.
func DictSafeImplements(got types.Type, iface *types.Interface) bool {
	if !containsDict(got) && !containsDict(iface) {
		return types.Implements(got, iface)
	}
	if got == nil || !iface.IsMethodSet() {
		return false
	}
	for m := range iface.Methods() {
		obj, _, _ := types.LookupFieldOrMethod(got, false, m.Pkg(), m.Name())
		f, ok := obj.(*types.Func)
		if !ok || !DictSafeIdentical(f.Type(), m.Type()) {
			return false
		}
	}
	return true
}

// dictCompatible reports whether a value of type got can stand where want is
// expected, for types nested inside a composite (where Go requires identity).
// It is identity, relaxed only for dicts as described above.
func dictCompatible(got, want types.Type) bool {
	if got == want {
		return true
	}
	if !containsDict(got) && !containsDict(want) {
		return types.Identical(got, want)
	}
	if got == nil || want == nil {
		return false
	}
	got, want = types.Unalias(got), types.Unalias(want)
	gd, gotIsDict := got.(*DictType)
	wd, wantIsDict := want.(*DictType)
	switch {
	case gotIsDict && wantIsDict:
		return dictCovers(gd, wd, dictValueCompatible)
	case gotIsDict:
		return isMapStringAny(want)
	case wantIsDict:
		return isMapStringAny(got)
	}
	return sameShape(got, want, dictCompatible)
}

// dictValueCompatible compares a dict value against an expected dict value
// with the same tolerance as a key-by-key template argument check: unknown
// types pass, and a pointer is accepted where its element is expected.
func dictValueCompatible(got, want types.Type) bool {
	if IsEmptyInterface(got) || IsEmptyInterface(want) {
		return true
	}
	return DictSafeAssignableTo(got, want) || pointerElemMatches(got, want)
}

// dictCovers reports whether got has every key of want, with values related
// by rel.
func dictCovers(got, want *DictType, rel func(got, want types.Type) bool) bool {
	for key, wantV := range want.Fields {
		gotV, ok := got.Fields[key]
		if !ok || !rel(gotV, wantV) {
			return false
		}
	}
	return true
}

// sameShape reports whether x and y are the same kind of composite type with
// components related by elem. Map keys always require identity. Both types
// must already be unaliased.
func sameShape(x, y types.Type, elem func(x, y types.Type) bool) bool {
	switch xt := x.(type) {
	case *types.Pointer:
		yt, ok := y.(*types.Pointer)
		return ok && elem(xt.Elem(), yt.Elem())
	case *types.Slice:
		yt, ok := y.(*types.Slice)
		return ok && elem(xt.Elem(), yt.Elem())
	case *types.Array:
		yt, ok := y.(*types.Array)
		return ok && xt.Len() == yt.Len() && elem(xt.Elem(), yt.Elem())
	case *types.Map:
		yt, ok := y.(*types.Map)
		return ok && DictSafeIdentical(xt.Key(), yt.Key()) && elem(xt.Elem(), yt.Elem())
	case *types.Chan:
		yt, ok := y.(*types.Chan)
		return ok && xt.Dir() == yt.Dir() && elem(xt.Elem(), yt.Elem())
	case *types.Named:
		yt, ok := y.(*types.Named)
		return ok && xt.Origin().Obj() == yt.Origin().Obj() &&
			typeListsMatch(xt.TypeArgs(), yt.TypeArgs(), elem)
	case *types.Tuple:
		yt, ok := y.(*types.Tuple)
		return ok && tuplesMatch(xt, yt, elem)
	case *types.Signature:
		yt, ok := y.(*types.Signature)
		return ok && xt.Variadic() == yt.Variadic() &&
			tuplesMatch(xt.Params(), yt.Params(), elem) &&
			tuplesMatch(xt.Results(), yt.Results(), elem)
	}
	return false
}

func typeListsMatch(x, y *types.TypeList, elem func(x, y types.Type) bool) bool {
	if x.Len() != y.Len() {
		return false
	}
	for i := range x.Len() {
		if !elem(x.At(i), y.At(i)) {
			return false
		}
	}
	return true
}

func tuplesMatch(x, y *types.Tuple, elem func(x, y types.Type) bool) bool {
	if x.Len() != y.Len() {
		return false
	}
	for i := range x.Len() {
		if !elem(x.At(i).Type(), y.At(i).Type()) {
			return false
		}
	}
	return true
}

// isMapStringAny reports whether t's underlying type is map[string]any, the
// runtime type a dict describes.
func isMapStringAny(t types.Type) bool {
	m, ok := t.Underlying().(*types.Map)
	if !ok {
		return false
	}
	k, ok := m.Key().Underlying().(*types.Basic)
	return ok && k.Kind() == types.String && IsEmptyInterface(m.Elem())
}

// containsDict reports whether t mentions a *DictType anywhere a go/types
// predicate could recurse into. Named types are checked through their type
// arguments only: a non-generic named type comes from Go source and cannot
// contain a dict.
func containsDict(t types.Type) bool {
	switch t := types.Unalias(t).(type) {
	case *DictType:
		return true
	case *types.Pointer:
		return containsDict(t.Elem())
	case *types.Slice:
		return containsDict(t.Elem())
	case *types.Array:
		return containsDict(t.Elem())
	case *types.Chan:
		return containsDict(t.Elem())
	case *types.Map:
		return containsDict(t.Key()) || containsDict(t.Elem())
	case *types.Named:
		for targ := range t.TypeArgs().Types() {
			if containsDict(targ) {
				return true
			}
		}
	case *types.Tuple:
		return tupleContainsDict(t)
	case *types.Signature:
		return tupleContainsDict(t.Params()) || tupleContainsDict(t.Results())
	case *types.Struct:
		for f := range t.Fields() {
			if containsDict(f.Type()) {
				return true
			}
		}
	case *types.Interface:
		for m := range t.Methods() {
			if containsDict(m.Type()) {
				return true
			}
		}
	}
	return false
}

func tupleContainsDict(t *types.Tuple) bool {
	for v := range t.Variables() {
		if containsDict(v.Type()) {
			return true
		}
	}
	return false
}
