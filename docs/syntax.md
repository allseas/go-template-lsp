# TextMate Grammar Generator

**Choice of Haskell** - the language was chosen for this due to having types as first class members, making it possible to specify the grammar as a typed object, and then iterate over all the types defined. This allows step by step verification for:
- Grammar definition
- Regex generation for each element
- Serialization into the proper format
In place of verifying a manually written json file all at once.

## Module Structure

```
syntax/
├── Grammar.hs      -- ADTs and constants (the specification)
|                   -- Separated to be easy to verify by hand whether it matches the go text/template documentation
├── TextMate.hs     -- TmPattern type, type aliases, JSON serialization
|                   -- ensures correct syntax of output
├── Generate.hs     -- Pattern generation (Grammar → TextMate), entry point
|                   -- ensures correct syntax of output
└── Regex.hs        -- regex constants for syntax elements of go template
```

**Grammar.hs** — Sum types (`TemplateNode`, `ActionBody`, `LoopAction`, `Term`,
`VariableOp`) and constants (`keywords`, `builtinFunctions`). All types derive
`Enum`/`Bounded` for enumeration via `[minBound .. maxBound]`.

**TextMate.hs** — `TmPattern` and `TmSyntax` types with type aliases for fields (`ScopeName`, `Regex`,
`Capture`, `RepoKey`, `Named`) and JSON serialization using aeson toJson instances. Language-agnostic.

**Generate.hs** — Total functions mapping each grammar constructor to TextMate
patterns. `allEntries` enumerates every constructor and assembles the repository.

**Regex.hs** — regex constants, specifying the elements of go template syntax

**Deduplication** — `dedup` keeps the first occurrence per key. Ordering in
`allEntries` determines priority.

## Running
in /syntax/
```sh
cabal run
```
or in the repo root

```sh
npm run generate:syntax
```
to automatically format the output json and copy it into the client extensions
## Limitations

TextMate grammars are regular. They cannot express arbitrary nesting depth,
context-sensitive constraints, or semantic resolution. The generator approximates
these where needed.