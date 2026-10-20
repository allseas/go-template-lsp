// Package outer is used as a generic type argument from a different package
// than the generic base (cg/model.View) and imports package inner.
package outer

import "cg/repro/inner"

// Thing has a field whose type lives in package inner.
type Thing struct {
	C inner.Inner
}
