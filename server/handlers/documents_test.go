package handlers

import (
	"sync"
	"testing"

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

	t.Run("Parse Errors", func(t *testing.T) {
		invalidContent := "{{ unclosed"
		ds.Set(uri, invalidContent)
		val, ok := ds.Get(uri)

		assert.True(t, ok)
		assert.Equal(t, invalidContent, val.text, "Content should be stored even if parsing fails")
		assert.Nil(t, val.tree, "Tree should be nil if parsing fails")
	})

	t.Run("Parse Valid Content", func(t *testing.T) {
		validContent := "{{ . }}"
		ds.Set(uri, validContent)
		val, ok := ds.Get(uri)

		assert.True(t, ok)
		assert.Equal(t, validContent, val.text)
		assert.NotNil(t, val.tree, "Tree should be set for valid content")
	})

	t.Run("Remove Nonexistent Document", func(t *testing.T) {
		nonexistentURI := "file:///nonexistent.txt"
		ds.Remove(nonexistentURI) // Should not panic
		_, ok := ds.Get(nonexistentURI)
		assert.False(t, ok, "Document should not exist in the store")
	})

	t.Run("Empty Content", func(t *testing.T) {
		emptyContent := ""
		ds.Set(uri, emptyContent)
		val, ok := ds.Get(uri)

		assert.True(t, ok)
		assert.Equal(t, emptyContent, val.text, "Content should be stored even if it's empty")
		assert.NotNil(t, val.tree, "Tree should be set for empty content (valid template)")
	})

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

	t.Run("Invalid URI", func(t *testing.T) {
		invalidURI := "not-a-valid-uri"
		ds.Set(invalidURI, "Some content")
		val, ok := ds.Get(invalidURI)

		assert.True(t, ok, "Even invalid URIs should be stored")
		assert.Equal(t, "Some content", val.text)
	})

	t.Run("Delete", func(t *testing.T) {
		uriToDelete := "file:///to-delete.txt"
		ds.Set(uriToDelete, "Content to delete")
		ds.Delete(uriToDelete)
		_, ok := ds.Get(uriToDelete)
		assert.False(t, ok, "Document should be deleted from the store")
	})
}

func TestDidOpenAndChange(t *testing.T) {
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

		err := didOpen(nil, openParams)
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

		err = didChange(nil, changeParams)
		assert.NoError(t, err, "didChange should not error")

		val, ok = store.Get(uri)
		assert.True(t, ok, "Document should still exist after didChange")
		assert.Equal(t, updatedContent, val.text, "Content should be updated after didChange")

		// Cleanup
		store.Delete(uri)
	})

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

		err := didOpen(nil, openParams)
		assert.NoError(t, err, "didOpen should not error")

		val, ok := store.Get(uri)

		assert.True(t, ok, "Document should exist after Open")
		assert.Equal(t, initialContent, val.text, "Content should match after Open")

		closeParams := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		}

		err = didClose(nil, closeParams)

		assert.NoError(t, err, "didClose should not error")
		val, ok = store.Get(uri)
		assert.False(t, ok, "Document should not exist after Close")
		assert.Nil(t, val, "Value should be nil after Close")
	})
}
