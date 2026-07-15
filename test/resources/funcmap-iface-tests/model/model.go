// Package model declares SignalInstance, a concrete type that satisfies
// iface.HasType via a pointer receiver.
package model

import (
	"text-template-server/funcmap-iface-tests/iface"
)

// SignalInstance is the concrete value passed to funcmap functions in
// templates. It implements iface.HasType.
type SignalInstance struct {
	Name string
}

// Kind reports the kind of the signal. Its receiver is a pointer so that
// only *SignalInstance satisfies iface.HasType. The return type is
// iface.Kind — a named type from another package — so the method signature
// references a *types.Named whose identity must be shared between the
// funcmap loader and the gotype-hint loader for types.Implements to succeed.
func (s *SignalInstance) Kind() iface.Kind { return iface.Kind("signal") }

// Root is the dot type used by the test template. It exposes a
// *SignalInstance so the template can pass it to a funcmap function.
type Root struct {
	Signal *SignalInstance
}
