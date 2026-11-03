# Change Log

All notable changes to the `gotmpls` extension will be documented in this file.

Check [Keep a Changelog](http://keepachangelog.com/) for recommendations on how to structure this file.

## [1.3.0] - 2026-11-03

### Added

- `gotype` hints accept any Go type expression: slices, arrays, maps, pointers and nesting.
- `gotype` hints can instantiate generic types, e.g. `pkg.View[Instance]`.
- `map{...}` hints can be nested inside other types and used as generic type arguments.
- Go to definition resolves fields and methods across packages when a hint spans several packages.
- Go to definition on a type inside a `gotype` comment jumps to its declaration.
- Dict arguments to `{{ template }}` are checked key by key (`missingTemplateArgField`,
  `templateArgFieldMismatch`).
- A template without a `gotype` hint infers its input type from its first call site.
- The results of `index` and `dict` are typed, and `.Key` access on Go maps resolves.
- Marketplace metadata: icon, keywords, homepage and issue links.

### Changed

- The extension is now distributed under the Allseas End User License Agreement
  (EULA) instead of the MIT License. Bundled third-party components remain under
  their own licenses; see `THIRD_PARTY_NOTICES.md` in the package.
- Invalid `gotype:` comments (e.g. `gotype: 123`) are reported as `malformedHint`.
- Hints that differ only in key order or spacing no longer trigger `conflictingHint`.

### Removed

- The `invalidWith` diagnostic: `{{ with }}` accepts any value.

### Fixed

- No false no-such-field diagnostic when a field chain passes through an `any`/`nil` value.
- Go to definition on types from transitively imported packages.
- A root `gotype` hint in one file leaked into other files.
- Semantic highlighting of fields, methods and variable segments no longer includes the
  leading `.`.
- Fields chained after a parenthesised term (e.g. `(dict "a" .Address).a.City`,
  `(index .Items 0).Name`) are highlighted at the correct positions.
- Hover and go to definition work on fields chained after a parenthesised term.

## [1.2.0] - 2026-07-02

### Added

- Go to definition on `{{ template "name" }}` calls jumps to the matching `{{ define }}`.
- Refactor rename for variables and functions.
- Configuration option for a custom language server binary path.
- Hover on user-defined functions shows the godoc-style comment.
- Hover on functions shows input and output types.
- `map[string]any` type hints, with completions, hover, go-to-definition and diagnostics.
- Type hints can be placed anywhere in the template.
- Support for more dynamically built custom func-maps.
- New diagnostic for invalid map/dict keys.

### Changed

- User-facing dict hint syntax renamed to `map`.
- Diagnostics on `with` and `template` reworked: removed redundant ones and added more accurate ones.
- Variable reassignment no longer changes the original variable's type.
- Operations on `any`-typed values produce warnings instead of errors.
- Extension renamed to `gotmpls`.

### Fixed

- Race condition when loading type hints.
- Package multi-loading comparison issue.
- Pointer template arguments now work correctly.
- Loading functions from variables no longer panics.
- Completions when using `any` instead of `nil` are handled correctly.

## [1.1.0] - 2026-06-21

Initial published release.
