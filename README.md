# GoTemplate Support Extension <!-- omit in toc -->

This repository contains the source code of the *GoTemplate Support* extension. This extension provides IDE support for text/template language via a language server. It is created for VS Code and JetBrains IDEs.

## Table of Contents <!-- omit in toc -->

- [Features](#features)
- [Repository Structure](#repository-structure)
  - [Documentation](#documentation)
- [Development](#development)
  - [Requirements](#requirements)
  - [Language Server](#language-server)
    - [Formatting and Linting](#formatting-and-linting)
    - [Building the Binaries](#building-the-binaries)
    - [Server Tests](#server-tests)
  - [Syntax Generation](#syntax-generation)
  - [VS Code Extension](#vs-code-extension)
    - [Prerequisites for VS Code](#prerequisites-for-vs-code)
    - [Formatting and Linting the Extension](#formatting-and-linting-the-extension)
    - [Extension Tests](#extension-tests)
    - [Running the Extension with Watching](#running-the-extension-with-watching)
    - [Running the Extension with Static Builds](#running-the-extension-with-static-builds)
    - [Packaging the Extension](#packaging-the-extension)
  - [JetBrains Plugin](#jetbrains-plugin)
    - [Formatting and Linting the Plugin](#formatting-and-linting-the-plugin)
    - [Plugin Tests](#plugin-tests)
    - [Running the Plugin for Testing](#running-the-plugin-for-testing)
    - [Building the Plugin](#building-the-plugin)
- [Architecture Overview](#architecture-overview)
  - [Why This Architecture?](#why-this-architecture)
- [Development Workflow](#development-workflow)
  - [Adding New Features](#adding-new-features)
- [Resources](#resources)

## Features

Extension features:

| Feature                                         | VS Code | JetBrains | Priority |
| ----------------------------------------------- | ------- | --------- | -------- |
| Static syntax highlighting                      | ✅       | ✅         | Must     |
| Dynamic syntax highlighting                     |         |           | Must     |
| Autocompletion on variables                     | ✅       | ✅         | Must     |
| Autocompletion on field names                   | ✅       | ✅         | Must     |
| Autocompletion on chained field accesses        | ✅       | ✅         | Must     |
| Autocompletion on global functions              | ✅       | ✅         | Must     |
| Autocompletion on local functions               |         |           | Must     |
| Inspection on duplicate variable names          | ✅       | ✅         | Must     |
| Jump to definition                              | ✅       | ✅         | Must     |
| Definition on hover                             | ✅       | ✅         | Must     |
| Toggling features                               |         |           | Must     |
| Find usages of a variable or function           | ✅       | ✅         | Must     |
| Type checking on template block                 |         |           | Must     |
| Type checking on function                       |         |           | Should   |
| Wrap selection in a comment                     | ✅       | ✅         | Should   |
| Highlight end of the current block              |         |           | Should   |
| Label functions as deprecated                   |         |           | Could    |
| Static syntax highlighting for target language  |         |           | Could    |
| Auto-closing of opened tags (snippets)          | ✅       | ✅         | Could    |
| Wrap selection in a block                       | ✅       | ✅         | Should   |
| Inspection for missing whitespace trims in tags |         |           | Could    |

Language server features:

| Feature                                                  | Supported | Priority |
| -------------------------------------------------------- | --------- | -------- |
| Locally defined syntax modifications                     |           | Should   |
| Completions and types of functions inferred from project |           | Should   |
| Support for adding more global functions and tag types   |           | Should   |
| Code formatting of the template part                     |           | Could    |
| Code completions and syntax highlighting for Helm        |           | Could    |
| Code completions and syntax highlighting for Hugo        |           | Could    |
| User defined inspections and an ignore comment           |           | Could    |

More can be read about the features in [docs/features.md](docs/features/README.md)

## Repository Structure

This repository is organized as follows:

```files
gotemplate-lsp/
├── server/                    # Language server (Go)
│   ├── main.go               # Entry point
│   ├── handlers/             # LSP request handlers
│   └── go.mod                # Go dependencies
├── clients/
│   ├── VSCode/               # VS Code extension (TypeScript)
│   │   ├── src/              # Extension source
│   │   ├── test/             # Extension tests
│   │   ├── package.json      # Extension manifest
│   │   └── syntaxes/         # TextMate grammar
│   └── JetBrains/            # JetBrains plugin (Kotlin)
│       └── go-text-template/ # Plugin source (Gradle project)
│           ├── src/
│           │   ├── main/
│           │   │   ├── kotlin/      # Kotlin source code
│           │   │   └── resources/
│           │   │       └── META-INF/
│           │   │           └── plugin.xml   # Plugin manifest
│           │   └── test/            # Unit tests
│           ├── build.gradle.kts     # Gradle build config
│           └── gradle/              # Gradle wrapper
├── scripts/                  # Build scripts (TypeScript)
├── syntax/                   # TextMate grammar generator (Haskell)
│   ├── Grammar.hs            # Grammar specification for Go templates
│   ├── TextMate.hs           # TextMate pattern types and JSON serialization
│   ├── Generate.hs           # Pattern generation (entry point)
│   └── Generate.hs           # Regex constants
├── docs/                     # Documentation
│   ├── features.md           # Feature overview and roadmap
│   ├── server.md             # Language server architecture
│   ├── vscode-extension.md   # VS Code extension development
│   ├── jetbrains-plugin.md   # JetBrains plugin development
│   └── example-*.md          # Detailed examples
└── test/resources/           # Test templates and resources
```

### Documentation

Comprehensive documentation is available in the `docs/` directory:

- **[features.md](docs/features/README.md)** — Complete feature matrix, design decisions, and contribution guide
- **[server.md](docs/server.md)** — Language server architecture and guide for adding new features
- **[vscode-extension.md](docs/vscode/README.md)** — VS Code extension development guide with examples
- **[jetbrains-plugin.md](docs/jetbrains/README.md)** — JetBrains plugin development guide with examples

## Development

### Requirements

- **Go:** 1.26.2 or later
- **Node.js:** 24.11.0 or later
- **npm:** 11.12.0 or later
- **Java:** 21 or later (for JetBrains plugin development)
- **gowatch:** For watch mode (`go install github.com/silenceper/gowatch@latest`)
- **GHC** 8.8.4 or later (for generating the tmLanguage syntax)

**Optional:**

- golangci-lint (for server linting)
- Gradle (auto-installed by gradlew for JetBrains plugin)

### Language Server

The language server is written in Go, using [glsp](https://github.com/tliron/glsp) for the LSP protocol.

#### Formatting and Linting

For server linting and formatting, [golangci-lint](https://golangci-lint.run) is used. The detailed configuration (with linters and formatters that are enabled) can be found in `server/.golangci.yml`. To run the checks and formatters run:

```bash
golangci-lint fmt
golangci-lint run
```

#### Building the Binaries

The easiest way to build binaries for all targets is via `npm` in the root directory:

```bash
npm run build:server
```

This script will build all the binaries to `/dist/server`.

#### Server Tests

To run the tests you can use `go test`:

```bash
cd server
go test ./...
```

To run them with calculated code coverage:

```bash
cd server
go test ./... --coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Syntax Generation

The TextMate grammar for syntax highlighting is generated from a formal Go template grammar specification written in Haskell. To regenerate the grammar (GHC required):

```bash
cd syntax
cabal run
```

This outputs `syntax/syntaxes/gotemplate.tmLanguage.json`, which is used by both VS Code and JetBrains.

or:

```bash
npm run generate:syntax
```

which automatically formats the file and copies it into both extensions.

### VS Code Extension

#### Prerequisites for VS Code

Install the npm packages:

```bash
npm i
cd clients/VSCode
npm i
cd ../..
```

#### Formatting and Linting the Extension

We use prettier and eslint for ensuring code quality. You can run the formatter with:

```bash
cd clients/VSCode
npm run format
```

To check linting you can run:

```bash
cd clients/VSCode
npm run lint:check
# npm run lint:fix
```

#### Extension Tests

Run all extension tests:

```bash
cd clients/VSCode
npm run test
```

This uses `@vscode/test-cli` to run tests in a headless VS Code instance. Tests are located in `src/test/` and should follow the naming pattern `*.test.ts`.

For detailed testing information, see [vscode-testing.md](docs/vscode/vscode-testing.md) and [vscode-extension.md](docs/vscode/README.md#testing).

#### Running the Extension with Watching

This is useful for development, as the server binaries and the extension will recompile on any source code change.

1. Run the watcher for server and extension source code:

    ```bash
    npm run watch:vscode
    ```

2. Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

3. For logs from the server, look into `Output` in the new VS Code window.

#### Running the Extension with Static Builds

1. Build the binaries and compile the extension:

    ```bash
    npm run build:vscode
    ```

2. You also need to manually copy the server binaries from `/dist/server` to the `out` folder where the extension is compiled.

3. Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

4. For logs from the server, look into `Output` in the new VS Code window.

#### Packaging the Extension

This script will build the server to `/dist/server` and package the extension to `clients/VSCode/dist`. (The binaries will automatically be copied to the extension).

```bash
npm run package:vscode
```

### JetBrains Plugin

#### Formatting and Linting the Plugin

For formatting and lining we use ktlint. You can run it with:

```bash
cd clients/JetBrains/go-text-template/
./gradlew ktLintFormat
./gradlew ktLintCheck
```

#### Plugin Tests

To run the tests:

```bash
cd clients/JetBrains/go-text-template/
./gradlew test
```

#### Running the Plugin for Testing

This will run an IntelliJ IDEA instance for testing purposes. It also builds the server binaries.

```bash
cd clients/JetBrains/go-text-template/
./gradlew runIde
```

#### Building the Plugin

Build the plugin distribution:

```bash
cd clients/JetBrains/go-text-template/
./gradlew build
```

This creates a distributable plugin ZIP file in `build/distributions/`.

To build a signed plugin for publishing to the JetBrains Marketplace:

```bash
./gradlew signPlugin
```

To publish directly to the JetBrains Marketplace (requires authentication token):

```bash
./gradlew publishPlugin
```

For more detailed information on plugin development, see [jetbrains-plugin.md](docs/jetbrains/README.md).

## Architecture Overview

This project uses a **client-server architecture** with a shared language server that communicates using LSP via stdio.

### Why This Architecture?

- **Single Server, Multiple Clients** — The Language Server Protocol allows a single Go backend to serve VS Code, JetBrains, and potentially other IDEs without code duplication
- **TypeScript for VS Code** — Uses standard VS Code extension APIs for seamless integration
- **Kotlin for JetBrains** — Follows JetBrains' modern plugin development standards

For detailed architecture information, see [server.md](docs/server.md).

## Development Workflow

### Adding New Features

1. **Add feature to roadmap** in [docs/features.md](docs/features/README.md)
2. **Implement in server** — Add LSP handler in `server/handlers/`
3. **Add to VS Code** — Update `clients/VSCode/` if needed
4. **Add to JetBrains** — Update `clients/JetBrains/` if needed
5. **Add tests** — Write tests for new functionality
6. **Update docs** — Document the new feature and any APIs

Each component has its own development guide:

- [server.md](docs/server.md) — How to add server features
- [vscode-extension.md](docs/vscode/README.md) — How to add VS Code features
- [jetbrains-plugin.md](docs/jetbrains/README.md) — How to add JetBrains features

## Resources

- [Language Server Protocol Spec](https://microsoft.github.io/language-server-protocol/)
- [Go text/template Package](https://pkg.go.dev/text/template)
- [VS Code Extension API](https://code.visualstudio.com/api)
- [JetBrains Plugin Development](https://plugins.jetbrains.com/docs/intellij/)
