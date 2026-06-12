# Function Hints (`//tmpl:func`)

Function hints let the language server discover **user-defined template functions** in the workspace's Go source and expose them to templates with full type information - completion items, type-checking in diagnostics, and hover details.

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

### Initial load (LSP initialize)

```mermaid
flowchart LR
    A[LSP Initialize] --> B["LoadGlobalFuncs - packages.Load + extract FuncMap literals"]
    B --> C["SetGlobalFuncs - update process-wide cache"]
    C --> D["Rebuild all typed trees with new funcs"]
    D --> E["completions & diagnostics consume GlobalFuncs()"]
```

### Hot reload (workspace/didChangeWatchedFiles)

```mermaid
flowchart LR
    A["any .go file saved"] --> B["LoadGlobalFuncs - re-scan workspace"]
    B --> C["SetGlobalFuncs - update cache"]
    C --> D["RefreshAllDocuments - rebuild every open template"]
    D --> E[publishDiagnostics for each document]
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

- [`server/types/analyse.go`](../../server/types/analyse.go) - `analyseIdentifier` looks names up in `ctx.funcs` (populated from the cache via `types.NewTree`). When found, the identifier node gets the function's signature as its `ValueType`, which lets pipes and commands compute downstream types. Unknown names produce an `ErrorTypeInvalidFunction` diagnostic only when absent from both the builtin list **and** `GlobalFuncs()`.
- [`server/handlers/completions_ast.go`](../../server/handlers/completions_ast.go) - `builtinItems()` appends one `CompletionItemKindFunction` item per `GlobalFuncs()` key (de-duplicating against the hard-coded builtin names).
- [`server/handlers/completion.go`](../../server/handlers/completion.go) - `allGlobalFunctions()` unions `builtinFunctions` with the cache keys for the regex-based fallback completion path; builtins always win.
- [`server/handlers/diagnostics.go`](../../server/handlers/diagnostics.go) - the `CommandNode` visitor allows an identifier if it is in `builtinOutput` **or** in `GlobalFuncs()`, suppressing the "unsupported function" diagnostic for registered user-defined functions.

## Limitations

- Function literals (inline `func(…) … { … }` values) register the name but expose no signature - they still complete and are not reported as unknown, but no type flows from them.
- A hint above a function that does not return a `FuncMap` literal is silently ignored (the next literal of a different type is not matched).
- Only the `global` scope is implemented; other scope strings are reserved.
- `packages.Load` resolves against the workspace root; workspaces with multiple Go modules at different roots may not pick up all packages.

## Live reload

The server registers a dynamic `workspace/didChangeWatchedFiles` watcher for `**/*.go` via `client/registerCapability` during the `initialized` callback (see [`server/handlers/watched_files.go`](../../server/handlers/watched_files.go)). Editors that support dynamic registration (VS Code, JetBrains LSP4IJ) will push `.go` change notifications automatically. On receipt the global-function cache is reloaded and every open template document is rebuilt, so completions and diagnostics update without a server restart.

## Test fixture

[`test/resources/funcmap-tests`](../../test/resources/funcmap-tests) - a minimal module with one package exposing a `global` map (`upper`, `lower`, `repeat`, `shout`, `sprintf`) and a non-global map (`localOnly`). Used by:

- `TestLoadGlobalFuncs` - end-to-end load via `packages.Load`.
- `TestCollectGlobalFuncs_OnlyGlobalHint` - inline AST test confirming non-global scopes are filtered.
- `TestGlobalFuncsCacheRoundTrip` - snapshot isolation of the cache.
- `TestIsFuncMapType` - unit coverage for the type-name matcher.
