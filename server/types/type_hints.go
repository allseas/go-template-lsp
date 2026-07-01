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
	"sort"
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
	typeHintMalformedDict
)

// TypeHint represents a `gotype:` type hint found in a template file.
type TypeHint struct {
	Type typeHintType
	// Text is the raw type reference that follows `gotype:` in the comment.
	// For struct hints this is the type path (e.g. "example.com/m.Order").
	// For dict hints this is the raw body between the braces of `map{...}`.
	Text string
	// Dict is populated for dict hints; it maps each declared key to its type
	// reference (e.g. "Order" -> "example.com/m.Order"). Nil for struct hints.
	Dict map[string]string
	// Line is the 1-based line number in the source text at which the hint
	// appears; 0 when the hint is unset.
	Line int
}

// IsMalformed reports whether the hint was recognised as a map hint but its
// body could not be parsed.
func (h TypeHint) IsMalformed() bool { return h.Type == typeHintMalformedDict }

var (
	structHintRe = regexp.MustCompile(`gotype:\s*([A-Za-z_][A-Za-z0-9_/.-]*)`)
	dictHintRe   = regexp.MustCompile(`gotype:\s*map\s*\{`)
	dictEntryRe  = regexp.MustCompile(`^\s*"([^"]+)"\s*:\s*([A-Za-z_][A-Za-z0-9_/.-]*)\s*$`)
)

// FindTreeHints scans each parse tree for a `gotype:` comment and returns a
// map of template names to the first hint found in that tree.
func FindTreeHints(text string, trees map[string]*parse.Tree) map[string]TypeHint {
	result := make(map[string]TypeHint)

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
			line := strings.Count(text[:int(c.Pos)], "\n") + 1
			// A dict marker takes priority: even if the body is malformed we
			// must not fall back to struct parsing, otherwise the leading
			// `dict` identifier would be captured as a struct type name.
			if dictHintRe.MatchString(c.Text) {
				if h, ok := parseDictHint(c.Text, line); ok {
					hint = h
				} else {
					hint = TypeHint{Type: typeHintMalformedDict, Line: line}
				}
				return
			}
			if h, ok := parseStructHint(c.Text, line); ok {
				hint = h
			}
		})
		if hint.Type != typeHintNone {
			result[name] = hint
		}
	}

	return result
}

// parseDictHint tries to interpret commentText as `gotype: map{...}`. It
// returns ok=false when the comment does not contain a dict marker at all;
// when the marker is present but the body is malformed the returned ok is
// still false so the caller does not fall back to struct parsing.
func parseDictHint(commentText string, line int) (TypeHint, bool) {
	loc := dictHintRe.FindStringIndex(commentText)
	if loc == nil {
		return TypeHint{}, false
	}
	rest := commentText[loc[1]:]
	end := strings.Index(rest, "}")
	if end < 0 {
		return TypeHint{}, false
	}
	body := rest[:end]
	dict, ok := parseDictBody(body)
	if !ok {
		return TypeHint{}, false
	}
	return TypeHint{
		Type: typeHintDict,
		Text: strings.TrimSpace(body),
		Dict: dict,
		Line: line,
	}, true
}

// parseDictBody parses the comma-separated `"key": typeref` entries between
// the braces of a dict hint. An empty body or any malformed entry rejects
// the whole hint.
func parseDictBody(body string) (map[string]string, bool) {
	entries := strings.Split(body, ",")
	dict := make(map[string]string, len(entries))
	for _, e := range entries {
		if strings.TrimSpace(e) == "" {
			return nil, false
		}
		m := dictEntryRe.FindStringSubmatch(e)
		if m == nil {
			return nil, false
		}
		dict[m[1]] = m[2]
	}
	if len(dict) == 0 {
		return nil, false
	}
	return dict, true
}

func parseStructHint(commentText string, line int) (TypeHint, bool) {
	m := structHintRe.FindStringSubmatch(commentText)
	if len(m) < 2 {
		return TypeHint{}, false
	}
	return TypeHint{
		Type: typeHintStruct,
		Text: m[1],
		Line: line,
	}, true
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

// DictType is a synthetic types.Type representing a `gotype: map{...}` hint.
// It behaves like a struct with named keys of arbitrary Go types, but is not
// a real Go type — LookupFieldOrMethod does not work on it. The analyser and
// completion code type-assert on *DictType to detect it.
type DictType struct {
	Fields map[string]types.Type
}

// Underlying implements types.Type; a dict is its own underlying.
func (d *DictType) Underlying() types.Type { return d }

// String implements types.Type. Keys are sorted so the output is stable.
func (d *DictType) String() string {
	if d == nil {
		return "map{}"
	}
	keys := d.DictKeys()
	var b strings.Builder
	b.WriteString("map{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q: %s", k, types.TypeString(d.Fields[k], nil))
	}
	b.WriteString("}")
	return b.String()
}

