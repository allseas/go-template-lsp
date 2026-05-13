// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"strings"
	"sync"
	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type document struct {
	text       string
	tree       *parse.Tree
	loadedType *LoadedType
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

	var lt *LoadedType
	if workspaceRoot != "" {
		hints := ParseTypeHints(strings.NewReader(text))
		if len(hints) > 0 {
			if loaded, lerr := LoadTypeFromHint(hints[0].Type, workspaceRoot); lerr == nil {
				lt = loaded
			} else {
				log.Debug().Str("hint", hints[0].Type).Err(lerr).Msg("type hint load failed")
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		if existing, ok := s.docs[uri]; ok {
			tree = existing.tree
		}
	}

	s.docs[uri] = &document{text: text, tree: tree, loadedType: lt}
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
	t.Mode = parse.ParsePartial | parse.SkipFuncCheck
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
func didOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	if !GetConfig().EnableServer {
		log.Debug().Msg("didOpen received but server is disabled by config")
		return nil
	}

	store.Set(params.TextDocument.URI, params.TextDocument.Text)
	if ctx != nil {
		publishDiagnostics(ctx, params.TextDocument.URI, params.TextDocument.Text)
	}
	return nil
}

// didChange is an LSP notification handler that updates the stored document content when the user edits the file.
func didChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	log.Debug().
		Str("uri", params.TextDocument.URI).
		Msg("document changed")

	if !GetConfig().EnableServer {
		log.Debug().Msg("didOpen received but server is disabled by config")
		return nil
	}

	for _, change := range params.ContentChanges {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEventWhole:
			store.Set(params.TextDocument.URI, c.Text)
			if ctx != nil {
				publishDiagnostics(ctx, params.TextDocument.URI, c.Text)
			}
		case protocol.TextDocumentContentChangeEvent:
			store.Set(params.TextDocument.URI, c.Text)
			if ctx != nil {
				publishDiagnostics(ctx, params.TextDocument.URI, c.Text)
			}
		}
	}
	return nil
}

// didClose is an LSP notification handler that removes a document from the store when the editor closes the file.
func didClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	store.Remove(params.TextDocument.URI)
	return nil
}

// nodeFind finds a node in a tree given the offset
func nodeFind(root parse.Node, offset parse.Pos) parse.Node {
	best := root
	bestPos := parse.Pos(0)

	var walk func(n parse.Node)
	walk = func(n parse.Node) {
		if n == nil {
			return
		}

		pos := n.Position()
		if pos <= offset && pos >= bestPos {
			bestPos = pos
			best = n
		}

		switch node := n.(type) {
		case *parse.ListNode:
			for _, child := range node.Nodes {
				walk(child)
			}
		case *parse.ActionNode:
			walk(node.Pipe)
		case *parse.PipeNode:
			for _, v := range node.Decl {
				walk(v)
			}
			for _, cmd := range node.Cmds {
				walk(cmd)
			}
		case *parse.CommandNode:
			for _, arg := range node.Args {
				walk(arg)
			}
		case *parse.ChainNode:
			walk(node.Node)
		case *parse.IfNode:
			walk(node.Pipe)
			walk(node.List)
			if node.ElseList != nil {
				walk(node.ElseList)
			}
		case *parse.RangeNode:
			walk(node.Pipe)
			walk(node.List)
			if node.ElseList != nil {
				walk(node.ElseList)
			}
		case *parse.WithNode:
			walk(node.Pipe)
			walk(node.List)
			if node.ElseList != nil {
				walk(node.ElseList)
			}
		case *parse.TemplateNode:
			walk(node.Pipe)
		case *parse.UndefinedNode:
			log.Debug().Msg("found the undefined node")
		}
	}

	walk(root)
	return best
}
