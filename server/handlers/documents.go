// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"path/filepath"
	"sync"
	parse "text-template-parser"
	"text-template-server/types"

	gotypes "go/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	lspuri "go.lsp.dev/uri"
)

// WorkspaceRoot is the path to the workspace root
var WorkspaceRoot string

type document struct {
	text string
	tree *parse.Tree
	// trees holds every parse tree produced for this document, keyed by tree name.
	// Includes the root tree under its own name and one entry per {{define}} block.
	trees map[string]*parse.Tree
	// deprecated, typedTree is enough, but functions should be rewritten
	loadedType *types.Tree
	typedTree  *types.Tree
	// loadedTypes is the per-tree dot-type resolved from the gotype hint that is located directly under each tree's {{define}}
	loadedTypes map[string]*types.Tree
	// typedTrees is the per-tree analysed (typed) tree, paired with loadedTypes.
	typedTrees map[string]*types.Tree
	// failedHints maps tree name to the hint that failed to load and the
	// resulting error message.
	failedHints map[string]failedHint
}

// failedHint pairs a gotype hint with the error produced when loading it.
type failedHint struct {
	Hint types.TypeHint
	Err  string
}

type documentStore struct {
	mu                 sync.RWMutex
	docs               map[string]*document
	templateInputTypes map[string]gotypes.Type // template name -> expected input type (from gotype hints on {{define}} blocks)
	uriTemplateNames   map[string][]string     // URI -> template names it contributes (for cleanup on close/update)
}

var store = &documentStore{
	docs:               make(map[string]*document),
	templateInputTypes: make(map[string]gotypes.Type),
	uriTemplateNames:   make(map[string][]string),
}

