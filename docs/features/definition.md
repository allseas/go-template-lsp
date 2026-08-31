# Go to Definition

The definition provider enables jump-to-definition (Ctrl+Click) for nodes. It is implemented as an LSP `textDocument/definition` handler in the language server and is consumed by both the VS Code and JetBrains clients.

## What the user sees

| Cursor position                                 | Behavior                                                              |
| ----------------------------------------------- | --------------------------------------------------------------------- |
| `{{ $x }}` (variable usage)                     | Jumps to all declarations of `$x` (all `$x :=` assignments)           |
| `{{ $x := 0 }}` (variable declaration)          | Shows all declarations of `$x` (same as usage - IDE shows references) |
| `.` inside `{{ range .Items }}...{{ . }}...end` | Jumps to the `range` pipe that redefines the dot context              |
| `.` inside `{{ with .Obj }}...{{ . }}...end`    | Jumps to the `with` pipe that redefines the dot context               |
| `.FieldName` (field access)                     | Jumps to the field or method declaration in the Go source file        |
| `.Nested.Field` (nested field access)           | Jumps to whichever identifier the cursor is on                        |
| `kebabCase` (user defined global function)      | Jumps to the function declaration in the Go source file               |
| type token inside a `gotype:` comment           | Jumps to that type's declaration in the Go source file                |
| `.` in a template with a hint but no `range`/`with` scope | Jumps to the `gotype:` hint comment                          |

## Supported node types

### Variables (`$x`)

When the cursor is on any `VariableNode`, the handler walks the entire AST and collects all `PipeNode`s whose `Decl` list contains a matching variable name. All declaration locations are returned, which means redeclarations are handled correctly:

```gotmpl
{{ $test := 0 }}   {{-/* definition 1 */-}}
{{ $test }}        {{-/* ctrl+click here shows both definitions */-}}
{{ $test := 1 }}   {{-/* definition 2 */-}}
{{ $test }}        {{-/* ctrl+click here shows both definitions */-}}
```

> **Client behaviour difference:** The language server always returns all declaration sites for a
> redeclared variable. VS Code surfaces all of them in the *Go to Definition* result list. JetBrains
> (via LSP4IJ) currently only navigates to the **first** result returned by the server - subsequent
> declaration sites are silently ignored. This is a limitation of the LSP4IJ client, not the server.

### Dot (`.`)

When the cursor is on a `DotNode`, the handler uses `buildPath` to reconstruct the path from the tree root to the node, then walks the path backwards looking for the nearest `RangeNode` or `WithNode`. The pipe of that branch node is returned as the definition - since `range` and `with` are the constructs that redefine the dot context.

```gotmpl
{{- range .Join }}
    {{ . }}        {{-/* ctrl+click jumps to ".Join" in the range pipe */-}}
{{- end }}
```

When there is no enclosing `range`/`with` but the template carries a `gotype:` hint, the dot instead jumps to the hint comment, giving a discoverable anchor for the dot's type. From the hint comment, each type token is itself a definition target (see below).

### Type tokens inside a `gotype:` comment

When the cursor is on a slash-qualified type token inside a `gotype:` comment (e.g. `cg/model/controlmodel.Instance` in a composite or generic hint), the handler resolves that token's package and jumps to the type declaration in the Go source. This makes the hint comment the hub for navigating to every type a composite or generic hint references.

```gotmpl
{{- /*gotype: cg/template.View[cg/model/controlmodel.Instance]*/ -}}
{{-/* ctrl+click on View -> View decl; on Instance -> Instance decl */-}}
```

### Fields (`.FieldName`)

When the cursor is on a `FieldNode`, the handler resolves the Go type using the `gotype` hint comment (e.g. `{{/*gotype: cg/model.Order*/}}`) and calls `gotypes.LookupFieldOrMethod` to locate the field or method declaration in the Go source. It returns a `Location` pointing to the exact line in the Go source file.

For chained access like `.Address.City`, the handler determines which identifier the cursor is over by comparing byte offsets, resolves each intermediate type in turn, and jumps to the correct target.

Methods are also supported: the handler follows the method's return type when resolving chained access through a method call.

Field and method jumps resolve to the correct source file even when the chain crosses package boundaries — for example a generic type argument or a dict value declared in a different package than the hint's base type. Each package a hint touches is loaded with its own `token.FileSet`, and the handler selects the FileSet by the resolved object's package.

If no `gotype` hint is present, or the type cannot be loaded, the handler returns `nil`.

```gotmpl
{{/*gotype: cg/model.Order*/}}
{{ .CustomerName }}   {{-/* ctrl+click jumps to CustomerName field in model.go */-}}
{{ .DisplayName }}    {{-/* ctrl+click jumps to DisplayName method in model.go */-}}
{{ .Address.City }}   {{-/* ctrl+click on Address -> Address field; on City -> City field */-}}
```

### User-defined global functions (`functionName`)

When the cursor is on a custom global function, the handler knows the functions introduced by the function maps (funcmaps) with the `//tmpl:func "global"` from the global functions store in `types`. Then it returns a `Location` pointing to the exact line in the Go source file where that function was defined.

If it was an inline function, it will jump to the line in the funcmap where it was defined.

It does not work for builtin global functions, or those that were not imported via the `//tmpl:func` comment.
