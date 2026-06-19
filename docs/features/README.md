# Features

This document provides an overview of all features implemented and planned for the GoTemplate LSP project.

## Feature Status

### Legend

- ✅ **Implemented** - Feature is fully functional
- 🚧 **In Progress** - Feature is currently being developed
- ⏳ **Planned** - Feature is scheduled for development
- ❓ **Proposed** - Feature under consideration

## Editor Features

| Feature                                | VS Code (tested) | JetBrains (tested) | Priority | Notes                                                                     |
| -------------------------------------- |------------------|--------------------| -------- | ------------------------------------------------------------------------- |
| **Syntax Highlighting**                |                  |                    |          |                                                                           |
| Static syntax highlighting             | ✅                | ✅                  | Must     | Syntax defined in `.tmpl` files                                           |
| Dynamic syntax highlighting            | ✅ (⏳)            | ✅ (⏳)              | Must     | Semantic tokens for variables, fields, functions, keywords                |
| Target language syntax                 | ❓ (❓)            | ❓ (❓)              | Could    | Syntax highlighting for embedded languages (SQL, HTML, etc.)              |
| **Code Completion**                    |                  |                    |          |                                                                           |
| Completion on template variables       | ✅ (✅)            | ✅ (✅)              | Must     | Suggests available variables in current scope                             |
| Completion on struct field names       | ✅ (✅)            | ✅ (✅)              | Must     | Auto-complete struct field access and chained field accesses              |
| Completion on built-in functions       | ✅ (⏳)            | ✅ (⏳)              | Must     | Suggests standard template functions                                      |
| Completion on local functions          | ✅ (⏳)            | ✅ (⏳)              | Must     | Suggests user-defined template functions via `//tmpl:func "global"` hints |
| **Navigation**                         |                  |                    |          |                                                                           |
| Jump to definition                     | ✅ (✅)            | ✅ (✅)              | Must     | Go to variable, field, or function definition                             |
| Find references                        | ✅ (✅)            | ✅ (✅)              | Must     | Find all usages of a variable or identifier in the current file           |
| Hover tooltips                         | ✅ (✅)            | ✅ (✅)              | Must     | Contextual documentation for every node type                              |
| **Inspections & Diagnostics**          |                  |                    |          |                                                                           |
| Syntax errors                          | ✅ (✅)            | ✅ (✅)              | Must     | Warn about incorrect syntax                                               |
| Duplicate variable detection           | ✅ (✅)            | ✅ (✅)              | Must     | Warn about redefined variables                                            |
| Unknown function detection             | ✅ (✅)            | ✅ (✅)              | Must     | Warn about calls to unregistered functions                                |
| Template type checking                 | ✅ (⏳)            | ✅ (⏳)              | Must     | Validate argument types on `{{template}}` calls                           |
| Function argument type checking        | ✅ (⏳)            | ✅ (⏳)              | Should   | Validate argument types on function calls                                 |
| Unused variable detection              | ⏳ (⏳)            | ⏳ (⏳)              | Should   | Flag declared but unused variables                                        |
| Missing whitespace trim detection      | ⏳ (⏳)            | ⏳ (⏳)              | Could    | Suggest whitespace trim operators when needed                             |
| **Type & Function Hints**              |                  |                    |          |                                                                           |
| Type hints (`/*gotype: pkg.Type*/`)    | ✅ (✅)            | ✅ (✅)              | Must     | Resolve dot type from Go source for completions, hover, and definition    |
| Custom function hints (`//tmpl:func`)  | ✅ (✅)            | ✅ (✅)              | Must     | Register user-defined `FuncMap` functions; hot-reloaded on `.go` save     |
| **Code Actions & Refactoring**         |                  |                    |          |                                                                           |
| Wrap selection in comment              | ✅ (⏳)            | ✅ (⏳)              | Should   | `{{- /* ... */ -}}`                                                       |
| Wrap selection in a block              | ✅ (⏳)            | ✅ (⏳)              | Should   | `{{- if ... }} ... {{- end }}`                                            |
| **Snippets**                           |                  |                    |          |                                                                           |
| Built-in snippets                      | ✅ (✅)            | ✅ (✅)              | Could    | Common template patterns                                                  |
| **Configuration**                      |                  |                    |          |                                                                           |
| Project config (`gotmpl.config.json`)  | ✅ (✅)            | ✅ (✅)              | Must     | Per-project settings; takes precedence over IDE settings                  |
| IDE-level settings                     | ✅ (✅)            | ✅ (✅)              | Should   | VS Code settings / JetBrains plugin settings                              |
| Per-file configuration comments        | ⏳ (⏳)            | ⏳ (⏳)              | Could    | `// @gotemplate disable-inspection-name`                                  |
| **Other**                              |                  |                    |          |                                                                           |
| Block boundary highlighting            | ⏳ (⏳)            | ⏳ (⏳)              | Should   | Highlight matching `{{- end }}` tags                                      |