func (s *documentStore) Set(uri, text string) {
	tree, treeSet, err := parseTemplate(uri, text)

	treeHints := types.FindTreeHints(text, treeSet)
	// Try the LSP workspace root first (legacy behaviour), then fall back to
	// the directory containing the .tmpl file itself. This means a user can
	// open a parent folder as the workspace and still get type resolution as
	// long as the file lives inside a Go module.
	loadDirs := []string{WorkspaceRoot, uriDir(uri)}

	loadedTypes := make(map[string]*types.Tree)
	failedHints := make(map[string]failedHint)
	for name := range treeSet {
		hint := treeHints[name]
		log.Debug().Str("name", name).Str("hint", hint.Text).Msg("found gotype hint")
		if hint.IsMalformed() {
			failedHints[name] = failedHint{Hint: hint, Err: "malformed map hint"}
			continue
		}
		if hint.Text == "" {
			continue
		}
		var lastErr error
		for _, dir := range loadDirs {
			if dir == "" {
				continue
			}
			loaded, lerr := types.CachedLoadHint(hint, dir)
			if lerr != nil {
				lastErr = lerr
				continue
			}
			loadedTypes[name] = loaded
			lastErr = nil
			break
		}
		if lastErr != nil {
			failedHints[name] = failedHint{Hint: hint, Err: lastErr.Error()}
		}
	}

	// Load gotype hints from {{define}} blocks before taking the lock (type loading can be slow).
	newTemplateTypes := make(map[string]gotypes.Type)

	if WorkspaceRoot != "" {
		for name, hint := range treeHints {
			if name == "" {
				continue
			}
			if hint.IsMalformed() {
				continue
			}

			loaded, err := types.CachedLoadHint(hint, WorkspaceRoot)
			if err != nil {
				log.Warn().
					Str("template", name).
					Str("hint", hint.Text).
					Err(err).
					Msg("type hint load failed")
				continue
			}
			switch {
			case loaded.DictType != nil:
				newTemplateTypes[name] = loaded.DictType
			case loaded.DotType != nil:
				newTemplateTypes[name] = loaded.DotType
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		if existing, ok := s.docs[uri]; ok {
			tree = existing.tree
			treeSet = existing.trees
			if existing.loadedTypes != nil {
				loadedTypes = existing.loadedTypes
			}
		}
	}

	// Update the template input type registry first so that same-file
	// {{template}} calls can be type-checked against defines in this document.
	if s.uriTemplateNames == nil {
		s.uriTemplateNames = make(map[string][]string)
	}
	if s.templateInputTypes == nil {
		s.templateInputTypes = make(map[string]gotypes.Type)
	}
	if oldNames, ok := s.uriTemplateNames[uri]; ok {
		for _, name := range oldNames {
			delete(s.templateInputTypes, name)
		}
	}
	if len(newTemplateTypes) > 0 {
		newNames := make([]string, 0, len(newTemplateTypes))
		for name, typ := range newTemplateTypes {
			s.templateInputTypes[name] = typ
			newNames = append(newNames, name)
		}
		s.uriTemplateNames[uri] = newNames
	} else {
		delete(s.uriTemplateNames, uri)
	}

	typedTrees := make(map[string]*types.Tree, len(treeSet))
	for name, tr := range treeSet {
		typedTrees[name] = buildTypedTree(tr, loadedTypes[name], s.templateInputTypes)
	}
	var typed *types.Tree
	if tree != nil {
		typed = typedTrees[tree.Name]
	}

	doc := &document{
		text:        text,
		tree:        tree,
		trees:       treeSet,
		typedTree:   typed,
		loadedTypes: loadedTypes,
		typedTrees:  typedTrees,
		failedHints: failedHints,
	}
	for _, tt := range doc.typedTrees {
		if tt != nil {
			types.SetEndsForTree(*tt, types.Pos(len(doc.text)), &doc.text)
		}
	}
	s.docs[uri] = doc
}

// buildTypedTree returns the analysed (typed) tree if the parse tree exists.
// It carries dot-type / package info from the loaded type hint when available.
// templateInputTypes maps template names to their expected input types; it is
// consulted when analysing {{template "name" arg}} call sites.
func buildTypedTree(
	tree *parse.Tree,
	lt *types.Tree,
	templateInputTypes map[string]gotypes.Type,
) *types.Tree {
	if tree == nil {
		return nil
	}
	var dotType gotypes.Type
	var pkg *gotypes.Package
	if lt != nil {
		switch {
		case lt.DictType != nil:
			dotType = lt.DictType
		case lt.DotType != nil:
			dotType = lt.DotType
		}
		pkg = lt.Pkg
	}
	t := types.NewTree(*tree, types.GlobalFuncs(), dotType, pkg, templateInputTypes)
	if lt != nil {
		t.DotType = lt.DotType
		t.DictType = lt.DictType
		t.Pkg = lt.Pkg
		t.Fset = lt.Fset
	}
	return &t
}

// treeAt returns the tightest parse tree that contains offset.
func (d *document) treeAt(offset parse.Pos) *parse.Tree {
	if d == nil {
		return nil
	}
	var best *parse.Tree
	var bestSpan parse.Pos
	for _, t := range d.trees {
		if t == nil || t.Root == nil {
			continue
		}
		start := t.Root.Position()
		end := t.End
		if start > offset || offset >= end {
			continue
		}
		span := end - start
		if best == nil || span < bestSpan {
			best = t
			bestSpan = span
		}
	}
	if best != nil {
		return best
	}
	return d.tree
}

// loadedTypeAt returns the loaded type for the tree that covers offset.
// Falls back to d.loadedType for legacy callers that did not populate loadedTypes.
//
// Deprecated: prefer typedTreeAt — the typed tree carries DotType / Pkg / Fset
// just like the raw loaded type, and is the new path for type-aware features.
func (d *document) loadedTypeAt(offset parse.Pos) *types.Tree {
	if d == nil {
		return nil
	}
	if d.loadedTypes != nil {
		if tr := d.treeAt(offset); tr != nil {
			if lt, ok := d.loadedTypes[tr.Name]; ok {
				return lt
			}
		}
	}
	return d.loadedType
}

// typedTreeAt returns the typed tree for the tree that covers offset.
// Falls back to d.typedTree for legacy callers.
func (d *document) typedTreeAt(offset parse.Pos) *types.Tree {
	if d == nil {
		return nil
	}
	if d.typedTrees != nil {
		if tr := d.treeAt(offset); tr != nil {
			if tt, ok := d.typedTrees[tr.Name]; ok {
				return tt
			}
		}
	}
	return d.typedTree
}

// typedTreeAtTyped returns the typed tree for the tree that covers offset.
// It is the parse-free entry point that other handlers should use; documents.go
// remains the only file that needs to bridge to the parse package.
func (d *document) typedTreeAtTyped(offset types.Pos) *types.Tree {
	return d.typedTreeAt(parse.Pos(offset))
}

// loadedTypeAtTyped is the parse-free counterpart of loadedTypeAt.
//
// Deprecated: prefer typedTreeAtTyped where possible.
func (d *document) loadedTypeAtTyped(offset types.Pos) *types.Tree {
	return d.loadedTypeAt(parse.Pos(offset))
}

// nodeRange converts a typed node into an LSP Range using its start position
// and rendered length.
func nodeRange(n types.Node, text string) protocol.Range {
	start := int(n.Position())
	length := len(n.String())
	end := start + length

	return protocol.Range{
		Start: offsetToPosition(text, start),
		End:   offsetToPosition(text, end),
	}
}

// FindVarDeclarationsTyped returns all variable declaration nodes for a given
// variable name found anywhere within the typed subtree rooted at root.
func FindVarDeclarationsTyped(root types.Node, varName string) []*types.VariableNode {
	var decls []*types.VariableNode

	types.Inspect(root, func(n types.Node) bool {
		pipe, ok := n.(*types.PipeNode)
		if !ok {
			return true
		}
		for _, decl := range pipe.Decl {
			if len(decl.Ident) > 0 && decl.Ident[0] == varName {
				decls = append(decls, decl)
			}
		}
		return true
	})
	return decls
}

// IsIndexVariableTyped reports whether target refers to the index variable
// (the first declared variable) of any enclosing range loop. It walks the
// typed node's parent chain to find an enclosing RangeNode whose first Decl
// matches target by identity or by name.
func IsIndexVariableTyped(target *types.VariableNode) bool {
	if target == nil || len(target.Ident) == 0 {
		return false
	}
	name := target.Ident[0]
	for cur := types.Node(target).Parent(); cur != nil; cur = cur.Parent() {
		rn, ok := cur.(*types.RangeNode)
		if !ok {
			continue
		}
		if rn.Pipe == nil || len(rn.Pipe.Decl) == 0 {
			continue
		}
		first := rn.Pipe.Decl[0]
		if first == nil || len(first.Ident) == 0 {
			continue
		}
		if first == target || first.Ident[0] == name {
			return true
		}
	}
	return false
}

func (s *documentStore) Get(uri string) (*document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[uri]
	return d, ok
}

// Delete removes a document from the store. It is an alias for Remove, kept
// for callers that prefer the map-style name.
func (s *documentStore) Delete(uri string) {
	s.Remove(uri)
}

func parseTemplate(uri, text string) (*parse.Tree, map[string]*parse.Tree, error) {
	tree, treeSet, err := tryParse(text)
	if err != nil {
		log.Debug().Str("uri", uri).Err(err).Msg("full parse failed, tree is not updated")
	}
	return tree, treeSet, err
}

func tryParse(text string) (*parse.Tree, map[string]*parse.Tree, error) {
	t := parse.New("t")
	t.Mode = parse.ParsePartial | parse.SkipFuncCheck | parse.ParseComments
	treeSet := map[string]*parse.Tree{}
	_, err := t.Parse(text, "{{", "}}", treeSet)
	if err != nil {
		return nil, nil, err
	}
	return t, treeSet, nil
}

// Remove deletes a document from the store, typically called when a file is closed in the editor.
func (s *documentStore) Remove(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if oldNames, ok := s.uriTemplateNames[uri]; ok {
		for _, name := range oldNames {
			delete(s.templateInputTypes, name)
		}
		delete(s.uriTemplateNames, uri)
	}
	delete(s.docs, uri)
}

// snapshot returns the current (uri, text) pairs in a stable order so callers
// can re-process every open document without holding the store lock.
func (s *documentStore) snapshot() []struct{ URI, Text string } {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]struct{ URI, Text string }, 0, len(s.docs))
	for uri, d := range s.docs {
		out = append(out, struct{ URI, Text string }{URI: uri, Text: d.text})
	}
	return out
}

