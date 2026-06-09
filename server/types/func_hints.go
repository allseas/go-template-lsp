package types

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
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
// otherwise nil.
func LoadGlobalFuncs(workspaceRoot string) (map[string]*types.Func, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Dir: workspaceRoot,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}

	out := map[string]*types.Func{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			collectGlobalFuncs(file, pkg.TypesInfo, out)
		}
	}
	return out, nil
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
