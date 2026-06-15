# Completions based on AST

Completions based on AST offer context-aware suggestions when the user types in (`.tmpl`) file. It is implemented as an LSP `textDocument/completion` handler and is consumed by both the VS Code and JetBrains clients  

## What the user sees

| Cursor position                              | Suggestions                                                                                                   |
| -------------------------------------------- |---------------------------------------------------------------------------------------------------------------|
| `{{ `**`$`**` }}`                            | All variables in scope (`$`, `$i`, `$v`, …) - without the `$` prefix and the current dot fields/methods       |
| `{{ `**`.`**` }}`                            | `.` item, or all fields and methods of the current dot type if a type hint is resolved                        |
| `{{ .Items\[0\].`**`▌`**` }}`                | Fields and methods available on the resolved field type (chained field accesses)                              |
| `{{ .Items \| `**`len`**` \| `**`▌`**` }}`   | Functions that accept `int` output: `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `not`, `print`, `printf`, `println`   |
| `{{ .IsAdmin \| `**`not`**` \| `**`▌`**` }}` | Functions that accept `bool` output: `and`, `or`, `not`, `print`, `printf`, `println`                         |
| `{{ .Name \| `**`html`**` \| `**`▌`**` }}`   | Functions that accept `string` output: `html`, `js`, `urlquery`, `len`, `print`, `printf`, `println`, `index` |
| `CommandNode`, `PipeNode`, `IfNode`, etc.    | All scope-aware function, variables and dot objects                                                           |

Completion is triggered automatically on `$` and `.`, or manually via the editor's invoke shortcut (`Ctrl + Space`).

## Request flow

```
User types / invokes completion
        │
        ▼
LSP client  ── textDocument/completion ──►  server/handlers/completion.go : completionWithFallback()
                                                        │
                                            1. attempt completionAst()
                                            2. on nil result → fall back to regex completion()
                                                        │
                                            completionAst()
                                            ├── store.Get(uri)              look up the parsed document
                                            ├── positionToOffset(text, pos) convert LSP position → byte offset
                                            ├── nodeFind(root, offset)      walk the AST to find the nearest preceding node
                                            ├── buildPath(root, cur, ctx)   reconstruct ancestor chain + collect in-scope vars
                                            ├── isInsideTemplate(text, off) guard: skip if cursor is outside {{ }}
                                            └── suggest(cur, parent, ctx, sChar, isInvoked)
                                                        │
                                                        ▼
                                            item builder functions
                                            (dotItem / varsToItems / builtinItems /
                                             typeFieldItems / typeMethodItems /
                                             pipeFilteredItems)
                                                        │
                                            ◄─── protocol.CompletionList ───────────
```

## Implementation details

### Entry point - `completionWithFallback()`

`completionWithFallback` is the LSP handler registered for `textDocument/completion`. It first delegates to `completionAst`. If that returns `nil` (parse failure, cursor outside a template block, node not found), it falls back to the simpler regex-based `completion` function, ensuring the user always receives some suggestions.

```
completionWithFallback()
 ├── completionAst()        primary path - AST-aware
 └── completion()           fallback  - regex-based
```

### AST completion - `completionAst()`

`completionAst` (`completion.go`) builds the full context before producing any items:

```
completionAst()
 ├── store.Get(uri)                   look up the parsed document
 ├── positionToOffset(text, pos)      convert LSP position → byte offset
 ├── nodeFind(root, offset)           walk the AST to find the nearest preceding node
 ├── buildPath(root, curNode, ctx)    reconstruct ancestor chain; populates ctx.Path, ctx.Vars, ctx.Pipe, ctx.DotType
 ├── isInsideTemplate(text, offset)   bail out if the cursor is not inside {{ … }}
 └── suggest(cur, parent, ctx, …)    dispatch to item builders
