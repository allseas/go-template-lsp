# TextMate Grammar Generator

## Module Structure

```
syntax/
├── Grammar.hs      -- ADTs and constants (the specification)
|                   -- Separated to be easy to verify by hand whether it matches the go text/template documentation
├── TextMate.hs     -- TmPattern type, type aliases, JSON serialization
|                   -- ensures correct syntax of output
└── Generate.hs     -- Pattern generation (Grammar → TextMate), entry point
```

**Grammar.hs** — Sum types (`TemplateNode`, `ActionBody`, `LoopAction`, `Term`,
`VariableOp`) and constants (`keywords`, `builtinFunctions`). All types derive
`Enum`/`Bounded` for enumeration via `[minBound .. maxBound]`.

**TextMate.hs** — `TmPattern` ADT with type aliases (`ScopeName`, `Regex`,
`Capture`, `RepoKey`, `Named`) and JSON serialization. Language-agnostic.

**Generate.hs** — Total functions mapping each grammar constructor to TextMate
patterns. `allEntries` enumerates every constructor and assembles the repository.

**Deduplication** — `dedup` keeps the first occurrence per key. Ordering in
`allEntries` determines priority.

## Running

```sh
runghc -isrc/grammar Generate.hs
```

## Limitations

TextMate grammars are regular. They cannot express arbitrary nesting depth,
context-sensitive constraints, or semantic resolution. The generator approximates
these where needed.