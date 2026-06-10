package handlers

import (
	"os"
	"testing"
	"text-template-server/types"
)

// TestMain seeds the global function cache with builtins before any handler
// test runs, mirroring what the LSP initialize handler does in production.
func TestMain(m *testing.M) {
	types.SetGlobalFuncs(types.BuiltinFuncs())
	os.Exit(m.Run())
}
