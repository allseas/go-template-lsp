package handlers

import (
	"strings"
	"sync"
	"testing"

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

func TestDocumentMultipleDefines(t *testing.T) {
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
	rootHint := hintTypeForTree(src, tree, true)
	hintA := hintTypeForTree(src, treeSet["A"], false)
	hintB := hintTypeForTree(src, treeSet["B"], false)
	assert.Equal(t, "", rootHint, "no hint expected on first line of file")
	assert.Equal(t, "cg/model.A", hintA)
	assert.Equal(t, "cg/model.B", hintB)

	// treeAt selects the right tree for an offset inside each define
	doc := &document{text: src, tree: tree, trees: treeSet}
	offA := strings.Index(src, "alpha-body")
	offB := strings.Index(src, "beta-body")
	assert.Equal(t, treeSet["A"], doc.treeAt(parse.Pos(offA)))
	assert.Equal(t, treeSet["B"], doc.treeAt(parse.Pos(offB)))
}

func TestDocumentRootHintOnFirstLine(t *testing.T) {
	src := "{{- /*gotype: cg/model.Root*/ -}}\nhello {{ . }}\n"
	tree, treeSet, err := tryParse(src)
	assert.NoError(t, err)
	assert.NotNil(t, tree)
	hint := hintTypeForTree(src, tree, true)
	assert.Equal(t, "cg/model.Root", hint)
	_ = treeSet
}

func TestDidOpenAndChange(t *testing.T) {
	// ai
	t.Run("DidOpen and DidChange", func(t *testing.T) {
		originalConfig := GetConfig()
		applyConfig(Config{EnableServer: true})
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
		applyConfig(Config{EnableServer: true})
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
