// Package types implements utils for types and the type tree
package types

import (
	"fmt"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"golang.org/x/tools/go/packages"
)

type typeHintType int

const (
	typeHintNone typeHintType = iota
	typeHintStruct
	typeHintDict
)

// TypeHint represents a `gotype:` type hint found in a template file.
type TypeHint struct {
	Type typeHintType
	// Text is the raw type reference that follows `gotype:` in the comment.
	// For struct hints this is the type path (e.g. "example.com/m.Order").
	// For dict hints this is the raw body between the braces of `dict{...}`.
	Text string
	// Dict is populated for dict hints; it maps each declared key to its type
	// reference (e.g. "Order" -> "example.com/m.Order"). Nil for struct hints.
	Dict map[string]string
	// Line is the 1-based line number in the source text at which the hint
	// appears; 0 when the hint is unset.
	Line int
}

func treeAt(offset int, trees map[string]*parse.Tree) *parse.Tree {
	var best *parse.Tree
	var bestSpan int
	for _, t := range trees {
		if t == nil || t.Root == nil {
			continue
		}
		start := int(t.Root.Position())
		end := int(t.End)
		if start > offset || offset >= end {
			continue
		}
		if span := end - start; best == nil || span < bestSpan {
			best, bestSpan = t, span
		}
	}
	if best != nil {
		return best
	}
	return trees["t"]
}

// FindTreeHints scans each parse tree for a `gotype:` comment and returns a
// map of template names to the first hint found in that tree.
func FindTreeHints(text string, trees map[string]*parse.Tree) map[string]TypeHint {
	result := make(map[string]TypeHint)

	re := regexp.MustCompile(`gotype:\s*([A-Za-z_][A-Za-z0-9_/.-]*)`)

	for name, tree := range trees {
		if tree == nil || tree.Root == nil {
			continue
		}
		var hint TypeHint
		inspectParsed(tree.Root, func(node parse.Node) {
			if hint.Type != typeHintNone {
				return
			}
			c, ok := node.(*parse.CommentNode)
			if !ok {
				return
			}
			m := re.FindStringSubmatch(c.Text)
			if len(m) < 2 {
				return
			}
			hint = TypeHint{
				Type: typeHintStruct,
				Text: m[1],
				Line: strings.Count(text[:int(c.Pos)], "\n") + 1,
			}
		})
		if hint.Type != typeHintNone {
			result[name] = hint
		}
	}

	return result
}

func inspectParsed(node parse.Node, f func(node parse.Node)) {
	if node == nil {
		return
	}
	f(node)
	for _, child := range parseNodeChildren(node) {
		inspectParsed(child, f)
	}
}

func parseNodeChildren(node parse.Node) []parse.Node {
	switch n := node.(type) {
	case *parse.ListNode:
		return n.Nodes
	case *parse.IfNode:
		return parseBranchChildren(n.List, n.ElseList)
	case *parse.RangeNode:
		return parseBranchChildren(n.List, n.ElseList)
	case *parse.WithNode:
		return parseBranchChildren(n.List, n.ElseList)
	case *parse.TemplateNode:
		return nil
	default:
		return extParseNodeChildren(node)
	}
}

func parseBranchChildren(list, elseList *parse.ListNode) []parse.Node {
	var out []parse.Node
	if list != nil {
		out = append(out, list)
	}
	if elseList != nil {
		out = append(out, elseList)
	}
	return out
}

type DictType struct {
	types map[string]types.Type
}

// TypeField is a resolved field from a struct type.
type TypeField struct {
	Name     string
	TypeName string
	Type     types.Type // actual type object
	Embedded bool
}

// MethodType is the struct for the functions in the model.
type MethodType struct {
	Func       *types.Func
	Name       string
	ReturnName string
	ReturnType types.Type
	Params     []ParamType
}

// ParamType is needed to extract parameter types of a function
type ParamType struct {
	Name     string
	Type     types.Type
	TypeName string
}

// goEnv returns the current process environment, augmenting PATH with the
// directory of the Go binary if it is not already resolvable. It also mutates
// the process PATH (os.Setenv) because golang.org/x/tools/go/packages calls
// exec.LookPath("go") against the *process* PATH (not cfg.Env) before
// invoking `go list`. This is needed when the server is spawned by a client
// (e.g. VS Code's test runner) that does not inherit the shell PATH where
// the Go toolchain lives.
func goEnv() []string {
	if _, err := exec.LookPath("go"); err == nil {
		return os.Environ()
	}
	// Fallback: check common well-known Go installation directories.
	candidates := []string{
		"/usr/local/go/bin",
		"/usr/lib/go/bin",
		"/usr/local/bin",
		"/usr/bin",
	}
	for _, dir := range candidates {
		if _, statErr := os.Stat(filepath.Join(dir, "go")); statErr == nil {
			newPATH := dir + string(os.PathListSeparator) + os.Getenv("PATH")
			_ = os.Setenv("PATH", newPATH)
			return os.Environ()
		}
	}
	return os.Environ()
}

var (
	typeHintCacheMu sync.RWMutex
	typeHintCache   = make(map[string]*Tree)
)

// InvalidateTypeHintCache clears the cached type-hint results
func InvalidateTypeHintCache() {
	typeHintCacheMu.Lock()
	defer typeHintCacheMu.Unlock()
	typeHintCache = make(map[string]*Tree)
}