## Language Server Features

The language server provides the backend for all editor features via LSP.

| Feature                                | Supported (tested) | Priority | Notes                                                                   |
| -------------------------------------- | ------------------ | -------- | ----------------------------------------------------------------------- |
| **Configuration**                      |                    |          |                                                                         |
| User configuration                     | ✅ (✅)              | Should   | Per-user IDE settings                                                   |
| Project-level settings                 | ✅ (✅)              | Must     | Per-project configuration via `gotmpl.config.json`                      |
| Per-file configuration comments        | ⏳ (⏳)              | Could    | `// @gotemplate disable-inspection-name`                                |
| **Analysis**                           |                    |          |                                                                         |
| Local syntax modifications             | ⏳ (⏳)              | Should   | Support custom delimiters or syntax extensions                          |
| Function inference from project        | ✅ (✅)              | Should   | Detect custom template functions from source via `//tmpl:func "global"` |
| Type inference                         | ⏳ (⏳)              | Should   | Infer types passed to templates without explicit hints                  |
| **Extensibility**                      |                    |          |                                                                         |
| Custom function registration           | ✅ (✅)              | Should   | Register user-defined functions via `//tmpl:func "global"` annotations  |
| Custom tag type support                | ⏳ (⏳)              | Should   | Support for custom action tag types                                     |
| **Formatting & Linting**               |                    |          |                                                                         |
| Template code formatting               | ⏳ (⏳)              | Could    | Format template expressions                                             |
| Template linting rules                 | ⏳ (⏳)              | Could    | Configurable linting rules                                              |

## Specialized Support

| Feature                 | Supported | Status  | Priority | Notes                                       |
| ----------------------- | --------- | ------- | -------- | ------------------------------------------- |
| **Helm Templates**      |           |         |          |                                             |
| Helm syntax support     | ⏳         | Planned | Could    | Special handling for Helm `.yaml.tpl` files |
| Helm built-in functions | ⏳         | Planned | Could    | Auto-complete and documentation             |
| **Hugo Templates**      |           |         |          |                                             |
| Hugo syntax support     | ⏳         | Planned | Could    | Support for Hugo-specific syntax            |
| Hugo built-in functions | ⏳         | Planned | Could    | Auto-complete and documentation             |

## Background

### Why LSP?

Using LSP means a single Go backend serves VS Code, JetBrains, and any other LSP-compatible editor. Adding support for a new editor only requires a thin client wrapper.

### Server in Go

The language server is written in Go so it can reuse the `text/template` parser and `go/types` type checker from the standard library directly, without reimplementing them.

## Feature Documentation

For detailed documentation on each feature, see:

- [Syntax Highlighting](syntax.md) - Static syntax highlighting for Go templates
- [Completions](completions.md) - Code completion for variables, fields, and functions
- [Definition](definition.md) - Jump to definition for symbols
- [References](references.md) - Find all usages of a symbol
- [Hover](hover.md) - Hover information for symbols
- [Diagnostics](diagnostics.md) - Error reporting and validation
- [Type Hints](type_hints.md) - Declaring template input types via `/*gotype:*/` comments
- [Function Hints](func_hints.md) - Registering custom template functions
- [Type Checking](template_checking.md) - Validating template call arguments against declared types
