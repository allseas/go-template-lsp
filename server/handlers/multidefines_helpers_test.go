package handlers

import (
	gotypes "go/types"
	"testing"

	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"golang.org/x/tools/go/packages"
)

// loadModelTypes loads the test typehints model package and returns Trees keyed by Go type name.
func loadModelTypes(t *testing.T, names ...string) map[string]*serverTypes.Tree {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedFiles,
		Dir:  "../../test/resources/typehints-tests",
	}
	pkgs, err := packages.Load(cfg, "text-template-server/src/model")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	pkg := pkgs[0]
	out := make(map[string]*serverTypes.Tree, len(names))
	for _, name := range names {
		obj := pkg.Types.Scope().Lookup(name)
		require.NotNil(t, obj, "type %q not found in test model package", name)
		named, ok := obj.Type().(*gotypes.Named)
		require.True(t, ok, "type %q is not a Named type", name)
		out[name] = &serverTypes.Tree{DotType: named, Pkg: pkg.Types, Fset: pkg.Fset}
	}
	return out
}

// setDocMulti inserts a document into the store with pre-populated parse trees and per-tree
// loaded types, bypassing the workspace-root-dependent type loading. perTree maps a parse-tree
// name (the {{define}} name, or the root tree name) to its loaded Go type.
func setDocMulti(t *testing.T, uri, src string, perTree map[string]*serverTypes.Tree) {
	t.Helper()
	tree, treeSet, err := tryParse(src)
	require.NoError(t, err)

	loadedTypes := make(map[string]*serverTypes.Tree, len(treeSet))
	typedTrees := make(map[string]*serverTypes.Tree, len(treeSet))
	for name, tr := range treeSet {
		lt := perTree[name]
		if lt != nil {
			loadedTypes[name] = lt
		}
		typedTrees[name] = buildTypedTree(tr, lt, nil)
	}
	var typed *serverTypes.Tree
	if tree != nil {
		typed = typedTrees[tree.Name]
	}

	store.mu.Lock()
	store.docs[uri] = &document{
		text:        src,
		tree:        tree,
		trees:       treeSet,
		typedTree:   typed,
		loadedTypes: loadedTypes,
		typedTrees:  typedTrees,
	}
	store.mu.Unlock()
}

// posOfSubStr returns the protocol.Position pointing to the first byte of the n-th
// (0-based) occurrence of substr in src.
func posOfSubStr(t *testing.T, src, substr string, occurrence int) protocol.Position {
	t.Helper()
	idx := -1
	count := 0
	for i := 0; i <= len(src)-len(substr); i++ {
		if src[i:i+len(substr)] == substr {
			if count == occurrence {
				idx = i
				break
			}
			count++
		}
	}
	require.GreaterOrEqual(t, idx, 0, "substring %q (occurrence %d) not found", substr, occurrence)

	line := uint32(0)
	char := uint32(0)
	for i := 0; i < idx; i++ {
		if src[i] == '\n' {
			line++
			char = 0
		} else {
			char++
		}
	}
	return protocol.Position{Line: line, Character: char}
}

// multiDefinesTemplate is the classic multi-define template
const multiDefinesTemplate = `{{- /*gotype: text-template-server/src/model.Address*/ -}}
{{ .Country }}
{{- define "OrderTpl" -}}
{{- /*gotype: text-template-server/src/model.Order*/ -}}
Order: {{ .CustomerName }} ({{ .ID }})
{{- end -}}
{{ .Zip }}
{{- define "AddressTpl" -}}
{{- /*gotype: text-template-server/src/model.Address*/ -}}
Address: {{ .Street }}, {{ .City }}
{{- end -}}
{{- define "NoHint" -}}
{{ $local := . }}
no hint here {{ $local }}
{{- end -}}
{{ .Country }}
`
