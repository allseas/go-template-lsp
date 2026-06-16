package types

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"

	"golang.org/x/tools/go/packages"
)

// funcHintRe matches a `//tmpl:func "name"` marker comment.
var funcHintRe = regexp.MustCompile(`tmpl:func\s+"([^"]+)"`)

// globalHintName is the reserved hint name whose FuncMap entries are exposed
// as template-global functions.
const globalHintName = "global"

var (
	globalFuncsMu    sync.RWMutex
	globalFuncsCache map[string]*types.Func
)

// SetGlobalFuncs replaces the cached global function map. A nil map clears it.
func SetGlobalFuncs(m map[string]*types.Func) {
	globalFuncsMu.Lock()
	defer globalFuncsMu.Unlock()
	globalFuncsCache = m
}

// GlobalFuncs returns a snapshot of the cached global function map.
// Values may be nil when a signature could not be resolved.
func GlobalFuncs() map[string]*types.Func {
	globalFuncsMu.RLock()
	defer globalFuncsMu.RUnlock()
	if globalFuncsCache == nil {
		return nil
	}
	out := make(map[string]*types.Func, len(globalFuncsCache))
	for k, v := range globalFuncsCache {
		out[k] = v
	}
	return out
}

// LoadGlobalFuncs scans every Go package under workspaceRoot for FuncMap
// literals annotated with `//tmpl:func "global"` and returns the union of
// their entries. Values are *types.Func when the signature can be resolved,
// otherwise nil. Builtin functions are NOT included; use ComputeGlobalFuncs
// to obtain the merged set.
func LoadGlobalFuncs(workspaceRoot string) (map[string]*types.Func, error) {
	out := map[string]*types.Func{}

	// Find all package roots to load (handles nested modules)
	packageRoots, err := findPackageRoots(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("findPackageRoots: %w", err)
	}

	for _, root := range packageRoots {
		cfg := &packages.Config{
			Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
				packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
			Dir: root,
		}
		pkgs, err := packages.Load(cfg, "./...")
		if err != nil {
			continue
		}

		for _, pkg := range pkgs {
			for _, file := range pkg.Syntax {
				collectGlobalFuncs(file, pkg.TypesInfo, out)
			}
		}
	}

	return out, nil
}

// findPackageRoots recursively discovers all directories that contain Go packages
// (either with go.mod or .go files). Returns roots from which to load packages.
func findPackageRoots(dir string) ([]string, error) {
	var roots []string
	seen := make(map[string]bool)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		if len(info.Name()) > 0 && info.Name()[0] == '.' {
			return filepath.SkipDir
		}
		// skip common cache directories
		if info.Name() == "node_modules" || info.Name() == "vendor" {
			return filepath.SkipDir
		}

		// Check if this directory has a go.mod file (module root)
		gomod := filepath.Join(path, "go.mod")
		if _, err := os.Stat(gomod); err == nil {
			if !seen[path] {
				roots = append(roots, path)
				seen[path] = true
			}
			// Don't descend into subdirectories of a module root
			// as packages.Load with "./..." will find them
			if path != dir {
				return filepath.SkipDir
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// If no go.mod files found, add the root itself (it may be a standalone package)
	if len(roots) == 0 {
		roots = append(roots, dir)
	}

	return roots, nil
}

// ComputeGlobalFuncs returns the full set of functions that should be available
// to every template: the language builtins merged with any workspace-defined
// functions discovered via `//tmpl:func "global"` annotations.
//
// Workspace-defined names that collide with a builtin are silently dropped, as
// builtins cannot be shadowed.
//
// If workspaceRoot is empty only the builtins are returned.
func ComputeGlobalFuncs(workspaceRoot string) (map[string]*types.Func, error) {
	merged := BuiltinFuncs()

	if workspaceRoot == "" {
		return merged, nil
	}

	workspace, err := LoadGlobalFuncs(workspaceRoot)
	if err != nil {
		return merged, fmt.Errorf("LoadGlobalFuncs: %w", err)
	}
	for k, v := range workspace {
		if _, isBuiltin := merged[k]; !isBuiltin {
			merged[k] = v
		}
	}
	return merged, nil
}

// collectGlobalFuncs walks file and merges FuncMap entries marked with
// //tmpl:func "global" into out.
func collectGlobalFuncs(file *ast.File, info *types.Info, out map[string]*types.Func) {
	lits := collectFuncMapLits(file)
	if len(lits) == 0 {
		return
	}

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			m := funcHintRe.FindStringSubmatch(c.Text)
			if m == nil || m[1] != globalHintName {
				continue
			}
			target := nextFuncMap(lits, c.End())
			if target == nil {
				continue
			}
			extractFuncMapInto(target, info, out)
		}
	}
}

func collectFuncMapLits(file *ast.File) []*ast.CompositeLit {
	var lits []*ast.CompositeLit
	ast.Inspect(file, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if isFuncMapType(cl.Type) {
			lits = append(lits, cl)
		}
		return true
	})
	return lits
}

func nextFuncMap(lits []*ast.CompositeLit, after token.Pos) *ast.CompositeLit {
	var best *ast.CompositeLit
	for _, cl := range lits {
		if cl.Pos() <= after {
			continue
		}
		if best == nil || cl.Pos() < best.Pos() {
			best = cl
		}
	}
	return best
}

func isFuncMapType(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.Ident:
		return t != nil && t.Name == "FuncMap"
	case *ast.SelectorExpr:
		return t != nil && t.Sel != nil && t.Sel.Name == "FuncMap"
	}
	return false
}

func extractFuncMapInto(cl *ast.CompositeLit, info *types.Info, out map[string]*types.Func) {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		bl, ok := kv.Key.(*ast.BasicLit)
		if !ok || bl.Kind != token.STRING {
			continue
		}
		name, err := strconv.Unquote(bl.Value)
		if err != nil {
			continue
		}
		if _, seen := out[name]; seen {
			continue
		}
		out[name] = resolveFuncObj(kv.Value, info)
	}
}

func resolveFuncObj(expr ast.Expr, info *types.Info) *types.Func {
	if info == nil {
		return nil
	}
	var ident *ast.Ident
	switch v := expr.(type) {
	case *ast.Ident:
		ident = v
	case *ast.SelectorExpr:
		ident = v.Sel
	default:
		return nil
	}
	if ident == nil {
		return nil
	}
	fn, _ := info.ObjectOf(ident).(*types.Func)
	return fn
}
