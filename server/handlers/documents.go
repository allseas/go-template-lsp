// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
	"regexp"
	"strings"
	"sync"
	parse "text-template-parser"
	"text-template-server/types"
	"text-template-server/utils"

	gotypes "go/types"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
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
}

type documentStore struct {
	mu                 sync.RWMutex
	docs               map[string]*document
	templateInputTypes map[string]gotypes.Type // template name → expected input type (from gotype hints on {{define}} blocks)
	uriTemplateNames   map[string][]string     // URI → template names it contributes (for cleanup on close/update)
}

var store = &documentStore{
	docs:               make(map[string]*document),
	templateInputTypes: make(map[string]gotypes.Type),
	uriTemplateNames:   make(map[string][]string),
}

func (s *documentStore) Set(uri, text string) {
	tree, treeSet, err := parseTemplate(uri, text)

	loadedTypes := make(map[string]*types.Tree)
	if WorkspaceRoot != "" {
		for name, tr := range treeSet {
			isRoot := tree != nil && tr == tree
			hint := hintTypeForTree(text, tr, isRoot)
			if hint == "" {
				continue
			}
			loaded, lerr := types.LoadTypeFromHint(hint, WorkspaceRoot)
			if lerr != nil {
				log.Warn().Str("hint", hint).Err(lerr).Msg("type hint load failed")
				continue
			}
			loadedTypes[name] = loaded
		}
	}

	var lt *types.Tree
	if tree != nil {
		lt = loadedTypes[tree.Name]
	}

	// Load gotype hints from {{define}} blocks before taking the lock (type loading can be slow).
	var newTemplateTypes map[string]gotypes.Type
	if WorkspaceRoot != "" {
		defineHints := types.ParseDefineTypeHints(text)
		if len(defineHints) > 0 {
			newTemplateTypes = make(map[string]gotypes.Type, len(defineHints))
			for name, hint := range defineHints {
				if loaded, lerr := types.LoadTypeFromHint(hint, WorkspaceRoot); lerr == nil {
					newTemplateTypes[name] = loaded.DotType
				} else {
					log.Warn().Str("template", name).Str("hint", hint).Err(lerr).Msg("define block type hint load failed")
				}
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
			if lt == nil {
				lt = existing.loadedType
			}
		}
	}

	typedTrees := make(map[string]*types.Tree, len(treeSet))
	for name, tr := range treeSet {
		typedTrees[name] = buildTypedTree(tr, loadedTypes[name], s.templateInputTypes)
	}
	var typed *types.Tree
	if tree != nil {
		typed = typedTrees[tree.Name]
	}

	// Update the template input type registry: remove stale entries for this URI,
	// then add the newly discovered ones.
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

	// Update the template input type registry: remove stale entries for this URI,
	// then add the newly discovered ones.
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

	s.docs[uri] = &document{
		text:        text,
		tree:        tree,
		trees:       treeSet,
		loadedType:  lt,
		typedTree:   typed,
		loadedTypes: loadedTypes,
		typedTrees:  typedTrees,
	}
}

// buildTypedTree returns the analysed (typed) tree if the parse tree exists.
// It carries dot-type / package info from the loaded type hint when available.
// templateInputTypes maps template names to their expected input types; it is
// consulted when analysing {{template "name" arg}} call sites.
func buildTypedTree(tree *parse.Tree, lt *types.Tree, templateInputTypes map[string]gotypes.Type) *types.Tree {
	if tree == nil {
		return nil
	}
	var dotType gotypes.Type
	var pkg *gotypes.Package
	if lt != nil {
		dotType = lt.DotType
		pkg = lt.Pkg
	}
	t := types.NewTree(*tree, types.GlobalFuncs(), dotType, pkg, templateInputTypes)
	if lt != nil {
		t.DotType = lt.DotType
		t.Pkg = lt.Pkg
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

func (s *documentStore) Get(uri string) (*document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[uri]
	return d, ok
}

func (s *documentStore) Delete(uri string) {
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

func parseTemplate(uri, text string) (*parse.Tree, map[string]*parse.Tree, error) {
	tree, treeSet, err := tryParse(text)
	if err != nil {
		log.Debug().Str("uri", uri).Err(err).Msg("full parse failed, tree is not updated")
	}
	return tree, treeSet, err
}

func tryParse(text string) (*parse.Tree, map[string]*parse.Tree, error) {
	t := parse.New("t")
	t.Mode = parse.ParsePartial | parse.SkipFuncCheck
	treeSet := map[string]*parse.Tree{}
	_, err := t.Parse(text, "{{", "}}", treeSet)
	if err != nil {
		return nil, nil, err
	}
	return t, treeSet, nil
}

// hintTypeForTree returns the gotype hint string for the given tree.
func hintTypeForTree(text string, tree *parse.Tree, isRoot bool) string {
	if isRoot {
		return firstHintIn(firstLineOf(text))
	}
	if tree == nil {
		return ""
	}
	line := findDefineLine(text, tree.Name)
	if line <= 0 {
		return ""
	}
	return firstHintIn(getLine(text, line+1))
}

func firstHintIn(line string) string {
	if line == "" {
		return ""
	}
	hints := types.ParseTypeHints(strings.NewReader(line))
	if len(hints) == 0 {
		return ""
	}
	return hints[0].Type
}

var defineRe = regexp.MustCompile(`\{\{-?\s*define\s+"([^"]*)"\s*-?\}\}`)

// findDefineLine returns the 1-based line number of the {{define "name"}}
// directive in text, or -1 if not found.
func findDefineLine(text, name string) int {
	for _, m := range defineRe.FindAllStringSubmatchIndex(text, -1) {
		// m: [start, end, nameStart, nameEnd]
		if len(m) < 4 {
			continue
		}
		got := text[m[2]:m[3]]
		if got != name {
			continue
		}
		line := 1
		for i := 0; i < m[0]; i++ {
			if text[i] == '\n' {
				line++
			}
		}
		return line
	}
	return -1
}

// getLine returns the 1-based line of text, without its trailing newline.
func getLine(text string, line int) string {
	if line <= 0 {
		return ""
	}
	start := 0
	cur := 1
	for cur < line && start < len(text) {
		nl := strings.IndexByte(text[start:], '\n')
		if nl < 0 {
			return ""
		}
		start += nl + 1
		cur++
	}
	if start >= len(text) {
		return ""
	}
	end := strings.IndexByte(text[start:], '\n')
	if end < 0 {
		return text[start:]
	}
	return text[start : start+end]
}

func firstLineOf(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return text[:i]
	}
	return text
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

// RefreshAllDocuments re-runs the document store pipeline (parse + typed tree)
// for every open document and re-publishes diagnostics. Use this after the
// workspace's global function map or other tree-wide inputs have changed.
func RefreshAllDocuments(ctx *glsp.Context) {
	for _, d := range store.snapshot() {
		store.Set(d.URI, d.Text)
		if ctx != nil {
			publishDiagnostics(ctx, d.URI, d.Text)
		}
	}
}

// DidOpen is an LSP notification handler that registers a new document in the store when it is opened.
func DidOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
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

// DidChange is an LSP notification handler that updates the stored document content when the user edits the file.
func DidChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
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

// DidClose is an LSP notification handler that removes a document from the store when the editor closes the file.
func DidClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	store.Remove(params.TextDocument.URI)
	return nil
}

// nodeFind finds a node in a tree given the offset
// deprecated, NodeFind should now find the correct node in a type tree
func nodeFind(root parse.Node, offset parse.Pos) parse.Node {
	best := root
	bestPos := parse.Pos(0)

	var walk func(n parse.Node)
	walk = func(n parse.Node) {
		if utils.IsNilNode(n) {
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

// walkAndAnalyze recursively walks the node tree, maintaining scope context, and calls fn on every node.
func walkAndAnalyze(
	node parse.Node,
	text string,
	ctx *Context,
	visited map[parse.Node]bool,
	fn func(parse.Node, string, *Context) []protocol.Diagnostic,
) (diagnostics []protocol.Diagnostic) {
	if node == nil || visited[node] {
		return nil
	}
	if ctx == nil {
		ctx = &Context{Vars: make(map[string]parse.Node)}
	}
	visited[node] = true
	defer delete(visited, node)

	diagnostics = append(diagnostics, fn(node, text, ctx)...)

	switch n := node.(type) {
	case *parse.ListNode:
		for _, child := range n.Nodes {
			diagnostics = append(diagnostics, walkAndAnalyze(child, text, ctx, visited, fn)...)
		}
	case *parse.ActionNode:
		if n.Pipe != nil {
			diagnostics = append(diagnostics, walkAndAnalyze(n.Pipe, text, ctx, visited, fn)...)
		}
	case *parse.PipeNode:
		if ctx.Vars == nil {
			ctx.Vars = make(map[string]parse.Node)
		}
		for _, v := range n.Decl {
			if v != nil && len(v.Ident) > 0 {
				ctx.Vars[v.Ident[0]] = n
			}
		}
		prevPipe := ctx.Pipe
		ctx.Pipe = n
		for _, cmd := range n.Cmds {
			diagnostics = append(diagnostics, walkAndAnalyze(cmd, text, ctx, visited, fn)...)
		}
		ctx.Pipe = prevPipe
	case *parse.CommandNode:
		for _, arg := range n.Args {
			diagnostics = append(diagnostics, walkAndAnalyze(arg, text, ctx, visited, fn)...)
		}
	case *parse.RangeNode, *parse.IfNode, *parse.WithNode:
		pipe, list, elseList := extractBranchNodes(n)
		if ctx.Vars == nil {
			ctx.Vars = make(map[string]parse.Node)
		}
		snapshot := snapshotVars(ctx.Vars)
		if pipe != nil {
			diagnostics = append(diagnostics, walkAndAnalyze(pipe, text, ctx, visited, fn)...)
		}
		if list != nil {
			diagnostics = append(diagnostics, walkAndAnalyze(list, text, ctx, visited, fn)...)
		}
		ctx.Vars = snapshot
		if elseList != nil {
			diagnostics = append(diagnostics, walkAndAnalyze(elseList, text, ctx, visited, fn)...)
		}
		ctx.Vars = snapshot
	}

	return diagnostics
}

// FindVarDeclarations returns all variable declaration nodes for a given variable name in the tree.
func FindVarDeclarations(root parse.Node, varName string) []*parse.VariableNode {
	var decls []*parse.VariableNode

	inspect(root, func(n parse.Node) bool {
		// this goes over the tree and finds declarations (inside PipeNode) of varName
		pipe, ok := n.(*parse.PipeNode)
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

// IsIndexVariable determines if a variable node refers to the index variable in a range.
func IsIndexVariable(target *parse.VariableNode, root *parse.ListNode) bool {
	if target == nil || len(target.Ident) == 0 {
		return false
	}
	ctx := &Context{Vars: make(map[string]parse.Node)}
	buildPath(root, target, ctx)

	path := ctx.Path
	if len(path) < 2 {
		return WasDeclaredAsIndex(target, ctx)
	}
	branch := path[len(path)-2]
	if _, ok := branch.(*parse.RangeNode); !ok {
		return WasDeclaredAsIndex(target, ctx)
	}
	branchNode := branch.(*parse.RangeNode)

	pipe := branchNode.Pipe
	if len(pipe.Decl) == 0 {
		return false
	}
	return pipe.Decl[0] == target
}

// WasDeclaredAsIndex checks whether the provided variable node was declared as the index by scanning
// the context constructed by buildPath (used when the variable isn't directly the declared index in the immediate range).
func WasDeclaredAsIndex(target *parse.VariableNode, ctx *Context) bool {
	if target == nil || len(target.Ident) == 0 {
		return false
	}
	for ident, pipe := range ctx.Vars {
		if ident != target.Ident[0] {
			continue
		}
		pn, ok := pipe.(*parse.PipeNode)
		if !ok || len(pn.Decl) == 0 || len(pn.Decl[0].Ident) == 0 {
			return false
		}
		if pn.Decl[0].Ident[0] != target.Ident[0] {
			return false
		}
		for _, node := range ctx.Path {
			rn, ok := node.(*parse.RangeNode)
			if !ok {
				continue
			}
			if rn.Pipe != pipe {
				continue
			}
			return true
		}
	}
	return false
}

// ResolveVarInfo resolves a variable node to its value and Go type
func ResolveVarInfo(
	_ parse.Node,
	target *parse.VariableNode,
	_ *types.Tree,
) (value any, goType gotypes.Type) {
	if target == nil || len(target.Ident) == 0 {
		return nil, nil
	}

	// TODO: add actual functionality, using the type tree

	return nil, nil
}