// LookupDictKey returns the value type for name, ok=false if absent.
func (d *DictType) LookupDictKey(name string) (types.Type, bool) {
	if d == nil {
		return nil, false
	}
	t, ok := d.Fields[name]
	return t, ok
}

// DictKeys returns the keys in sorted order for deterministic output.
func (d *DictType) DictKeys() []string {
	if d == nil {
		return nil
	}
	keys := make([]string, 0, len(d.Fields))
	for k := range d.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// DictTypeFields projects a *DictType into TypeField rows so completion code
// can treat dict keys and struct fields uniformly.
func DictTypeFields(d *DictType) []TypeField {
	if d == nil {
		return nil
	}
	keys := d.DictKeys()
	fields := make([]TypeField, 0, len(keys))
	for _, k := range keys {
		t := d.Fields[k]
		fields = append(fields, TypeField{
			Name:     k,
			TypeName: types.TypeString(t, nil),
			Type:     t,
		})
	}
	return fields
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

// CachedLoadHint dispatches on the hint kind and delegates to the appropriate
// cached loader. Struct hints go through CachedLoadTypeFromHint; dict hints go
// through CachedLoadDictFromHint.
func CachedLoadHint(hint TypeHint, workspaceRoot string) (*Tree, error) {
	switch hint.Type {
	case typeHintDict:
		return CachedLoadDictFromHint(hint, workspaceRoot)
	case typeHintStruct:
		return CachedLoadTypeFromHint(hint.Text, workspaceRoot)
	case typeHintMalformedDict:
		return nil, fmt.Errorf("malformed map hint")
	default:
		return nil, fmt.Errorf("unknown hint type")
	}
}

// dictCacheKey returns a deterministic key for a dict hint independent of map
// iteration order.
func dictCacheKey(hint TypeHint, workspaceRoot string) string {
	keys := make([]string, 0, len(hint.Dict))
	for k := range hint.Dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("dict\x00")
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(hint.Dict[k])
		b.WriteByte('\x01')
	}
	b.WriteString("\x00")
	b.WriteString(workspaceRoot)
	return b.String()
}

// CachedLoadDictFromHint is the cached counterpart of LoadDictFromHint.
func CachedLoadDictFromHint(hint TypeHint, workspaceRoot string) (*Tree, error) {
	key := dictCacheKey(hint, workspaceRoot)

	typeHintCacheMu.RLock()
	if t, ok := typeHintCache[key]; ok {
		typeHintCacheMu.RUnlock()
		return t, nil
	}
	typeHintCacheMu.RUnlock()

	t, err := LoadDictFromHint(hint, workspaceRoot)
	if err != nil {
		return nil, err
	}

	typeHintCacheMu.Lock()
	typeHintCache[key] = t
	typeHintCacheMu.Unlock()

	return t, nil
}

// LoadDictFromHint loads every value type of a dict hint and returns a Tree
// whose DictType is populated. DotType is left nil.
func LoadDictFromHint(hint TypeHint, workspaceRoot string) (*Tree, error) {
	if hint.Type != typeHintDict {
		return nil, fmt.Errorf("LoadDictFromHint: hint is not a dict")
	}
	if len(hint.Dict) == 0 {
		return nil, fmt.Errorf("LoadDictFromHint: dict is empty")
	}
	fields := make(map[string]types.Type, len(hint.Dict))
	var pkg *types.Package
	var fset *token.FileSet
	for _, k := range sortedKeys(hint.Dict) {
		ref := hint.Dict[k]
		lt, err := LoadTypeFromHint(ref, workspaceRoot)
		if err != nil {
			return nil, fmt.Errorf("map key %q (%s): %w", k, ref, err)
		}
		fields[k] = lt.DotType
		if pkg == nil {
			pkg = lt.Pkg
		}
		if fset == nil {
			fset = lt.Fset
		}
	}
	return &Tree{
		DictType: &DictType{Fields: fields},
		Pkg:      pkg,
		Fset:     fset,
	}, nil
}

// sortedKeys returns the keys of m in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
