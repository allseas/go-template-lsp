# Change Log

All notable changes to the `gotmpls` extension will be documented in this file.

Check [Keep a Changelog](http://keepachangelog.com/) for recommendations on how to structure this file.

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
