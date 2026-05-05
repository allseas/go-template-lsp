package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentStore(t *testing.T) {
	// FIX: docs must be map[string]*document, not map[string]string
	ds := &documentStore{docs: make(map[string]*document)}
	uri := "file:///test-document.txt"
	content := "Initial Content"

	t.Run("Set and Get", func(t *testing.T) {
		ds.Set(uri, content)
		val, ok := ds.Get(uri)

		assert.True(t, ok, "Document should exist in the store")
		// FIX: val is a *document, so check val.text
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
}
