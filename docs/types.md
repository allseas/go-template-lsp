# `types` Package

The `server/types` package builds a **typed parse tree** from a Go
`text/template` parse tree produced by the `parse` package. It annotates every
node with a `go/types.Type` so that downstream features (hover, completions,
definitions, type hints, diagnostics) can answer "what is the type of this
expression at this position?" without re-walking the source.

## Responsibilities

- Mirror every parse-tree node as a typed counterpart (see [node.go](../server/types/node.go)).
- Resolve types for fields, methods, variables, identifiers (functions),
  pipelines, chains and the `.` (dot) context.
- Track scope: variable declarations, dot rebinding inside `with` / `range`,
  scoped pops on block exit.
- Collect type errors as structured `TError` values rather than panicking.

## Public Surface

Defined in [analyse.go](../server/types/analyse.go):

| Symbol                                         | Purpose                                                                                                                                                                                                                     |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Tree`                                         | Typed counterpart of `parse.Tree`. Holds `Root *ListNode`, the function map, the optional `DotType` / `Pkg`, and accumulated `TypeErrors`.                                                                                  |
| `NewTree(parseTree, funcs, dotType, pkg) Tree` | Main constructor. Walks the parse tree and returns a fully analysed `Tree`. `funcs` carries the template's known global functions (typically `types.GlobalFuncs()` — see [features/func_hints.md](features/func_hints.md)). |
| `NewTreeWithType(...)`                         | Thin wrapper around `NewTree` for callers that have a `*types.Named` dot type.                                                                                                                                              |
| `TError`                                       | A type error attached to a specific typed `Node`, categorised by `ErrorType`.                                                                                                                                               |
| `ErrorType` constants                          | `ErrorTypeInvalidField`, `ErrorTypeInvalidFunction`, `ErrorTypeInvalidCommand`, `ErrorTypeInvalidRange`, `ErrorTypeInvalidIf`, `ErrorTypeInvalidWith`, `ErrorUndeclaredVariable`, `ErrorDoubleDeclaredVariable`.            |
| `Node` interface                               | Common interface for typed nodes; adds `ValueType() types.Type` and `IsElseList() bool` on top of the usual parse-node API.                                                                                                 |

Concrete node types (in [node.go](../server/types/node.go)) mirror the parse
package: `ListNode`, `ActionNode`, `PipeNode`, `CommandNode`, `IdentifierNode`,
`VariableNode`, `FieldNode`, `ChainNode`, `DotNode`, `IfNode`, `RangeNode`,
`WithNode`, `TemplateNode`, `NumberNode`, `StringNode`, `BoolNode`, `NilNode`,
`TextNode`, `CommentNode`, `BreakNode`, `ContinueNode`, `UndefinedNode`.

Each typed node stores its resolved `go/types.Type` in an unexported `typ`
field, exposed via `ValueType()`. A `nil` type means "unknown" — analysis is
best-effort and never aborts on a missing type.

## Analysis Flow

The entry point `NewTree` seeds an `analysisCtx` and walks the root:

```go
type analysisCtx struct {
    vars    []*VariableNode        // variables currently in scope
    dotType types.Type             // current dot (.) type
    funcs   map[string]*types.Func // template function map
    tree    *Tree                  // for appending TypeErrors
}
```

`analyseNode` dispatches on the parse node kind to a per-node helper
(`analyseList`, `analyseAction`, `analysePipe`, `analyseCommand`,
`analyseField`, `analyseVariable`, `analyseIdentifier`, `analyseChain`,
`analyseDot`, `analyseIf`, `analyseRange`, `analyseWith`, …).

### Scope rules

`analyseList` snapshots `len(ctx.vars)` on entry and truncates back to that
length on exit, popping any variables declared inside the list. The same
pattern is used inside branch helpers:

- **`with`** — evaluates the pipe in the outer dot, then sets
  `ctx.dotType = pipe.typ` for `List`, restores the outer dot for `ElseList`.
- **`range`** — like `with`, but `dotType` becomes the element type derived
  by `getRangeableType` (array / slice / map / chan element, or the integer
  itself for `range N`). The single-decl form binds `$v` to the element type;
  the two-decl form binds `$i` to `uint` and `$v` to the element type.
- **`if`** — does *not* introduce a new dot scope. Both `List` and `ElseList`
  are analysed with the surrounding dot. Variables declared in the condition
  pipe are popped on exit.

### Field / method chains

`walkFieldChain` resolves a sequence of identifiers against a starting type
using `types.LookupFieldOrMethod`. It is shared by `FieldNode`, `ChainNode`
and dotted `VariableNode` (`$x.A.B`). Methods are accepted when they return
one value, or two values where the second is `error`; anything else produces
an `ErrorTypeInvalidField`.

### Pipelines and commands

- `analyseCommand` types a command from its head argument. If the head is a
  function, the command type is either the function's result (when fully
  applied) or a *curried* `*types.Signature` taking the one remaining
  parameter (when one argument is missing). Other head kinds use the head's
  own type.
- `analysePipe` types a pipe as the type of its last command. For multi-stage
  pipes whose last stage is still a function (curried), the pipe's type is
  the first result of that signature.
- Declarations on a pipe (`{{ $x := … }}`) bind `$x` to the pipe's type and
  push it onto `ctx.vars`. Redeclaration in the same scope produces an
  `ErrorDoubleDeclaredVariable`. Reassignment (`{{ $x = … }}`) requires the
  variable to exist and the types to match.

### Mising Functionality

- checks on amount of params and their types when passed to a function
- support for variadic functions
- support for iter.Seq in a range
- indexing using key on maps in a range
- type checking between different templates
- special case for `call`

### Error reporting

`(*analysisCtx).errorf` appends a `TError` to `tree.TypeErrors` with the
offending node, a formatted message and an `ErrorType`. Analysis continues
with the affected node left untyped.

## File Layout

| File                                                         | Contents                                                                                                                                                                                                             |
| ------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [analyse.go](../server/types/analyse.go)                     | `Tree`, `NewTree`, `analysisCtx`, all `analyseXxx` helpers, `walkFieldChain`, `TError`.                                                                                                                              |
| [node.go](../server/types/node.go)                           | Typed `Node` interface and all concrete node structs, including `String()` / `Copy()` / `writeTo()` implementations.                                                                                                 |
| [analyse_test.go](../server/types/analyse_test.go)           | Table-driven test runner and structural comparison helpers used to assert tree equality.                                                                                                                             |
| [analyse_testcases.go](../server/types/analyse_testcases.go) | Test fixtures: mock types (`MockDot`, `Inner`), mock function map, parse-tree and typed-tree builder helpers, and the `analyseTestCases` table.                                                                      |
| [func_hints.go](../server/types/func_hints.go)               | Scans workspace Go sources for `//tmpl:func "global"` annotations and exposes the resulting `map[string]*types.Func` via `GlobalFuncs()` / `SetGlobalFuncs()`. See [features/func_hints.md](features/func_hints.md). |
| [func_hints_test.go](../server/types/func_hints_test.go)     | Unit tests for the global-function loader (uses [`test/resources/funcmap-tests`](../test/resources/funcmap-tests)).                                                                                                  |

`types/node.go` is excluded from linting and code coverage requirements, since it is taken from the standard Go library and only modified slightly.

## Testing

Tests live in the same package and are driven from a single table in
`analyse_testcases.go`. Each case supplies:

- a `parse.Tree` (built with the lowercase helpers — `tree`, `list`,
  `actpipe`, `pipe`, `com`, `field`, `varn`, `ifN`, `withN`, `rangeN`, …),
- the expected typed `Tree` (built with the `t`-prefixed mirrors — `ttree`,
  `tlist`, `tactpipe`, `tpipe`, `tcom`, `tfield`, `tvarn`, `tifN`, `twithN`,
  `trangeN`, …),
- a `funcs` map, optional `dotType` / `pkg`, and any expected `TError`s.

Run them with:

```sh
go test ./server/types/ -run TestAnalyze -v
```

To add a new case, append an `analyseTestCase` to `analyseTestCases`. Reuse
the existing builders rather than constructing nodes by hand so that fields
like `NodeType` stay consistent with what `analyse.go` produces.