```

### Trigger-character shortcuts

The LSP server advertises `$` and `.` as trigger characters. `suggest` checks the character at the node's start position (`sChar`) before doing any parent-type dispatch:

- **`$`** → `varsToItems(ctx, true)` - all variables in scope, labels stripped of their `$` prefix.
- **`.`** → if a resolved `LoadedType` is available, `typeFieldItems` + `typeMethodItems`; otherwise a bare `.` item.
### Per-parent dispatch - `suggest()`

For all other positions, `suggest` switches on the type of the *parent* node in the ancestor path:

| Parent type                                                             | Items returned                                                |
| ----------------------------------------------------------------------- | ------------------------------------------------------------- |
| `CommandNode`, first argument                                           | built-in functions and user defined functions only            |
| `CommandNode`, later arguments                                          | `all()` - dot + vars + builtins, filtered by pipe output kind |
| `ChainNode`, `TemplateNode`                                             | `dotAndVars()` - dot + vars only                              |
| `PipeNode`, `IfNode`, `RangeNode`, `WithNode`, `ListNode`, `ActionNode` | `all()`                                                       |
| default                                                                 | `all()`                                                       |

`all()` first calls `pipeOutputKind` to check whether the preceding command in the pipe produces a typed output, and returns `pipeFilteredItems` when it does.

### Pipe output filtering - `pipeOutputKind()` / `pipeFilteredItems()`

When completions are chained with `|`, the item list is narrowed to functions that can meaningfully consume the preceding command's output type.

`pipeOutputKind` inspects `ctx.Pipe.Cmds`:

- Non-invoked: looks at the command *before* the current one (`len(cmds) - 2`).
- Invoked: looks at the current command itself (`len(cmds) - 1`).
  It resolves the first argument of that command to an `IdentifierNode` (in case it is not an `IdentifierNode`, `outputAny` is returned) and looks it up in `builtinOutput`, a map from built-in name to `outputKind`:

| `outputKind`    | Produced by                                            |
| --------------- | ------------------------------------------------------ |
| `outputInt`     | `len`                                                  |
| `outputBool`    | `not`, `and`, `or`, `eq`, `ne`, `lt`, `le`, `gt`, `ge` |
| `outputString`  | `html`, `js`, `urlquery`, `print`, `printf`, `println` |
| `outputUntyped` | `call`, `index`, `slice`                               |
| `outputAny`     | unknown identifier or no pipe                          |

`pipeFilteredItems` converts the `outputKind` to an allowlist from `functionsAccepting`, then returns only those built-in names together with dot and in-scope variables. When the kind is `outputUntyped` or `outputAny`, the full unfiltered list is returned.

### User-defined global functions

User-defined functions annotated with `//tmpl:func "global"` (see [func_hints.md](func_hints.md)) are included in completions through two paths:

- **AST path** (`completionAst` / `builtinItems`) - `builtinItems()` builds the hard-coded builtin list and then appends one `CompletionItemKindFunction` item per name in `serverTypes.GlobalFuncs()`, skipping any name already in the builtin list.
- **Regex fallback path** (`completion` / `allGlobalFunctions`) - `allGlobalFunctions()` unions `builtinFunctions` with `GlobalFuncs()` keys and de-duplicates; builtins always win.

Both paths read from the same process-wide cache, which is populated at initialize and refreshed whenever a `.go` file changes in the workspace.

### Scope-aware variable collection - `buildPath()`

Variable tracking is performed inside `buildPath` / `buildPathChildren` as the ancestor chain is being reconstructed. When a `PipeNode` with declarations is encountered, each declared variable is registered in `ctx.Vars` before recursing into its children. `buildPathBranch` takes a snapshot of `ctx.Vars` and restores it if the target is not found in that branch, preventing variables from leaking across sibling branches.

### Type-aware field and method completion

When a `LoadedType` is attached to the document (resolved from a `// tmpl: Type` hint in the file header), hovering `.` produces `typeFieldItems` and `typeMethodItems` instead of the generic dot item. Both carry a `Detail` string - the Go type name for fields, the return type name for methods - which editors render as a secondary label in the completion menu.

`buildPathChildren` propagates and narrows the dot type as the tree is descended:

- **`RangeNode`**: calls `resolvePipeDotType` with `unwrapSlice: true` - unwraps a `[]T` field to `T` so completions inside a range body reflect the element type.
- **`WithNode`**: calls `resolvePipeDotType` with `unwrapSlice: false` - narrows to the field's struct type without unwrapping.
### Item builder functions

| Function                       | Items produced                                                                   |
| ------------------------------ | -------------------------------------------------------------------------------- |
| `dotItem()`                    | Single `.` variable item                                                         |
| `varsToItems(ctx, delSign)`    | One item per entry in `ctx.Vars`; strips `$` when `delSign` is true              |
| `builtinItems()`               | Fixed list of all built-in names (`and`, `call`, `html`, …, `ge`, `if`, `range`) |
| `typeFieldItems(fields)`       | One field item per `TypeField`; `Detail` = type name                             |
| `typeMethodItems(methods)`     | One method item per `MethodType`; `Detail` = return type name                    |
| `pipeFilteredItems(kind, ctx)` | Dot + vars + allowlisted built-in names for the given `outputKind`               |

### Ordering of completion items

When both fields and methods are present, **fields are listed before methods**. This is an intentional design decision to make it more clear and readable for the user. IDEs automatically sort everything alphabetically, but we override it for user convinince

## Tests

Tests live in `server/handlers/completion_ast_test.go`.

Each test case specifies:

- A template document string, optionally including a type hint comment.
- A cursor position (`line`, `character`).
- The expected `protocol.CompletionList`