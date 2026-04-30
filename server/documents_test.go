package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentStore(t *testing.T) {
	ds := &DocumentStore{docs: make(map[string]string)}
	uri := "file:///test-document.txt"
	content := "Initial Content"

	t.Run("Set and Get", func(t *testing.T) {
		ds.Set(uri, content)
		val, ok := ds.Get(uri)

		assert.True(t, ok, "Document should exist in the store")
		assert.Equal(t, content, val)
	})

	t.Run("Overwrite Content", func(t *testing.T) {
		newContent := "Updated Content"
		ds.Set(uri, newContent)
		val, _ := ds.Get(uri)

		assert.Equal(t, newContent, val, "Content should match the updated value")
	})

	t.Run("Remove Document", func(t *testing.T) {
		ds.Remove(uri)
		_, ok := ds.Get(uri)

		assert.False(t, ok, "Document should no longer exist after removal")
	})
}
