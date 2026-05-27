# Go to Definition

The definition provider enables jump-to-definition (Ctrl+Click) for nodes. It is implemented as an LSP `textDocument/definition` handler in the language server and is consumed by both the VS Code and JetBrains clients.

## What the user sees

| Cursor position                                 | Behavior                                                              |
| ----------------------------------------------- | --------------------------------------------------------------------- |
| `{{ $x }}` (variable usage)                     | Jumps to all declarations of `$x` (all `$x :=` assignments)           |
| `{{ $x := 0 }}` (variable declaration)          | Shows all declarations of `$x` (same as usage - IDE shows references) |
| `.` inside `{{ range .Items }}...{{ . }}...end` | Jumps to the `range` pipe that redefines the dot context              |
| `.` inside `{{ with .Obj }}...{{ . }}...end`    | Jumps to the `with` pipe that redefines the dot context               |
| `.FieldName` (field access)                     | No in-template definition (field is defined in Go source)             |

## Supported node types

### Variables (`$x`)

When the cursor is on any `VariableNode`, the handler walks the entire AST and collects all `PipeNode`s whose `Decl` list contains a matching variable name. All declaration locations are returned, which means redeclarations are handled correctly:

```gotmpl
{{ $test := 0 }}   {{-/* definition 1 */-}}
{{ $test }}        {{-/* ctrl+click here shows both definitions */-}}
{{ $test := 1 }}   {{-/* definition 2 */-}}
{{ $test }}        {{-/* ctrl+click here shows both definitions */-}}
```

### Dot (`.`)

When the cursor is on a `DotNode`, the handler uses `buildPath` to reconstruct the path from the tree root to the node, then walks the path backwards looking for the nearest `RangeNode` or `WithNode`. The pipe of that branch node is returned as the definition — since `range` and `with` are the constructs that redefine the dot context.

```gotmpl
{{- range .Join }}
    {{ . }}        {{-/* ctrl+click jumps to ".Join" in the range pipe */-}}
{{- end }}
```

### Fields (`.FieldName`)

Field nodes refer to fields on the Go data structure passed to the template. Since there is no in-template definition, the handler returns `nil` (no result).
