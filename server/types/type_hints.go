// Package types implements utils for types and the type tree
package types

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

// describe returns a short human-readable rendering of the hint for use in
// diagnostic messages.
func (h TypeHint) describe() string {
	switch h.Type {
	case typeHintStruct:
		return h.Text
	case typeHintDict:
		return "map{" + h.Text + "}"
	case typeHintMalformedDict:
		return "malformed map{...}"
	default:
		return "<none>"
	}
}

// parseHintText parses the raw body of a comment (with or without leading
// whitespace) into a TypeHint. Line is left at 0; callers that need position
// information should use ParseHintComment instead.
func parseHintText(commentText string) (TypeHint, bool) {
	if dictHintRe.MatchString(commentText) {
		if h, ok := parseDictHint(commentText, 0); ok {
			return h, true
		}
		return TypeHint{Type: typeHintMalformedDict}, true
	}
	return parseStructHint(commentText, 0)
}

// hintsEqual reports whether two hints refer to the same type. Line numbers
// are ignored.
func hintsEqual(a, b TypeHint) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case typeHintStruct:
		return a.Text == b.Text
	case typeHintDict:
		if len(a.Dict) != len(b.Dict) {
			return false
		}
		for k, v := range a.Dict {
			if b.Dict[k] != v {
				return false
			}
		}
		return true
	case typeHintMalformedDict:
		return true
	default:
		return true
	}
}

var (
	dictHintRe = regexp.MustCompile(`gotype:\s*map\s*\{`)
	// dictEntryRe splits a single `"key": typeref` dict entry. The value is
	// captured loosely so that full type expressions (slices, maps, pointers,
	// nesting) reach resolveTypeExpr; it is validated with looksLikeTypeExpr.
	dictEntryRe = regexp.MustCompile(`^\s*"([^"]+)"\s*:\s*(.+?)\s*$`)
	// qualifiedTypeRe matches a slash-bearing import path followed by `.Type`
	// (e.g. `cg/model/controlmodel.Block`). Such references parse as a
	// division expression in go/parser, so preprocessHint rewrites them to
	// `lastSegment.Type` before ParseExpr runs. Group 1 is the import path,
	// group 2 the type name.
	qualifiedTypeRe = regexp.MustCompile(
		`([A-Za-z_][A-Za-z0-9_.\-]*(?:/[A-Za-z0-9_.\-]+)+)\.([A-Za-z_][A-Za-z0-9_]*)`,
	)
)

// FindTreeHints scans each parse tree for a `gotype:` comment and returns a
// map of template names to the first hint found in that tree. Any additional
// hint comments in the same tree are ignored here; they are detected during
// analysis and reported as ErrorTypeConflictingHint diagnostics.
func FindTreeHints(text string, trees map[string]*parse.Tree) map[string]TypeHint {
	result := make(map[string]TypeHint)

	for name, tree := range trees {
		if tree == nil || tree.Root == nil {
			continue
		}
		if h, ok := findFirstTreeHint(text, tree.Root); ok {
			result[name] = h
		}
	}

	return result
}

// findFirstTreeHint walks root and returns the first gotype hint comment found,
// stopping the traversal as soon as one is discovered.
func findFirstTreeHint(text string, root parse.Node) (TypeHint, bool) {
	for node := range walkParsed(root) {
		c, isComment := node.(*parse.CommentNode)
		if !isComment {
			continue
		}
		if h, ok := ParseHintComment(text, c); ok {
			return h, true
		}
	}
	return TypeHint{}, false
}

// ParseHintComment inspects a CommentNode and returns the gotype hint it
// carries, if any. A dict marker takes priority over struct parsing so a
// malformed `map{...}` body is not silently reinterpreted as a struct hint.
func ParseHintComment(text string, c *parse.CommentNode) (TypeHint, bool) {
	if c == nil {
		return TypeHint{}, false
	}
	line := strings.Count(text[:int(c.Pos)], "\n") + 1
	if dictHintRe.MatchString(c.Text) {
		if h, ok := parseDictHint(c.Text, line); ok {
			return h, true
		}
		return TypeHint{Type: typeHintMalformedDict, Line: line}, true
	}
	if h, ok := parseStructHint(c.Text, line); ok {
		return h, true
	}
	return TypeHint{}, false
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
// the whole hint. Values are stored verbatim (with any slash import paths);
// they are resolved later through resolveTypeExpr.
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
		key, value := m[1], m[2]
		if !looksLikeTypeExpr(value) {
			return nil, false
		}
		dict[key] = value
	}
	if len(dict) == 0 {
		return nil, false
	}
	return dict, true
}

