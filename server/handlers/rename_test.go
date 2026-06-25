package handlers

import (
	gotypes "go/types"
	"testing"
	serverTypes "text-template-server/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"golang.org/x/tools/go/packages"
)

func renameParams(uri string, pos protocol.Position, newName string) *protocol.RenameParams {
	return &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
		NewName: newName,
	}
}

func editLen(e protocol.TextEdit) uint32 {
	return e.Range.End.Character - e.Range.Start.Character
}

func chainRootType(t *testing.T) *serverTypes.Tree {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedFiles,
		Dir:  "../../test/resources/definition-tests-server",
	}
	pkgs, err := packages.Load(cfg, "text-template-server/src/model")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	pkg := pkgs[0]
	obj := pkg.Types.Scope().Lookup("ChainRoot")
	require.NotNil(t, obj)
	named, ok := obj.Type().(*gotypes.Named)
	require.True(t, ok)
	return &serverTypes.Tree{
		DotType: named,
		Pkg:     pkg.Types,
		Fset:    pkg.Fset,
	}
}

func setRenameDoc(t *testing.T, uri string, tc renameTestCase, orderType, chainType *serverTypes.Tree) {
	t.Helper()
	switch tc.typeKind {
	case "source":
		setDocFromSource(t, uri, tc.src)
	case "order":
		setDocWithType(t, uri, tc.src, orderType)
	case "chain":
		setDocWithType(t, uri, tc.src, chainType)
	default:
		store.Set(uri, tc.src)
	}
	t.Cleanup(func() { store.Delete(uri) })
}

func assertRename(t *testing.T, uri string, tc renameTestCase, char uint32) {
	t.Helper()
	edit, err := Rename(nil, renameParams(uri, position(tc.line, char), tc.newName))
	require.NoError(t, err)

	if tc.wantNil {
		assert.Nil(t, edit)
		return
	}
	require.NotNil(t, edit)

	edits := edit.Changes[uri]
	require.Len(t, edits, tc.wantCount)
	for _, e := range edits {
		assert.Equal(t, tc.wantText, e.NewText)
		if tc.wantLen > 0 {
			assert.Equal(t, tc.wantLen, editLen(e))
		}
		if tc.wantStarts != nil {
			assert.Contains(t, tc.wantStarts, e.Range.Start.Character)
		}
	}
}

func TestRename(t *testing.T) {
	orderType := definitionOrderType(t)
	chainType := chainRootType(t)
	const uri = "file:///rename.tmpl"

	for _, tc := range renameTestCases {
		t.Run(tc.name, func(t *testing.T) {
			chars := tc.chars
			if chars == nil {
				chars = []uint32{tc.char}
			}
			for _, char := range chars {
				setRenameDoc(t, uri, tc, orderType, chainType)
				assertRename(t, uri, tc, char)
			}
		})
	}
}
