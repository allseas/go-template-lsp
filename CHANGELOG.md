# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

### Removed

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