func parseStructHint(commentText string, line int) (TypeHint, bool) {
	raw, ok := extractStructHintText(commentText)
	if !ok {
		return TypeHint{}, false
	}
	return TypeHint{
		Type: typeHintStruct,
		Text: raw,
		Line: line,
	}, true
}

// extractStructHintText pulls the raw type expression that follows `gotype:`
// out of a comment. The comment's trailing `*/` marker and surrounding
// whitespace are stripped. It returns ok=false when no `gotype:` marker is
// present or the remainder does not look like a Go type expression, so a
// stray comment is not misreported as a broken hint.
func extractStructHintText(commentText string) (string, bool) {
	idx := strings.Index(commentText, "gotype:")
	if idx < 0 {
		return "", false
	}
	rest := commentText[idx+len("gotype:"):]
	rest = strings.TrimSpace(rest)
	rest = strings.TrimSuffix(rest, "*/")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	if !looksLikeTypeExpr(rest) {
		return "", false
	}
	return rest, true
}

// walkParsed returns an iterator over node and its descendants in pre-order.
// The caller can break out of the range loop to stop the walk early.
func walkParsed(root parse.Node) iter.Seq[parse.Node] {
	var visit func(parse.Node, func(parse.Node) bool) bool
	visit = func(n parse.Node, yield func(parse.Node) bool) bool {
		if n == nil {
			return true
		}
		if !yield(n) {
			return false
		}
		for _, child := range parseNodeChildren(n) {
			if !visit(child, yield) {
				return false
			}
		}
		return true
	}
	return func(yield func(parse.Node) bool) {
		visit(root, yield)
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

type loadedPackage struct {
	pkg  *types.Package
	fset *token.FileSet
}

var (
	typeHintCacheMu sync.RWMutex
	typeHintCache   = make(map[string]*Tree)
	loadedPackages  = make(map[string]*loadedPackage)
)

// InvalidateTypeHintCache clears the cached type-hint results
func InvalidateTypeHintCache() {
	typeHintCacheMu.Lock()
	defer typeHintCacheMu.Unlock()
	typeHintCache = make(map[string]*Tree)
	loadedPackages = make(map[string]*loadedPackage)
}

// RegisterLoadedPackage caches a *types.Package under the same
// (importPath, workspaceRoot) key that loadPackageCached uses. It lets other
// loaders (e.g. LoadGlobalFuncs) seed the cache with packages they have
// already type-checked so that subsequent hint lookups reuse the exact same
// *types.Package — and therefore share *types.Named identity for
// types.Identical / types.Implements checks. First registration wins so
// callers can register speculatively without racing.
func RegisterLoadedPackage(
	importPath, workspaceRoot string,
	pkg *types.Package,
	fset *token.FileSet,
) {
	if pkg == nil || importPath == "" {
		return
	}
	key := importPath + "\x00" + workspaceRoot
	typeHintCacheMu.Lock()
	defer typeHintCacheMu.Unlock()
	if _, ok := loadedPackages[key]; !ok {
		loadedPackages[key] = &loadedPackage{pkg: pkg, fset: fset}
	}
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
	fsets := make(map[*types.Package]*token.FileSet)
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
		for p, fs := range lt.Fsets {
			if _, ok := fsets[p]; !ok {
				fsets[p] = fs
			}
		}
	}
	return &Tree{
		DictType: &DictType{Fields: fields},
		Pkg:      pkg,
		Fset:     fset,
		Fsets:    fsets,
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

// loadPackageCached loads (or returns a cached) *types.Package for the given
// import path. Every hint that resolves to the same package therefore shares
// one *types.Package instance, so *types.Named identity comparisons work
// across hints.
func loadPackageCached(importPath, workspaceRoot string) (*loadedPackage, error) {
	key := importPath + "\x00" + workspaceRoot

	typeHintCacheMu.RLock()
	if lp, ok := loadedPackages[key]; ok {
		typeHintCacheMu.RUnlock()
		return lp, nil
	}
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
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedSyntax,
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
		typeHintCacheMu.RUnlock()
		return nil, fmt.Errorf("packages.Load(%q): %w", importPath, err)
	}
	if len(pkgs) == 0 {
		log.Error().Str("importPath", importPath).Msg("LoadTypeFromHint: no packages found")
		typeHintCacheMu.RUnlock()
		return nil, fmt.Errorf("no packages found for import path %q", importPath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		log.Error().
			Str("importPath", importPath).
			Str("error", pkg.Errors[0].Msg).
			Msg("LoadTypeFromHint: package has errors")
		typeHintCacheMu.RUnlock()
		return nil, fmt.Errorf("package %q has errors: %v", importPath, pkg.Errors[0])
	}
	lp := &loadedPackage{pkg: pkg.Types, fset: fset}
	loadedPackages[key] = lp
	typeHintCacheMu.RUnlock()
	return lp, nil
}

// LoadTypeFromHint resolves the type expression carried by a struct-shaped
// gotype hint and returns a Tree with DotType, Pkg and Fset set. The hint may
// be an arbitrary Go type expression: a package-qualified named type
// (`pkg/path.Type`), a builtin (`int`), a pointer, slice, array, map, or any
// nesting of these (e.g. `map[string][]*pkg/path.Type`). Pkg/Fset are those of
// the first named type encountered during the walk, so go-to-definition keeps
// working.
func LoadTypeFromHint(hint, workspaceRoot string) (*Tree, error) {
	log.Debug().
		Str("hint", hint).
		Str("workspaceRoot", workspaceRoot).
		Msg("LoadTypeFromHint: attempting to resolve type hint")

	dot, pkg, fset, fsets, err := resolveTypeExpr(hint, workspaceRoot)
	if err != nil {
		log.Error().Err(err).Str("hint", hint).Msg("LoadTypeFromHint: failed to resolve type hint")
		return nil, err
	}

	log.Debug().
		Str("hint", hint).
		Str("type", types.TypeString(dot, nil)).
		Msg("LoadTypeFromHint: type resolved successfully")

	return &Tree{DotType: dot, Pkg: pkg, Fset: fset, Fsets: fsets}, nil
}

// LocateTypeDecl resolves the source declaration position of the type named
// typeName in the package importPath, loading the package if necessary. ok is
// false when the package or type cannot be found or has no source position.
func LocateTypeDecl(importPath, typeName, workspaceRoot string) (token.Position, bool) {
	lp, err := loadPackageCached(importPath, workspaceRoot)
	if err != nil || lp == nil || lp.pkg == nil || lp.fset == nil {
		return token.Position{}, false
	}
	obj := lp.pkg.Scope().Lookup(typeName)
	if obj == nil || !obj.Pos().IsValid() {
		return token.Position{}, false
	}
	return lp.fset.Position(obj.Pos()), true
}

// HintRefAt returns the import path and type name of the slash-qualified type
// token that covers byteOffset within commentText (a gotype comment's raw
// text). ok is false when no qualified token spans that offset.
func HintRefAt(commentText string, byteOffset int) (importPath, typeName string, ok bool) {
	for _, m := range qualifiedTypeRe.FindAllStringSubmatchIndex(commentText, -1) {
		// m[0:2] full match; m[2:4] group 1 (import path); m[4:6] group 2 (type).
		if byteOffset >= m[0] && byteOffset <= m[1] {
			return commentText[m[2]:m[3]], commentText[m[4]:m[5]], true
		}
	}
	return "", "", false
}

// preprocessHint rewrites every `import/path/with/slashes.Type` reference in a
// raw hint to `lastSegment.Type` so that go/parser.ParseExpr can parse it (a
// slash-bearing qualified name otherwise parses as a division expression). It
// records lastSegment -> full import path for later resolution and reports a
// conflict when two different import paths share the same last segment.
func preprocessHint(hint string) (rewritten string, imports map[string]string, err error) {
	imports = make(map[string]string)
	var conflict error
	rewritten = qualifiedTypeRe.ReplaceAllStringFunc(hint, func(match string) string {
		sub := qualifiedTypeRe.FindStringSubmatch(match)
		importPath, typeName := sub[1], sub[2]
		seg := importPath[strings.LastIndex(importPath, "/")+1:]
		if existing, ok := imports[seg]; ok && existing != importPath {
			if conflict == nil {
				conflict = fmt.Errorf(
					"conflicting import paths %q and %q map to the same package name %q",
					existing, importPath, seg,
				)
			}
			return match
		}
		imports[seg] = importPath
		return seg + "." + typeName
	})
	if conflict != nil {
		return "", nil, conflict
	}
	return rewritten, imports, nil
}

// looksLikeTypeExpr reports whether raw parses (after preprocessing) into a Go
// type expression shape the resolver understands. It performs no package
// loading, so it is safe to call while scanning for hints. A preprocess
// conflict counts as hint-shaped so the error surfaces later as a diagnostic.
func looksLikeTypeExpr(raw string) bool {
	rewritten, _, err := preprocessHint(raw)
	if err != nil {
		return true
	}
	expr, err := parser.ParseExpr(rewritten)
	if err != nil {
		return false
	}
	return isTypeExprShape(expr)
}

// isTypeExprShape reports whether expr is one of the type constructs the hint
// resolver supports. Predeclared/name resolution is not attempted here.
func isTypeExprShape(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		_, ok := e.X.(*ast.Ident)
		return ok
	case *ast.StarExpr:
		return isTypeExprShape(e.X)
	case *ast.ArrayType:
		return isTypeExprShape(e.Elt)
	case *ast.MapType:
		return isTypeExprShape(e.Key) && isTypeExprShape(e.Value)
	case *ast.IndexExpr:
		return isTypeExprShape(e.X) && isTypeExprShape(e.Index)
	case *ast.IndexListExpr:
		if !isTypeExprShape(e.X) {
			return false
		}
		for _, idx := range e.Indices {
			if !isTypeExprShape(idx) {
				return false
			}
		}
		return true
	case *ast.ParenExpr:
		return isTypeExprShape(e.X)
	default:
		return false
	}
}

// hintResolver walks a parsed type-expression AST and builds the matching
// go/types.Type, loading packages on demand. It records the first named
// type's package/fset so callers can preserve go-to-definition positions.
type hintResolver struct {
	workspaceRoot string
	imports       map[string]string
	pkg           *types.Package
	fset          *token.FileSet
	// fsets records every package the resolver loaded together with the
	// FileSet its positions belong to, so cross-package definition lookups can
	// pick the right FileSet for an object.
	fsets map[*types.Package]*token.FileSet
}

// resolveTypeExpr preprocesses, parses and resolves a raw hint into a
// go/types.Type. It also returns the package and fset of the first named type
// encountered (nil for pure-builtin hints such as `int` or `[]string`) and the
// per-package FileSet map covering every package the hint touched.
func resolveTypeExpr(
	hint, workspaceRoot string,
) (types.Type, *types.Package, *token.FileSet, map[*types.Package]*token.FileSet, error) {
	rewritten, imports, err := preprocessHint(hint)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	expr, err := parser.ParseExpr(rewritten)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("cannot parse type hint %q: %w", hint, err)
	}
	r := &hintResolver{
		workspaceRoot: workspaceRoot,
		imports:       imports,
		fsets:         make(map[*types.Package]*token.FileSet),
	}
	t, err := r.resolve(expr)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return t, r.pkg, r.fset, r.fsets, nil
}

// resolve recursively converts a type-expression AST node into a
// go/types.Type.
func (r *hintResolver) resolve(expr ast.Expr) (types.Type, error) {
	switch e := expr.(type) {
	case *ast.Ident:
		return r.resolveIdent(e.Name)
	case *ast.SelectorExpr:
		pkgIdent, ok := e.X.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf(
				"unsupported qualifier in type hint: %s", types.ExprString(expr),
			)
		}
		return r.resolveSelector(pkgIdent.Name, e.Sel.Name)
	case *ast.StarExpr:
		elem, err := r.resolve(e.X)
		if err != nil {
			return nil, err
		}
		return types.NewPointer(elem), nil
	case *ast.ArrayType:
		elem, err := r.resolve(e.Elt)
		if err != nil {
			return nil, err
		}
		if e.Len == nil {
			return types.NewSlice(elem), nil
		}
		n, err := arrayLen(e.Len)
		if err != nil {
			return nil, err
		}
		return types.NewArray(elem, n), nil
	case *ast.MapType:
		key, err := r.resolve(e.Key)
		if err != nil {
			return nil, err
		}
		val, err := r.resolve(e.Value)
		if err != nil {
			return nil, err
		}
		return types.NewMap(key, val), nil
	case *ast.IndexExpr:
		return r.resolveGeneric(e.X, []ast.Expr{e.Index})
	case *ast.IndexListExpr:
		return r.resolveGeneric(e.X, e.Indices)
	case *ast.ParenExpr:
		return r.resolve(e.X)
	default:
		return nil, fmt.Errorf(
			"unsupported type expression in hint: %s", types.ExprString(expr),
		)
	}
}

