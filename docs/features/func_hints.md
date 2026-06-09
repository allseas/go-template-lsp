# Function Hints (`//tmpl:func`)

Function hints let the language server discover **user-defined template functions** in the workspace's Go source and expose them to templates with full type information - completion items, signatures for diagnostics, and (once wired) hover details.

## What the user writes

Annotate the Go function that returns a `FuncMap` (or any value with `FuncMap` in its type) with a `//tmpl:func "<scope>"` comment immediately above it:

```go
package funcs

import "text/template"

//tmpl:func "global"
func GlobalFuncs() template.FuncMap {
    return template.FuncMap{
        "upper":   strings.ToUpper,
        "lower":   strings.ToLower,
        "repeat":  strings.Repeat,
        "sprintf": fmt.Sprintf,
        "shout":   func(s string) string { return s + "!" },
    }
}
```

Scopes:

| Scope         | Behaviour                                                             |
| ------------- | --------------------------------------------------------------------- |
| `global`      | All entries are exposed to every template as global functions.        |
| anything else | Reserved for future per-template / per-tag scopes; currently ignored. |

The next `FuncMap` composite literal encountered after the comment is the one harvested - typically the literal returned inside the function body, but any literal of a type whose name is `FuncMap` (`template.FuncMap`, `html/template.FuncMap`, an alias, …) is accepted.

## Resolution flow

```
LSP Initialize
        │
        ▼
types.LoadGlobalFuncs(workspaceRoot)
 ├── packages.Load("./...", NeedName|NeedFiles|NeedSyntax|NeedTypes|NeedTypesInfo|NeedImports)
 └── for each file:
        ├── collectFuncMapLits(file)               → []*ast.CompositeLit
        └── for each //tmpl:func "global" comment:
              ├── nextFuncMap(lits, comment.End)   pick next literal in source order
              └── extractFuncMapInto(lit, info)    "name": expr → resolve expr → *types.Func
                       │
                       ▼
        types.SetGlobalFuncs(map[string]*types.Func)   (process-wide cache)
                       │
                       ▼
documentStore.Set → buildTypedTree → types.NewTree(parse, types.GlobalFuncs(), …)
                       │
                       ▼
        analyseIdentifier resolves global names to their *types.Func
        completion.allGlobalFunctions() unions builtins ∪ GlobalFuncs() keys
```

## Implementation details

**Comment detection**: `regexp` `tmpl:func\s+"([^"]+)"` matched against every comment in `file.Comments`. The doc-comment of a `FuncDecl` is included because `packages.Load` with `NeedSyntax` parses with `parser.ParseComments`.

**Map type check**: `isFuncMapType` accepts `FuncMap` either as a bare identifier or as the `Sel` of a selector expression (`template.FuncMap`, `html_template.FuncMap`, alias names, …). The actual underlying type is not checked - by convention only `FuncMap` literals are annotated.

**Function resolution**: `extractFuncMapInto` walks the literal's `Elts`, requires each key to be a string literal, and resolves the value:

| Value form       | Resolved to                            |
| ---------------- | -------------------------------------- |
| `Ident`          | `info.ObjectOf(ident).(*types.Func)`   |
| `SelectorExpr`   | `info.ObjectOf(sel.Sel).(*types.Func)` |
| function literal | `nil` (the name is still registered)   |
| anything else    | `nil`                                  |

A `nil` value means "the name is known but the signature is not". The identifier still completes; type-aware diagnostics simply skip it.

**Caching**: `SetGlobalFuncs` / `GlobalFuncs` guard a process-wide map with a `sync.RWMutex`. `GlobalFuncs` returns a snapshot so callers may mutate freely.

**Consumption**:

- [`server/types/analyse.go`](../../server/types/analyse.go) - `analyseIdentifier` looks names up in `ctx.funcs` (populated from the cache via `types.NewTree`). When found, the identifier node gets the function's signature as its `ValueType`, which lets pipes and commands compute downstream types.
- [`server/handlers/completion.go`](../../server/handlers/completion.go) - `allGlobalFunctions()` unions `builtinFunctions` with the cache keys and de-duplicates by name; builtins always win.

## Limitations

- Loading runs **once at initialize**. Edits to the workspace's Go sources are not picked up until the server restarts.
- Function literals (inline `func(…) … { … }` values) register the name but expose no signature.
- A hint above a function that does not return a `FuncMap` literal is silently ignored (the next literal of a different type is not matched).
- Only the `global` scope is implemented; other scope strings are reserved.

## Test fixture

[`test/resources/funcmap-tests`](../../test/resources/funcmap-tests) - a minimal module with one package exposing a `global` map and a non-global map; used by `TestLoadGlobalFuncs`.
