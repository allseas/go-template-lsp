# Parser

This is an extended version of the go/text/template/parse package.

## Changes implemented

- added new node type (Undefined)
- added new operation mode to the parser (ParsePartial)
- added non-breaking error handling in ParsePartial mode
- added tests enforcing tree structure on malformed input
- added new field to Tree: Errors[]

- although some execution paths on the default mode have been slightly modified, they only get reached in case of incorrect input. And still return a nil tree, as well as the same error message as before. Thus not violating backwards compatibility.

- any malformed input is constrained to remain a local undefined node to the best of our ability, i.e. a syntax error inside a pipeline will remain inside a single command, instead of corrupting the whole pipeline.

- The lexer has been modified to detect a left delimiter inside an already open action, which it didnt do before.
- The lexer response to ':' has been changed. Now if its not followed by '=' it simply returns the unicode char, instead of consuming the following character and producing an error.

- Fixed `FieldNode` position for chained field accesses (e.g. `.Address.Country`). Previously the node's `Pos` pointed to the second field in the chain rather than the first, because the `FieldNode` used to anchor at the next peeked token's position. The `FieldNode` is now anchored at the original node's position, so the resulting `FieldNode.Pos` correctly points to the leading `.` of the first field.

- Added end field to tree, signifying the position in file where the definition ends.

We aim to have these changes eventually merged into the upstream of go, then having this package locally will become obsolete.

## Usage

The package exposes two distinct parse modes.

### Normal mode (`Parse`)

Behaves identically to the upstream `text/template/parse` package: returns an error and an empty tree set on any syntactic mistake.

```go
treeSet, err := parse.Parse("myTemplate", text, "{{", "}}", funcMap)
if err != nil {
    // template had a syntax error; treeSet is empty
}
```

### Partial mode (`ParsePartial`)

Used by the language server so that an incomplete or syntactically broken template still produces a usable AST. Errors are collected in `Tree.Errors` rather than aborting the parse. Malformed nodes become `UndefinedNode` values localised to the smallest containing construct.

```go
t := parse.New("t")
t.Mode = parse.ParsePartial | parse.SkipFuncCheck | parse.ParseComments
treeSet := map[string]*parse.Tree{}
_, err := t.Parse(text, "{{", "}}", treeSet)
// err is non-nil only for truly unrecoverable failures.
// Partial errors are in t.Errors and as UndefinedNode leaves in the AST.
```

Flags that can be combined with `|`:

| Flag            | Effect                                                                                                           |
| --------------- | ---------------------------------------------------------------------------------------------------------------- |
| `ParseComments` | Include `{{/* … */}}` nodes in the tree (required for gotype hints)                                              |
| `SkipFuncCheck` | Do not validate that called functions are registered (required when the function map is not known at parse time) |
| `ParsePartial`  | Non-fatal error recovery; populate `Tree.Errors` instead of aborting                                             |

### `UndefinedNode`

An `UndefinedNode` in the tree signals a position where parsing failed but recovery was possible. The server's diagnostics handler walks the tree and converts these into LSP diagnostics. See [docs/features/diagnostics.md](../docs/features/diagnostics.md) for details.
