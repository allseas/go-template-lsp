// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DocumentStore manages the state of open text documents,
// providing access to document content across concurrent LSP requests.
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]string
}

var store = &DocumentStore{
	docs: make(map[string]string),
}

// Set updates or creates the content for a given document URI.
func (ds *DocumentStore) Set(uri, text string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.docs[uri] = text
}

// Get retrieves the content of a document by its URI.
// It returns the content and a boolean indicating if the document exists
func (ds *DocumentStore) Get(uri string) (string, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	text, ok := ds.docs[uri]
	return text, ok
}

// Remove deletes a document from the store, typically called when a file is closed in the editor.
func (ds *DocumentStore) Remove(uri string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.docs, uri)
}

// didOpen is an LSP notification handler that registers a new document in the store when it is opened.
func didOpen(_ *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	if !GetConfig().EnableServer {
		log.Debug().Msg("didOpen received but server is disabled by config")
		return nil
	}

	store.Set(params.TextDocument.URI, params.TextDocument.Text)
	return nil
}

// didChange is an LSP notification handler that updates the stored document content when the user edits the file.
func didChange(_ *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	log.Info().
		Str("uri", params.TextDocument.URI).
		Any("config", GetConfig()).
		Msg("document changed")

	if !GetConfig().EnableServer {
		log.Debug().Msg("didOpen received but server is disabled by config")
		return nil
	}

	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			store.Set(params.TextDocument.URI, c.Text)
		}
	}
	return nil
}

// didClose is an LSP notification handler that removes a document from the store when the editor closes the file.
func didClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	store.Remove(params.TextDocument.URI)
	return nil
}
