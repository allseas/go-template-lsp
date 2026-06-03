# Diagnostics

Diagnostics report errors in template files as squiggly underlines. They are published via the LSP `textDocument/publishDiagnostics` notification whenever a document is opened or changed.

## What the user sees

| Template input               | Diagnostic message                                  | Range                                        |
| ---------------------------- | --------------------------------------------------- | -------------------------------------------- |
| `{{ $?????`                  | `undefined variable: bad character U+003F '?'`      | The `{{ … }}` block containing the bad token |
| `{{ }}`                      | `template: …: missing value for command`            | The full `{{ }}`                             |
| `{{ $x }}` (undeclared)      | `undefined variable: $x`                            | The `{{ … }}` block                          |
| `{{ foo }}` Unknown function | `unsupported function or unregistered command: foo` | The `{{ … }}` block                          |

## Request flow

```
Document opened / changed
        │
        ▼
handlers/documents.go : onOpen() / onChange()
        │
        ▼
handlers/diagnostics.go : publishDiagnostics()
        │
        ├── collectDiagnostics(text, uri)
        │       │
        │       ├── store.Get(uri)          re-use already-parsed tree if available
        │       ├── tryParse(text)          otherwise parse with ParsePartial mode
        │       └── walkAndAnalyze(root, text, ctx, visited, analyzeNode)
        │               │
        │               └── analyzeNode(node, text, ctx)   called for every AST node
        │                       │
        │                       ├── declareNode()           record variable declarations
        │                       └── switch node type
        │                               ├── UndefinedNode   → error from parser (see below)
        │                               ├── ActionNode      → checkPipeUsage()
        │                               ├── RangeNode       → checkPipeUsage()
        │                               ├── IfNode          → checkPipeUsage()
        │                               ├── WithNode        → checkPipeUsage()
        │                               └── CommandNode     → unknown function check
        │
        ▼
ctx.Notify(publishDiagnostics, diagnostics)
```

## UndefinedNode

There are two distinct categories of `UndefinedNode`:

### 1. Real error nodes (`str != ""`)

Created when the lexer encounters a bad token (e.g. `bad character U+003F '?'`) or the parser encounters something it cannot handle. `n.str` contains the original source fragment and `n.Err` holds the corresponding error. These are reported directly as diagnostics.

```
{{ $?????
         ↑
         lexer.errorf("bad character U+003F '?'")
         → UndefinedNode{Pos: <offset of $>, str: "bad character U+003F '?'", Err: …}
```

### 2. Recovery markers (`str == ""`)

`checkPipeline` (parse.go) inserts an empty-str `UndefinedNode` when a pipeline has no commands at all (e.g. `{{ }}`). There are two sub-cases:

| Sub-case                                 | Err message                     | Action                                                                                     |
| ---------------------------------------- | ------------------------------- | ------------------------------------------------------------------------------------------ |
| `{{ }}` — empty action with no lex error | `"… missing value for command"` | Report the error; position is valid                                                        |
| `{{ $?????` — post-lex-error recovery    | nil or unrelated message        | Skip; the real error is already covered by the non-empty `UndefinedNode` for the bad token |

The server handles these with the following logic (`analyzeNode`):

```
str != ""                               →  report as "undefined variable: <str>"
str == "" && err contains "missing value"  →  report using err.Error() as message
str == "" && everything else            →  skip (structural artifact)
```

`lexer.errorf()` records the error item and returns nil to terminate the current state-machine step. It does not modify `l.pos`, `l.start`, or `l.input`, so subsequent `nextItem()` calls resume lexing from the position immediately after the bad token:

```go
func (l *lexer) errorf(format string, args ...any) stateFn {
    l.item = item{itemError, l.start, fmt.Sprintf(format, args...), l.startLine}
    return nil
}
```

This means each bad character in `{{ $?????` produces its own error token (and therefore its own `UndefinedNode` and diagnostic), rather than one diagnostic covering the whole run.

## Range computation

Diagnostics use `expandToFullBracketsFromOffset(pos, text)` to expand a byte offset to the full surrounding `{{ … }}` block:

1. Search backwards from `pos` for the nearest `{{` — that becomes the start.
2. Search forwards from `pos` for the nearest `}}` — that becomes the end.
3. If no `}}` is found before a newline, the end is capped at the newline (handles unterminated actions).

Both offsets are then converted to `(line, character)` positions with `offsetToPosition`.
