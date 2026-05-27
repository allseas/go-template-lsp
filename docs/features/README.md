# Features

This document provides an overview of all features implemented and planned for the GoTemplate LSP project.

## Overview

The GoTemplate LSP provides IDE support for Go's `text/template` language across VS Code and JetBrains IDEs. Features are organized by category and priority level.

## Feature Status

### Legend

- ✅ **Implemented** — Feature is fully functional
- 🚧 **In Progress** — Feature is currently being developed
- ⏳ **Planned** — Feature is scheduled for development
- ❓ **Proposed** — Feature under consideration

## Editor Features

| Feature                           | VS Code | JetBrains | Priority | Notes                                                        |
|-----------------------------------|---------|-----------|----------|--------------------------------------------------------------|
| **Syntax Highlighting**           |         |           |          |
| Static syntax highlighting        | ✅       | ✅         | Must     | Syntax defined in `.tmpl` files                              |
| Dynamic syntax highlighting       | ⏳       | ⏳         | Must     | Highlighting based on variables and context                  |
| Target language syntax            | ❓       | ❓         | Could    | Syntax highlighting for embedded languages (SQL, HTML, etc.) |
| **Code Completion**               |         |           |          |
| Completion on template variables  | ✅       | ✅         | Must     | Suggests available variables in current scope                |
| Completion on struct field names  | ⏳       | ⏳         | Must     | Auto-complete struct field access                            |
| Completion on built-in functions  | ✅       | ✅         | Must     | Suggests standard template functions                         |
| Completion on local functions     | ⏳       | ⏳         | Must     | Suggests user-defined template functions                     |
| **Navigation**                    |         |           |          |
| Jump to definition                | ✅       | ✅         | Must     | Go to variable or function definition                        |
| Find references / Usages          | ✅       | ✅         | Must     | Find all usages of a symbol                                  |
| Peek definition on hover          | ⏳       | ⏳         | Must     | Show definition in hover tooltip                             |
| **Inspections & Diagnostics**     |         |           |          |
| Incorrect syntax                  | ⏳       | ⏳         | Must     | Warn about incorrect syntax                                  |
| Duplicate variable detection      | ⏳       | ⏳         | Must     | Warn about redefined variables                               |
| Type checking                     | ⏳       | ⏳         | Should   | Validate type compatibility in templates                     |
| Unused variable detection         | ⏳       | ⏳         | Should   | Flag declared but unused variables                           |
| Missing whitespace trim detection | ⏳       | ⏳         | Could    | Suggest whitespace trim operators when needed                |
| **Code Actions & Refactoring**    |         |           |          |
| Wrap selection in comment         | ✅       | ✅         | Should   | `{{- /* ... */ -}}`                                          |
| Wrap selection in a block         | ✅       | ✅         | Should   | `{{- if ... }} ... {{- end }}`                               |
| **Snippets**                      |         |           |          |
| Built-in snippets                 | ✅       | ✅         | Could    | Common template patterns                                     |
| **Other**                         |         |           |          |
| Block boundary highlighting       | ⏳       | ⏳         | Should   | Highlight matching `{{- end }}` tags                         |
| Feature toggle support            | ⏳       | ⏳         | Must     | Enable/disable features per file or project                  |

## Language Server Features

The language server provides the backend intelligence for all editor features. These features support multiple IDEs through the LSP protocol.

| Feature                         | Supported | Priority | Notes                                               |
|---------------------------------|-----------|----------|-----------------------------------------------------|
| **Configuration**               |           |          |
| User configuration              | ✅         | Should   | Per-user configuration, lower priority than project |
| Project-level settings          | ✅         | Must     | Per-project configuration override                  |
| Per-file configuration comments | ⏳         | Could    | `// @gotemplate disable-inspection-name`            |
| **Analysis**                    |           |          |
| Local syntax modifications      | ⏳         | Should   | Support custom delimiters or syntax extensions      |
| Function inference from project | ⏳         | Should   | Detect custom template functions from source        |
| Type inference                  | ⏳         | Should   | Infer types passed to templates                     |
| **Extensibility**               |           |          |
| Custom function registration    | ⏳         | Should   | Register user-defined functions with server         |
| Custom tag type support         | ⏳         | Should   | Support for custom action tag types                 |
| **Formatting & Linting**        |           |          |
| Template code formatting        | ⏳         | Could    | Format template expressions                         |
| Template linting rules          | ⏳         | Could    | Configurable linting rules                          |

## Specialized Support

| Feature                 | Supported | Status  | Priority | Notes                                       |
|-------------------------|-----------|---------|----------|---------------------------------------------|
| **Helm Templates**      |           |         |          |                                             |
| Helm syntax support     | ⏳         | Planned | Could    | Special handling for Helm `.yaml.tpl` files |
| Helm built-in functions | ⏳         | Planned | Could    | Auto-complete and documentation             |
| **Hugo Templates**      |           |         |          |                                             |
| Hugo syntax support     | ⏳         | Planned | Could    | Support for Hugo-specific syntax            |
| Hugo built-in functions | ⏳         | Planned | Could    | Auto-complete and documentation             |

## Design Decisions

### Why LSP?

The Language Server Protocol (LSP) was chosen to provide a single backend implementation that works across multiple IDEs (VS Code, JetBrains, Neovim, etc.). This reduces code duplication and allows all editors to benefit from improvements to the core server.

### Server in Go

The language server is implemented in Go to make use of the parser already present in the Go library and the type checker.