// resolveIdent resolves a bare identifier: a predeclared builtin from
// types.Universe, otherwise a type in the local package (import path ".").
func (r *hintResolver) resolveIdent(name string) (types.Type, error) {
	if obj, ok := types.Universe.Lookup(name).(*types.TypeName); ok {
		return obj.Type(), nil
	}
	return r.lookupType(".", name)
}

// resolveSelector resolves `pkg.Type`. When pkg was recorded by preprocessHint
// it maps to a full slash import path; otherwise the qualifier is itself the
// import path.
func (r *hintResolver) resolveSelector(qualifier, typeName string) (types.Type, error) {
	importPath := qualifier
	if full, ok := r.imports[qualifier]; ok {
		importPath = full
	}
	return r.lookupType(importPath, typeName)
}

// resolveGeneric instantiates a generic named type. baseExpr must resolve to a
// generic *types.Named and the number of type arguments must match its type
// parameters. Each argument is resolved to a type and the result is produced
// with types.Instantiate (which also validates the arguments against their
// constraints).
func (r *hintResolver) resolveGeneric(baseExpr ast.Expr, argExprs []ast.Expr) (types.Type, error) {
	baseType, err := r.resolve(baseExpr)
	if err != nil {
		return nil, err
	}
	named := namedTypeOf(baseType)
	if named == nil {
		return nil, fmt.Errorf(
			"%s is not a named type and cannot take type arguments",
			types.ExprString(baseExpr),
		)
	}
	if named.TypeParams().Len() != len(argExprs) {
		return nil, fmt.Errorf(
			"%s expects %d type argument(s), got %d",
			types.ExprString(baseExpr), named.TypeParams().Len(), len(argExprs),
		)
	}
	basePkg := named.Obj().Pkg()
	targs := make([]types.Type, len(argExprs))
	for i, ae := range argExprs {
		at, argErr := r.resolveTypeArg(ae, basePkg)
		if argErr != nil {
			return nil, argErr
		}
		targs[i] = at
	}
	inst, err := types.Instantiate(nil, named, targs, true)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate %s: %w", types.ExprString(baseExpr), err)
	}
	return inst, nil
}