// setDocument stores text for uri and, when ctx is non-nil, re-publishes
// diagnostics for it. It centralises the parse-and-publish step shared by the
// open/change/refresh notification handlers.
func setDocument(ctx *glsp.Context, uri, text string) {
	store.Set(uri, text)
	if ctx != nil {
		publishDiagnostics(ctx, uri, text)
	}
}

// RefreshAllDocuments re-runs the document store pipeline (parse + typed tree)
// for every open document and re-publishes diagnostics. Use this after the
// workspace's global function map or other tree-wide inputs have changed.
func RefreshAllDocuments(ctx *glsp.Context) {
	for _, d := range store.snapshot() {
		setDocument(ctx, d.URI, d.Text)
	}
}

// DidOpen is an LSP notification handler that registers a new document in the store when it is opened.
func DidOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	setDocument(ctx, params.TextDocument.URI, params.TextDocument.Text)
	return nil
}

// DidChange is an LSP notification handler that updates the stored document content when the user edits the file.
func DidChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	log.Debug().
		Str("uri", params.TextDocument.URI).
		Msg("document changed")

	for _, change := range params.ContentChanges {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEventWhole:
			setDocument(ctx, params.TextDocument.URI, c.Text)
		case protocol.TextDocumentContentChangeEvent:
			setDocument(ctx, params.TextDocument.URI, c.Text)
		}
	}
	return nil
}

// DidClose is an LSP notification handler that removes a document from the store when the editor closes the file.
func DidClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	store.Remove(params.TextDocument.URI)
	return nil
}

// uriDir returns the local filesystem directory that contains the file
// referenced by uri, or "" if the uri is not a parseable local file URI.
// Used as a fallback root when resolving gotype hints, so type resolution
// works even when the LSP workspace root is an ancestor of the file's
// Go module instead of the module root itself.
func uriDir(uri string) string {
	u, err := lspuri.Parse(uri)
	if err != nil {
		return ""
	}
	path := u.Filename()
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}
