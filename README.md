# GoTemplate Support <!-- omit in toc -->

IDE support for Go's `text/template` language via a language server. Works with VS Code and JetBrains IDEs.

This project was started as a 10-week student project at Allseas by a group of 5 second-year BSc Computer Science and Engineering students at TU Delft.

## Table of Contents <!-- omit in toc -->

- [Features](#features)
- [User Testimonies](#user-testimonies)
- [Repository Structure](#repository-structure)
- [Development](#development)
  - [Requirements](#requirements)
  - [Language Server](#language-server)
  - [Syntax Generation](#syntax-generation)
  - [VS Code Extension](#vs-code-extension)
  - [JetBrains Plugin](#jetbrains-plugin)
- [Resources](#resources)
- [Third-Party Notices](#third-party-notices)

## Features

| Feature                                                  | VS Code | JetBrains | Priority |
| -------------------------------------------------------- | ------- | --------- | -------- |
| Static syntax highlighting                               | ✅       | ✅         | Must     |
| Dynamic syntax highlighting                              | ✅       | ✅         | Must     |
| Autocompletion on variables                              | ✅       | ✅         | Must     |
| Autocompletion on field names                            | ✅       | ✅         | Must     |
| Autocompletion on chained field accesses                 | ✅       | ✅         | Must     |
| Autocompletion on global functions                       | ✅       | ✅         | Must     |
| Autocompletion on local functions                        | ✅       | ✅         | Must     |
| Hover tooltips                                           | ✅       | ✅         | Must     |
| Jump to definition                                       | ✅       | ✅         | Must     |
| Find references                                          | ✅       | ✅         | Must     |
| Diagnostics (syntax, undeclared vars, unknown functions) | ✅       | ✅         | Must     |
| Type hints (`/*gotype: pkg.Type*/`)                      | ✅       | ✅         | Must     |
| Custom function hints (`//tmpl:func`)                    | ✅       | ✅         | Must     |
| Template type checking                                   | ✅       | ✅         | Must     |
| Configuration (project + IDE)                            | ✅       | ✅         | Must     |
| Snippets                                                 | ✅       | ✅         | Could    |
| Wrap selection in a comment                              | ✅       | ✅         | Should   |
| Wrap selection in a block                                | ✅       | ✅         | Should   |
| Type checking on function arguments                      | ⏳       | ⏳         | Should   |
| Highlight matching `{{end}}`                             | ⏳       | ⏳         | Should   |
| Unused variable detection                                | ⏳       | ⏳         | Should   |
| Missing whitespace trim detection                        | ⏳       | ⏳         | Could    |
| Syntax highlighting for embedded language                | ❓       | ❓         | Could    |

For the full feature roadmap, see [docs/features/README.md](docs/features/README.md).
For an end-user guide, see [docs/usage.md](docs/usage.md).

## User Testimonies

The following quotes are from the users of our extension after seeing the demonstration.

> 5/5. Looks comprehensive and addresses the major points of concern when using Go templates.

> 5/5. It seems very complete and like it would improve my life.

> 🔥!

## Repository Structure

```bash
gotemplate-lsp/
├── server/          # Language server (Go)
│   └── handlers/   # LSP request handlers
├── clients/
│   ├── VSCode/     # VS Code extension (TypeScript)
│   └── JetBrains/ # JetBrains plugin (Kotlin)
├── syntax/          # TextMate grammar generator (Haskell)
├── scripts/         # Build scripts
├── docs/            # Documentation
└── test/resources/  # Test fixtures
```

## Development

### Requirements

| Tool          | Version  | Purpose                                                        |
| ------------- | -------- | -------------------------------------------------------------- |
| Go            | 1.26.2+  | Language server                                                |
| Node.js       | 24.11.0+ | Build scripts and VS Code extension                            |
| npm           | 11.12.0+ | Package management                                             |
| Java          | 21+      | JetBrains plugin                                               |
| GHC           | 8.8.4+   | Regenerating the TextMate grammar                              |
| gowatch       | latest   | Watch mode (`go install github.com/silenceper/gowatch@latest`) |
| golangci-lint | latest   | Server linting (optional)                                      |

### Language Server

The language server is written in Go using [glsp](https://github.com/tliron/glsp).

```bash
# Lint and format
golangci-lint fmt && golangci-lint run

# Build binaries for all targets (output: dist/server/)
npm run build:server

# Tests
cd server && go test ./...

# Tests with coverage
cd server && go test ./... --coverprofile=coverage.out && go tool cover -func=coverage.out
```

See [docs/server.md](docs/server.md) for the server architecture.

### Syntax Generation

The TextMate grammar is generated from a Haskell specification. Output goes to `syntax/syntaxes/gotemplate.tmLanguage.json` and is shared by both clients.

```bash
npm run generate:syntax   # build + format + copy to both extensions
```

### VS Code Extension

```bash
# Install dependencies
npm i && cd clients/VSCode && npm i && cd ../..

# Format and lint
cd clients/VSCode && npm run format && npm run lint:check

# Tests
cd clients/VSCode && npm run test

# Watch mode (recompiles on change, then press F5 in clients/VSCode to launch)
npm run watch:vscode

# Package (builds server + bundles extension into clients/VSCode/dist/)
npm run package:vscode
```

See [docs/vscode/README.md](docs/vscode/README.md) for more detail.

### JetBrains Plugin

```bash
cd clients/JetBrains/go-text-template/

./gradlew ktLintFormat   # format
./gradlew ktLintCheck    # lint
./gradlew test           # tests
./gradlew runIde         # run a test IDE instance (also builds server binaries)
./gradlew build          # build distributable ZIP (build/distributions/)
```

See [docs/jetbrains/README.md](docs/jetbrains/README.md) for more detail.

## Resources

- [Language Server Protocol Spec](https://microsoft.github.io/language-server-protocol/)
- [Go text/template Package](https://pkg.go.dev/text/template)
- [VS Code Extension API](https://code.visualstudio.com/api)
- [JetBrains Plugin Development](https://plugins.jetbrains.com/docs/intellij/)

## Third-Party Notices

See [docs/RESOURCES.md](docs/RESOURCES.md) for third-party attributions and licenses.
