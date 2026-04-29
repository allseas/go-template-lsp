package main

import (
	"sync"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]string
}

var store = &DocumentStore{
	docs: make(map[string]string),
}

func (ds *DocumentStore) Set(uri, text string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.docs[uri] = text
}

func (ds *DocumentStore) Get(uri string) (string, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	text, ok := ds.docs[uri]
	return text, ok
}

func (ds *DocumentStore) Remove(uri string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.docs, uri)
}

func didOpen(_ *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	store.Set(params.TextDocument.URI, params.TextDocument.Text)
	return nil
}

func didChange(_ *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			store.Set(params.TextDocument.URI, c.Text)
		}
	}
	return nil
}

func didClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	store.Remove(params.TextDocument.URI)
	return nil
}
