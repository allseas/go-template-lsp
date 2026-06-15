# Documentation

- [**Usage guide**](usage.md) - end-user guide: features, type hints, function hints, configuration
- [Features](features/README.md) - feature matrix and status for both clients
- [Server](server.md) - language server architecture and how to add new handlers
- [Types package](types.md) - typed parse tree, scope rules, and analysis flow
- [Syntax](syntax.md) - TextMate grammar generator (Haskell)
- [Configuration](configuration.md) - configuration options and hierarchy
  - [VS Code configuration](vscode/vscode-configuration.md)
  - [JetBrains configuration](jetbrains/jetbrains-configuration.md)
  - [Config file](features/config.md) - `gotmpl.config.json`
- [VS Code extension](vscode/README.md) - extension architecture and how to add new features
- [JetBrains plugin](jetbrains/README.md) - plugin architecture and how to add new features
- [Testing](testing.md)
  - [VS Code testing](vscode/vscode-testing.md)
  - [JetBrains testing](jetbrains/jetbrains-testing.md)

## Feature docs

- [Completions](features/completions.md) - AST-based completion logic
- [Hover](features/hover.md) - hover tooltips
- [Definition](features/definition.md) - go-to-definition
- [Diagnostics](features/diagnostics.md) - error reporting
- [Type hints](features/type_hints.md) - `gotype:` comments and type loading
- [Function hints](features/func_hints.md) - `//tmpl:func` annotations
- [Template checking](features/template_checking.md) - type checking on `{{template}}` calls
- [References](features/references.md) - find all references
