// Package iface declares HasType, a named interface referenced by both a
// workspace funcmap function and by a concrete type in the model package.
// Its purpose in this fixture is to exercise interface satisfaction across
// packages loaded by two different code paths (LoadGlobalFuncs and the
// gotype-hint loader).
package iface

// Kind labels a category of signal. It is a named type deliberately declared
// in this package so that the signature of HasType.Kind references a
// *types.Named owned by iface. Without a shared package cache the funcmap
// loader and the gotype-hint loader would each own a distinct *types.Named
// for Kind, and types.Implements would then return false when comparing an
// interface method against a concrete method that returns Kind.
type Kind string

// HasType is a minimal named interface. Templates receive concrete values
// implementing this interface as arguments to funcmap functions.
type HasType interface {
	Kind() Kind
}
