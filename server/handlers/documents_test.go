package handlers

import (
	"strings"
	"sync"
	"testing"
	"text-template-server/types"

	parse "text-template-parser"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDocumentStore(t *testing.T) {
	ds := &documentStore{docs: make(map[string]*document)}
	uri := "file:///test-document.txt"
	content := "Initial Content"

	t.Run("Set and Get", func(t *testing.T) {
		ds.Set(uri, content)
		val, ok := ds.Get(uri)

		assert.True(t, ok, "Document should exist in the store")
		assert.Equal(t, content, val.text)
	})

	t.Run("Overwrite Content", func(t *testing.T) {
		newContent := "Updated Content"
		ds.Set(uri, newContent)
		val, ok := ds.Get(uri)

		assert.True(t, ok)
		assert.Equal(t, newContent, val.text, "Content should match the updated value")
	})

	t.Run("Remove Document", func(t *testing.T) {
		ds.Remove(uri)
		_, ok := ds.Get(uri)

		assert.False(t, ok, "Document should no longer exist after removal")
	})

	// ai
	t.Run("Concurrent Access", func(t *testing.T) {
		uris := []string{"file:///doc1.txt", "file:///doc2.txt", "file:///doc3.txt"}
		var wg sync.WaitGroup

		for _, u := range uris {
			wg.Add(1)
			go func(uri string) {
				defer wg.Done()
				ds.Set(uri, "Content for "+uri)
			}(u)
		}
		wg.Wait() // Wait for all Set operations to complete

		for _, u := range uris {
			val, ok := ds.Get(u)
			assert.True(t, ok, "Document should exist in the store")
			assert.Equal(t, "Content for "+u, val.text)
		}
	})

	// ai
	t.Run("Parse Errors", func(t *testing.T) {
		invalidContent := "{{ unclosed"
		ds.Set(uri, invalidContent)
		val, ok := ds.Get(uri)

		assert.True(t, ok)
		assert.Equal(t, invalidContent, val.text, "Content should be stored even if parsing fails")
		// tree is not nil with the new parser
		assert.NotNil(t, val.tree, "Tree should not be nil if parsing fails")
	})

	// ai
	t.Run("Parse Valid Content", func(t *testing.T) {
		validContent := "{{ . }}"
		ds.Set(uri, validContent)
		val, ok := ds.Get(uri)

		assert.True(t, ok)
		assert.Equal(t, validContent, val.text)
		assert.NotNil(t, val.tree, "Tree should be set for valid content")
	})

	// ai
	t.Run("Remove Nonexistent Document", func(t *testing.T) {
		nonexistentURI := "file:///nonexistent.txt"
		ds.Remove(nonexistentURI) // Should not panic
		_, ok := ds.Get(nonexistentURI)
		assert.False(t, ok, "Document should not exist in the store")
	})

	// ai
	t.Run("Empty Content", func(t *testing.T) {
		emptyContent := ""
		ds.Set(uri, emptyContent)
		val, ok := ds.Get(uri)

		assert.True(t, ok)
		assert.Equal(t, emptyContent, val.text, "Content should be stored even if it's empty")
		assert.NotNil(t, val.tree, "Tree should be set for empty content (valid template)")
	})

	// ai
	t.Run("Multiple Documents", func(t *testing.T) {
		uris := []string{"file:///docA.txt", "file:///docB.txt", "file:///docC.txt"}
		for _, u := range uris {
			ds.Set(u, "Content for "+u)
		}

		for _, u := range uris {
			val, ok := ds.Get(u)
			assert.True(t, ok)
			assert.Equal(t, "Content for "+u, val.text)
		}
	})

	// ai
	t.Run("Invalid URI", func(t *testing.T) {
		invalidURI := "not-a-valid-uri"
		ds.Set(invalidURI, "Some content")
		val, ok := ds.Get(invalidURI)

		assert.True(t, ok, "Even invalid URIs should be stored")
		assert.Equal(t, "Some content", val.text)
	})

	// ai
	t.Run("Delete", func(t *testing.T) {
		uriToDelete := "file:///to-delete.txt"
		ds.Set(uriToDelete, "Content to delete")
		ds.Delete(uriToDelete)
		_, ok := ds.Get(uriToDelete)
		assert.False(t, ok, "Document should be deleted from the store")
	})
}

// ai
func TestDocumentHintFoundBeyondFirstLine(t *testing.T) {
	t.Run("root hint several lines into the document", func(t *testing.T) {
		src := "Some preamble text\n" +
			"more preamble\n" +
			"{{- /*gotype: cg/model.Root*/ -}}\n" +
			"hello {{ . }}\n"

		tree, treeSet, err := tryParse(src)
		assert.NoError(t, err)
		assert.NotNil(t, tree)

		hints := types.FindTreeHints(src, treeSet)
		hint, ok := hints[tree.Name]
		assert.True(t, ok, "expected a hint to be found for the root tree")
		assert.Equal(t, "cg/model.Root", hint.Type)

		wantLine := strings.Count(src[:strings.Index(src, "gotype:")], "\n") + 1
		assert.Equal(
			t,
			wantLine,
			hint.Line,
			"hint line should reflect its actual line in the document, not just line 1",
		)
	})

	t.Run("define-block hint several lines into the define body", func(t *testing.T) {
		src := "{{- define \"A\" -}}\n" +
			"line one of A\n" +
			"line two of A\n" +
			"{{- /*gotype: cg/model.A*/ -}}\n" +
			"line four of A\n" +
			"{{- /*gotype: cg/model.B*/ -}}\n" +
			"{{- end -}}\n"

		tree, treeSet, err := tryParse(src)
		assert.NoError(t, err)
		assert.NotNil(t, tree)
		assert.Contains(t, treeSet, "A")
		assert.NotContains(t, treeSet, "B")

		hints := types.FindTreeHints(src, treeSet)
		hintA, ok := hints["A"]
		assert.True(t, ok, "expected a hint to be found for define A")
		assert.Equal(t, "cg/model.A", hintA.Type)

		wantLine := strings.Count(src[:strings.Index(src, "gotype:")], "\n") + 1
		assert.Equal(
			t,
			wantLine,
			hintA.Line,
			"hint line should reflect its actual line within the document",
		)

		// The hint sits inside define A's span, so the root tree must not
		// also pick it up.
		rootHint, hasRoot := hints[tree.Name]
		assert.False(
			t,
			hasRoot && rootHint.Type != "",
			"root should not pick up define A's hint, got: %+v",
			rootHint,
		)
	})
}

func TestDocumentMultipleDefines(t *testing.T) {
	// ai
	src := "{{- define \"A\"}}\n" +
		"{{- /*gotype: cg/model.A*/}}\n" +
		"alpha-body\n" +
		"{{- end }}\n" +
		"{{- define \"B\" }}\n" +
		"{{- /*gotype: cg/model.B*/}}\n" +
		"beta-body\n" +
		"{{- end }}\n"

	tree, treeSet, err := tryParse(src)
	assert.NoError(t, err)
	assert.NotNil(t, tree)
	// 3 entries: root + A + B
	assert.Contains(t, treeSet, "A")
	assert.Contains(t, treeSet, "B")
	assert.Contains(t, treeSet, tree.Name)

	// per-tree type hint lookup
	hints := types.FindTreeHints(src, treeSet)
	assert.Equal(t, "", hints[tree.Name].Type, "no hint expected on first line of file")
	assert.Equal(t, "cg/model.A", hints["A"].Type)
	assert.Equal(t, "cg/model.B", hints["B"].Type)

	doc := &document{text: src, tree: tree, trees: treeSet}

	// treeAt: positions inside each define body -> correct define tree
	offA := strings.Index(src, "alpha-body")
	offB := strings.Index(src, "beta-body")
	assert.Equal(t, treeSet["A"], doc.treeAt(parse.Pos(offA)), "offset inside define A body")
	assert.Equal(t, treeSet["B"], doc.treeAt(parse.Pos(offB)), "offset inside define B body")

	// treeAt: position at the very start of the file (on the {{define}} directive,
	// before any define's content) -> root template
	assert.Equal(
		t,
		tree,
		doc.treeAt(parse.Pos(0)),
		"offset at start of file (before define A content) should be root",
	)

	// treeAt: position at the {{define "B"}} directive (after A's {{end}},
	// before B's content) -> root template
	offDefineB := strings.Index(src, "{{- define \"B\"")
	assert.Greater(t, offDefineB, 0, "sanity: found define B directive")
	assert.Equal(
		t,
		tree,
		doc.treeAt(parse.Pos(offDefineB)),
		"offset at define B directive should be root",
	)

	// treeAt: position at the very last byte of the source (after all {{end}}s)
	// -> root template
	assert.Equal(
		t,
		tree,
		doc.treeAt(parse.Pos(len(src)-1)),
		"offset after all defines should be root",
	)
}

func TestDocumentRootHintOnFirstLine(t *testing.T) {
	src := "{{- /*gotype: cg/model.Root*/ -}}\nhello {{ . }}\n"
	tree, treeSet, err := tryParse(src)
	assert.NoError(t, err)
	assert.NotNil(t, tree)

	hints := types.FindTreeHints(src, treeSet)
	assert.Equal(t, "cg/model.Root", hints[tree.Name].Type)
}

// TestTreeAtMultiDefinesWithRoot verifies the treeAt logic for the canonical
// multiDefinesTemplate which has a root type hint and content outside the
// define blocks.  Positions inside a {{define}} must return that define's tree;
// positions in the root (before/between/after defines) must return the root tree.
func TestTreeAtMultiDefinesWithRoot(t *testing.T) {
	src := multiDefinesTemplate
	tree, treeSet, err := tryParse(src)
	assert.NoError(t, err)
	assert.NotNil(t, tree)

	doc := &document{text: src, tree: tree, trees: treeSet}

	// --- positions inside define bodies -> correct define tree ---

	offCustomerName := strings.Index(src, "CustomerName")
	assert.Equal(t, treeSet["OrderTpl"], doc.treeAt(parse.Pos(offCustomerName)),
		"offset inside OrderTpl body (CustomerName) should return OrderTpl tree")

	offStreet := strings.Index(src, ".Street")
	assert.Equal(t, treeSet["AddressTpl"], doc.treeAt(parse.Pos(offStreet)),
		"offset inside AddressTpl body (.Street) should return AddressTpl tree")

	offLocal := strings.Index(src, "$local := .")
	assert.Equal(t, treeSet["NoHint"], doc.treeAt(parse.Pos(offLocal)),
		"offset inside NoHint body ($local) should return NoHint tree")

	// --- positions in root template -> root tree ---

	// First root content line: {{ .Country }} (occurrence 0, before OrderTpl)
	offCountry0 := strings.Index(src, ".Country")
	assert.Equal(t, tree, doc.treeAt(parse.Pos(offCountry0)),
		"offset at first .Country in root (before defines) should return root tree")

	// Root content between OrderTpl and AddressTpl: {{ .Zip }}
	offZip := strings.Index(src, ".Zip")
	assert.Equal(t, tree, doc.treeAt(parse.Pos(offZip)),
		"offset at .Zip in root (between defines) should return root tree")

	// Root content after all defines: {{ .Country }} (occurrence 1)
	offCountry1 := strings.LastIndex(src, ".Country")
	assert.NotEqual(t, offCountry0, offCountry1, "sanity: two distinct .Country occurrences")
	assert.Equal(t, tree, doc.treeAt(parse.Pos(offCountry1)),
		"offset at last .Country in root (after all defines) should return root tree")

	// Root type hint on first line also lands in root tree
	assert.Equal(t, tree, doc.treeAt(parse.Pos(0)),
		"offset 0 (root hint line) should return root tree")
}

func TestDidOpenAndChange(t *testing.T) {
	// ai
	t.Run("DidOpen and DidChange", func(t *testing.T) {
		originalConfig := GetConfig()
		applyConfig(
			Config{
				EnableHover:          true,
				EnableDefinition:     true,
				EnableDiagnostics:    true,
				EnableAutocompletion: true,
			},
		)
		defer applyConfig(originalConfig)

		uri := "file:///test-open-change.txt"
		initialContent := "Initial Content"
		updatedContent := "Updated Content"

		// Test DidOpen
		openParams := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:  uri,
				Text: initialContent,
			},
		}

		err := DidOpen(nil, openParams)
		assert.NoError(t, err, "didOpen should not error")

		val, ok := store.Get(uri)
		assert.True(t, ok, "Document should exist after didOpen")
		assert.Equal(t, initialContent, val.text, "Content should match after didOpen")

		// Test DidChange
		changeParams := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
				Version:                2,
			},
			ContentChanges: []any{
				protocol.TextDocumentContentChangeEventWhole{
					Text: updatedContent,
				},
			},
		}

		err = DidChange(nil, changeParams)
		assert.NoError(t, err, "didChange should not error")

		val, ok = store.Get(uri)
		assert.True(t, ok, "Document should still exist after didChange")
		assert.Equal(t, updatedContent, val.text, "Content should be updated after didChange")

		// Cleanup
		store.Delete(uri)
	})

	// ai
	t.Run("DidClose", func(t *testing.T) {
		originalConfig := GetConfig()
		applyConfig(
			Config{
				EnableHover:          true,
				EnableDefinition:     true,
				EnableDiagnostics:    true,
				EnableAutocompletion: true,
			},
		)
		defer applyConfig(originalConfig)

		uri := "file:///test-open-change.txt"
		initialContent := "Initial Content"

		openParams := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:  uri,
				Text: initialContent,
			},
		}

		err := DidOpen(nil, openParams)
		assert.NoError(t, err, "didOpen should not error")

		val, ok := store.Get(uri)

		assert.True(t, ok, "Document should exist after Open")
		assert.Equal(t, initialContent, val.text, "Content should match after Open")

		closeParams := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		}

		err = DidClose(nil, closeParams)

		assert.NoError(t, err, "didClose should not error")
		val, ok = store.Get(uri)
		assert.False(t, ok, "Document should not exist after Close")
		assert.Nil(t, val, "Value should be nil after Close")
	})
}
