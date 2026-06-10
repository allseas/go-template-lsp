// Package handlers provides a document synchronization and completion engine for Go text/templates.
package handlers

import (
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
	// deprecated, typedTree is enough, but functions should be rewritten
	loadedType *types.Tree
	typedTree  *types.Tree
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

	var lt *types.Tree
	if WorkspaceRoot != "" {
		hints := types.ParseTypeHints(strings.NewReader(text))
		if len(hints) > 0 {
			if loaded, lerr := types.LoadTypeFromHint(hints[0].Type, WorkspaceRoot); lerr == nil {
				lt = loaded
			} else {
				log.Warn().Str("hint", hints[0].Type).Err(lerr).Msg("type hint load failed")
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

	s.docs[uri] = &document{
		text:       text,
		tree:       tree,
		loadedType: lt,
		typedTree:  buildTypedTree(tree, lt),
	}
}

// buildTypedTree returns the analysed (typed) tree if the parse tree exists.
// It carries dot-type / package info from the loaded type hint when available.
func buildTypedTree(tree *parse.Tree, lt *types.Tree) *types.Tree {
	if tree == nil {
		return nil
	}
	var dotType gotypes.Type
	var pkg *gotypes.Package
	if lt != nil {
		dotType = lt.DotType
		pkg = lt.Pkg
	}
	t := types.NewTree(*tree, types.GlobalFuncs(), dotType, pkg)
	if lt != nil {
		t.DotType = lt.DotType
		t.Pkg = lt.Pkg
	}
	return &t
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
	t.Mode = parse.ParsePartial | parse.SkipFuncCheck | parse.ParseComments
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