// CachedLoadTypeFromHint is like LoadTypeFromHint but returns the previously
// computed result when the same (hint, workspaceRoot) pair has been resolved
// before and the cache has not been invalidated.
func CachedLoadTypeFromHint(hint, workspaceRoot string) (*Tree, error) {
	key := hint + "\x00" + workspaceRoot

	typeHintCacheMu.RLock()
	if t, ok := typeHintCache[key]; ok {
		typeHintCacheMu.RUnlock()
		return t, nil
	}
	typeHintCacheMu.RUnlock()

	t, err := LoadTypeFromHint(hint, workspaceRoot)
	if err != nil {
		return nil, err
	}

	typeHintCacheMu.Lock()
	typeHintCache[key] = t
	typeHintCacheMu.Unlock()

	return t, nil
}

// LoadTypeFromHint loads the Go package identified by the hint and returns a
// Tree with DotType and Pkg set.
func LoadTypeFromHint(hint, workspaceRoot string) (*Tree, error) {
	importPath, typeName := splitTypeHint(hint)

	log.Debug().
		Str("hint", hint).
		Str("importPath", importPath).
		Str("typeName", typeName).
		Str("workspaceRoot", workspaceRoot).
		Msg("LoadTypeFromHint: attempting to load type")

	// possibly add packages.NeedTypesInfo | packages.NeedImports |  packages.NeedName | packages.NeedFiles | packages.NeedSyntax later (some used in code_gen)
	dir := workspaceRoot
	if _, err := os.Stat(dir); err != nil {
		log.Warn().
			Str("dir", dir).
			Msg("LoadTypeFromHint: workspace root does not exist on disk, using process cwd")
		if cwd, cwdErr := os.Getwd(); cwdErr == nil {
			dir = cwd
		}
	}
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedTypes,
		Dir:  dir,
		Fset: fset,
		Env:  goEnv(),
	}

	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		log.Error().
			Err(err).
			Str("importPath", importPath).
			Str("dir", workspaceRoot).
			Msg("LoadTypeFromHint: packages.Load failed")
		return nil, fmt.Errorf("packages.Load(%q): %w", importPath, err)
	}
	if len(pkgs) == 0 {
		log.Error().Str("importPath", importPath).Msg("LoadTypeFromHint: no packages found")
		return nil, fmt.Errorf("no packages found for import path %q", importPath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		log.Error().
			Str("importPath", importPath).
			Str("error", pkg.Errors[0].Msg).
			Msg("LoadTypeFromHint: package has errors")
		return nil, fmt.Errorf("package %q has errors: %v", importPath, pkg.Errors[0])
	}

	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		log.Error().
			Str("typeName", typeName).
			Str("importPath", importPath).
			Msg("LoadTypeFromHint: type not found in package scope")
		return nil, fmt.Errorf("type %q not found in package %q", typeName, importPath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		log.Error().Str("typeName", typeName).Msg("LoadTypeFromHint: type is not a named type")
		return nil, fmt.Errorf("%q is not a named type in package %q", typeName, importPath)
	}

	log.Debug().
		Str("typeName", typeName).
		Str("importPath", importPath).
		Int("numFields", named.Underlying().(*types.Struct).NumFields()).
		Int("numMethods", named.NumMethods()).
		Msg("LoadTypeFromHint: type loaded successfully")

	tree := &Tree{DotType: named, Pkg: pkg.Types, Fset: fset}
	return tree, nil
}

// splitTypeHint splits a raw gotype hint into (importPath, typeName).
func splitTypeHint(hint string) (importPath, typeName string) {
	idx := strings.LastIndex(hint, ".")
	if idx == -1 {
		return ".", hint
	}
	return hint[:idx], hint[idx+1:]
}

// NamedMethods extracts the methods from the model
func NamedMethods(named *types.Named) []MethodType {
	var methods []MethodType
	for i := range named.NumMethods() {
		fn := named.Method(i)
		if !fn.Exported() {
			continue
		}

		sig := fn.Signature()
		results := sig.Results()

		if results.Len() == 0 || results.Len() > 2 {
			continue
		}

		var params []ParamType
		// if the generics are used in the functions, then sig.TypeParams should be extracted
		sigParams := sig.Params()
		for j := range sigParams.Len() {
			p := sigParams.At(j)
			params = append(params, ParamType{
				Name:     p.Name(),
				Type:     p.Type(),
				TypeName: types.TypeString(p.Type(), nil),
			})
		}

		ret := results.At(0)
		methods = append(methods, MethodType{
			Func:       fn,
			Name:       fn.Name(),
			ReturnType: ret.Type(),
			ReturnName: types.TypeString(ret.Type(), nil),
			Params:     params,
		})
	}
	return methods
}

// StructFields returns the exported fields of the struct
func StructFields(named *types.Named) []TypeField {
	// Underlying returns structs fields and types
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}

	fields := make([]TypeField, 0, st.NumFields())
	for i := range st.NumFields() {
		f := st.Field(i)
		// we can't access unexported fields
		if !f.Exported() {
			continue
		}
		fields = append(fields, TypeField{
			Name:     f.Name(),
			TypeName: types.TypeString(f.Type(), nil),
			Type:     f.Type(),
			Embedded: f.Embedded(),
		})
	}
	return fields
}
