// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"sync"
	"text/template/parse"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type document struct {
	text string
	tree *parse.Tree
}

type documentStore struct {
	mu   sync.RWMutex
	docs map[string]*document
}

var store = &documentStore{
	docs: make(map[string]*document),
}

func (s *documentStore) Set(uri, text string) {
	tree, err := parseTemplate(uri, text)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		if existing, ok := s.docs[uri]; ok {
			tree = existing.tree
		}
	}

	s.docs[uri] = &document{text: text, tree: tree}
}

func (s *documentStore) Get(uri string) (*document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[uri]
	return d, ok
}

func (s *documentStore) Delete(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

func parseTemplate(uri, text string) (*parse.Tree, error) {
	tree, err := tryParse(text)
	if err != nil {
		log.Debug().Str("uri", uri).Err(err).Msg("full parse failed, tree is not updated")
	}
	return tree, err
}

func tryParse(text string) (*parse.Tree, error) {
	t := parse.New("t")
	t.Mode = parse.SkipFuncCheck
	treeSet := map[string]*parse.Tree{}
	_, err := t.Parse(text, "{{", "}}", treeSet)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Remove deletes a document from the store, typically called when a file is closed in the editor.
func (s *documentStore) Remove(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
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
	log.Debug().
		Str("uri", params.TextDocument.URI).
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
