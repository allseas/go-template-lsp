package main

import (
	"testing"
)

func TestDocumentStore(t *testing.T) {
	ds := &DocumentStore{docs: make(map[string]string)}
	uri := "file:///test-document.txt"
	content := "Initial Content"

	t.Run("Set and Get", func(t *testing.T) {
		ds.Set(uri, content)
		val, ok := ds.Get(uri)
		if !ok {
			t.Fatal("Expected to find document in store")
		}
		if val != content {
			t.Errorf("Expected %q, got %q", content, val)
		}
	})

	t.Run("Overwrite Content", func(t *testing.T) {
		newContent := "Updated Content"
		ds.Set(uri, newContent)
		val, _ := ds.Get(uri)
		if val != newContent {
			t.Errorf("Expected %q after update, got %q", newContent, val)
		}
	})

	t.Run("Remove Document", func(t *testing.T) {
		ds.Remove(uri)
		_, ok := ds.Get(uri)
		if ok {
			t.Error("Document should have been deleted from store")
		}
	})
}