// resolveTypeArg resolves a single type-argument expression. A bare,
// non-builtin identifier is first looked up in basePkg (the generic type's own
// package) so an unqualified argument such as `Instance` in `pkg.View[Instance]`
// resolves alongside `View`. Anything else (builtins, qualified names, nested
// composites) falls back to the standard resolution path.
func (r *hintResolver) resolveTypeArg(expr ast.Expr, basePkg *types.Package) (types.Type, error) {
	if id, ok := expr.(*ast.Ident); ok && basePkg != nil {
		if _, builtin := types.Universe.Lookup(id.Name).(*types.TypeName); !builtin {
			if obj := basePkg.Scope().Lookup(id.Name); obj != nil {
				if tn, ok := obj.(*types.TypeName); ok {
					if r.pkg == nil {
						r.pkg = basePkg
					}
					return tn.Type(), nil
				}
			}
		}
	}
	return r.resolve(expr)
}

// lookupType loads the package for importPath and returns the type named
// typeName, recording the first package/fset for definition support.
func (r *hintResolver) lookupType(importPath, typeName string) (types.Type, error) {
	lp, err := loadPackageCached(importPath, r.workspaceRoot)
	if err != nil {
		return nil, err
	}
	if r.pkg == nil {
		r.pkg = lp.pkg
		r.fset = lp.fset
	}
	if r.fsets != nil && lp.pkg != nil && lp.fset != nil {
		if _, ok := r.fsets[lp.pkg]; !ok {
			r.fsets[lp.pkg] = lp.fset
		}
	}
	obj := lp.pkg.Scope().Lookup(typeName)
	if obj == nil {
		return nil, fmt.Errorf("type %q not found in package %q", typeName, importPath)
	}
	// Accept any *types.TypeName, including non-named types such as a defined
	// alias to a builtin (`type Int = int`).
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil, fmt.Errorf("%q is not a type in package %q", typeName, importPath)
	}
	return tn.Type(), nil
}

// arrayLen extracts the integer length of a fixed-size array type expression.
func arrayLen(expr ast.Expr) (int64, error) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, fmt.Errorf("unsupported array length: %s", types.ExprString(expr))
	}
	n, err := strconv.ParseInt(lit.Value, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid array length %q: %w", lit.Value, err)
	}
	return n, nil
}

// namedTypeOf unwraps pointer indirection and aliases to reach the underlying
// *types.Named, returning nil when t is not (a pointer to) a named type.
func namedTypeOf(t types.Type) *types.Named {
	switch u := types.Unalias(t).(type) {
	case *types.Named:
		return u
	case *types.Pointer:
		if n, ok := types.Unalias(u.Elem()).(*types.Named); ok {
			return n
		}
	}
	return nil
}

// NamedMethods extracts the methods from the model. The dot type may be a
// named type or a pointer to one.
func NamedMethods(dot types.Type) []MethodType {
	named := namedTypeOf(dot)
	if named == nil {
		return nil
	}
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

// StructFields returns the exported fields of the struct. The dot type may be
// a named struct type or a pointer to one.
func StructFields(dot types.Type) []TypeField {
	named := namedTypeOf(dot)
	if named == nil {
		return nil
	}
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
