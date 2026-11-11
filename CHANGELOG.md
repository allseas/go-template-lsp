# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Completion after a pipe no longer crashes the server when a global FuncMap entry's
  signature cannot be resolved.

## [1.3.0] - 2026-11-03

### Added

- `gotmpls check` CLI subcommand for batch diagnostics on template files or globs,
  with text or JSON output, stdin input, `--root`, `--verbose` and `--min-severity`
  (filters the output and sets the exit-code threshold).
- Freestanding `gotmpls` and check-only `gotmpls-check` binaries, and a `cli` build
  tag that excludes the LSP server runtime.
- `gotype` hints accept any Go type expression: slices, arrays, maps, pointers and
  arbitrary nesting, with builtins in any position.
- `gotype` hints can instantiate generic types, e.g. `pkg.View[Instance]` or `pkg.Pair[A, B]`.
- `map{...}` hints can be nested inside other types and used as generic type arguments
  (`[]map{...}`, `map[string]map{...}`, `*map{...}`, `pkg.View[map{...}]`).
- Go to definition resolves fields and methods across packages when a hint spans
  several packages (generic type arguments, dict values).
- Go to definition on a type inside a `gotype` comment jumps to its declaration; on a
  bare `.` in a hinted template it jumps to the hint comment.
- Dict arguments to `{{ template }}` are checked key by key, with new
  `missingTemplateArgField` and `templateArgFieldMismatch` diagnostics. Dicts nested
  in other types are compared component by component.
- A template without a `gotype` hint infers its input type from its first call site.
- The results of the `index` and `dict` builtins are typed, and `.Key` access on Go
  maps resolves, so field access on them works.

### Changed

- The published VS Code extension and JetBrains plugin are now distributed to
  end users under the Allseas End User License Agreement (EULA). The source code
  in this repository remains MIT-licensed.
- A `gotype:` comment whose body is not a valid type expression (e.g. `gotype: 123`)
  is now reported as `malformedHint` instead of being silently ignored.
- Hints that differ only in key order or spacing no longer trigger `conflictingHint`.
- A pointer directly to a dict (`*map{...}`) is treated as the dict itself, matching
  text/template's pointer dereferencing.

### Removed

- The `invalidWith` diagnostic: text/template's `{{ with }}` accepts any value.

### Fixed

- No false no-such-field diagnostic when a field chain passes through an `any`/`nil` value.
- Go to definition returned a broken location for types from transitively imported packages.
- A root `gotype` hint in one file leaked into the root template of other files.
- CLI checks gave wrong results when run from a different workspace folder.
- `gotmpls check` exits with code 2 when a glob matches no files, instead of silently
  passing; files matched more than once are analysed once.
- Semantic tokens for fields, methods and variable segments cover only the identifier,
  not the leading `.`. In JetBrains, go to definition on a highlighted field now resolves
  (LSP4IJ resolves from the token's start offset); in VS Code the dot is no longer
  highlighted as part of the field.
- Fields chained after a parenthesised term (e.g. `(dict "a" .Address).a.City`,
  `(index .Items 0).Name`) get one correctly placed token per field instead of a single
  miscomputed range, so highlighting no longer drifts with extra spaces or parentheses.
- Hover and go to definition work on fields chained after a parenthesised term.

## [1.2.0] - 2026-07-02

### Added

- Go to definition on `{{ template "name" }}` calls jumps to the matching `{{ define }}`.
- Refactor rename for symbols.
- Configuration option for a custom language server binary path.
- Hover on user-defined functions shows the godoc-style comment.
- Hover on functions shows input and output types.
- `map[string]any` type hints, with completions, hover, go-to-definition and diagnostics.
- Type hints can be placed anywhere in the template.
- Support for more dynamically built custom func-maps.
- New diagnostic for invalid map/dict keys.

JetBrains:

- Autoclosing of comments and combined comment + trim deletion.

### Changed

- Diagnostics on `with` and `template` reworked: removed redundant ones and added more accurate ones.
- Variable reassignment no longer changes the original variable's type.
- Operations on `any`-typed values produce warnings instead of errors.
- Extension renamed to `gotmpls`.

### Fixed

- Package multi-loading comparison issue.
- Pointer template arguments now work correctly.
- Loading functions from variables no longer panics.
- Completions when using `any` are handled correctly.
- Variable reassignment bug.

JetBrains:

- Autoclosing `{{-` now works correctly.

## [1.1.0] - 2026-06-21

### Added

- Completions for field access can be set to `full` mode, showing the full path to the field with a matching type. The previous behaviour with only showing one field property at a time is still the default and is named `step` mode.
- Completions for user global functions are type-aware.
- Diagnostic error when using `with` on a non-struct.

### Changed

- Functions that are suggested are ordered, showing the concrete-typed ones first.

### Removed

- ErrorTypeInvalidIf, since almost any `if` is valid.

### Fixed

- Hover on field access is correct when paired with a variable.
- Root variable `$` is set to *any* type correctly.
- Removed duplicate diagnostic message on incorrect function.
- Third bracket used to appear sometimes in JetBrains when typing `{{}}`.
- `else` was incorrectly highlighted as a keyword, even when in plain text.
- Inline defined user global functions gave an `undefined functions` diagnostic.
- Template type checking would sometimes fail if the type was the same.

JetBrains:

- The IDE would give random suggestions when there were none from the language server.

## [1.0.0] - 2026-06-17

### Added

- Diagnostics on incorrect gotype hint.
- Iterating over a struct diagnostic.
- Go to definition of a user defined global function.
- Type checking for functions.
- Diagnostic of variable redeclaration in a range block.

### Changed

- Configuration for diagnostics can now change their severity and not only disable them.

### Fixed

- Funcmaps were not loaded when nested too much.
- Go to definition and hover inside an if/with block.
- Semantic tokens generation with multiple define blocks.
- DocumentSymbols failing with define with empty name.
- Field access on a variable node highlights and goes to definition correctly.
- Field hover definition gave incorrect dot context type.
- Incorrect comments are handled correctly by parser.

## [0.2.0] - 2026-06-12

### Added

Features:

- Inspections: on incorrect syntax, duplicate variable names.
- Go to Definition: on template variables and field accesses (goes to the Go source file).
- Configuration in a workspace file: config can be saved in `gotmpl.config.json`.
- Auto completions on chained field accesses.
- Semantic syntax highlighting.
- User defined global functions are read from the Go files.
- Type checking on `{{ template }}` block when the named template has a type hint.

Plugins:

- Released for JetBrains!

Repository:

- MIT license! The project becomes open source.
- A separate `Allseas` branch to support their parser changes.

### Changed

- Improved hover messages to be more informative and similar in style to Go.
- Templates with multiple `{{ define }}` are supported.
- Field/method suggestions are sorted by type.

### Fixed

- Static grammar rules are now correct. They are generated with Haskell.
- Comment syntax is now correct on auto toggling.
- Server would sometimes panic on syntactically incorrect templates.
- Auto-completions would create a double dot in some cases.

## [0.1.0] - 2026-05-21

### Added

- Syntax highlighting.
- Auto completions - template variables and built-in functions. The suggestions are scope aware.
- Find references/usages of a function or variable.
- Wrap selection in a comment.
- Wrap selection in a tag block.
- Snippets for tags.
- User/project configuration.
- Hover definitions.
- File type icon.
- [VS Code] Welcome message on first install.
