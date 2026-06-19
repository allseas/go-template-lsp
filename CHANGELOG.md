# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Integration test coverage for the custom funcmap (`//tmpl:func "global"`)
  feature. New shared cross-client testcases (grep prefix `Custom funcmap:`) in
  `test/testcases/{completion,definition,diagnostics}.json` exercise completion,
  go-to-definition and diagnostics for workspace-defined global template
  functions, backed by a new fixture
  `test/resources/templ-tests/funcs/global_funcs.go` (`gtmplShout`,
  `gtmplRepeatN`, `gtmplGreet`). The completion/definition cases carry a
  `gotype:` hint so the JetBrains client routes them through the heavy fixture
  (which loads the Go workspace); the diagnostics cases are consumed by the
  VSCode suite only (JetBrains has no shared diagnostics test pipeline).
  Note: the "no diagnostic" diagnostics case uses only *named-reference* global
  funcs (`gtmplShout`, `gtmplRepeatN`); inline-literal funcmap values (e.g.
  `gtmplGreet`) currently still raise an `undefined function` diagnostic in
  `analyse.go` even though they complete fine — pre-existing behaviour, left
  untouched here.
- Type-awareness diagnostics coverage for custom global funcs: the
  `Custom funcmap:` diagnostics cases now also assert that calling a typed
  named-reference func with too few arguments, a wrong argument type, or a
  nested call whose return type is incompatible each produces the expected
  `argumentNumberMismatch` / `invalidCommand` diagnostic (and that a correctly
  typed call stays clean).

### Changed

- JetBrains `DefinitionGotypeTest` now honours each testcase's `poll` flag,
  retrying go-to-declaration (and pumping the IDE event queue) until a
  cross-file target resolves. This mirrors the VSCode `pollDefinitions` helper
  and removes latent flakiness in the heavy-fixture definition tests where the
  first asynchronous LSP definition response could arrive after the assertion.

### Removed

### Fixed

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
