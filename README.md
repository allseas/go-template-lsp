# GoTemplate Support Extension <!-- omit in toc -->

This repository contains the source code of the *GoTemplate Support* extension. This extension provides IDE support for text/template language via a language server. It is created for VS Code and JetBrains IDEs.

## Table of Contents <!-- omit in toc -->

- [Features](#features)
- [Repository Structure](#repository-structure)
- [Development](#development)
  - [Requirements](#requirements)
  - [Language Server](#language-server)
    - [Formatting and Linting](#formatting-and-linting)
    - [Building the Binaries](#building-the-binaries)
    - [Server Tests](#server-tests)
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

## Features

Extension features:

| Feature                                         | VS Code | JetBrains | Priority |
| ----------------------------------------------- | ------- | --------- | -------- |
| Static syntax highlighting                      | ✅       | ✅         | Must     |
| Dynamic syntax highlighting                     |         |           | Must     |
| Autocompletion on variables                     | ✅       | ✅         | Must     |
| Autocompletion on field names                   |         |           | Must     |
| Autocompletion on global functions              | ✅       | ✅         | Must     |
| Autocompletion on local functions               |         |           | Must     |
| Inspection on duplicate variable names          |         |           | Must     |
| Jump to definition                              |         |           | Must     |
| Definition on hover                             |         |           | Must     |
| Toggling features                               |         |           | Must     |
| Find usages of a variable or function           | ✅       | ✅         | Must     |
| Type checking on template                       |         |           | Must     |
| Type checking on function                       |         |           | Should   |
| Wrap selection in a comment                     | ✅       | ✅         | Should   |
| Highlight end of the current block              |         |           | Should   |
| Label functions as deprecated                   |         |           | Could    |
| Static syntax highlighting for target langauge  |         |           | Could    |
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

More can be read about the features in [docs/features.md](docs/features.md)

## Repository Structure

This repository contains:

- source code of the text/tempalte language server in `server`
- source code of the modified text/template parser in `parse`
- source code of the JetBrains plugin in `clients/JetBrains`
- source code of the VS Code extension in `clients/VSCode`
- build scripts for the server in `scripts`
- resources for testing the extension in `test/resources`

## Development

### Requirements

- Go: 1.26.2+
- Node: 24.11.0+
- Npm: 11.12.0+
- gowatch (<https://github.com/silenceper/gowatch>)
- Java [TODO: WHICH VERSION? 21?]

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

[TODO: HOW TO RUN VS CODE TESTS]

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

This will run an Intellij IDE for testing purposes. It also builds the server binaries.

```bash
cd clients/JetBrains/go-text-template/
./gradlew runIde
```

#### Building the Plugin

[TODO: I HAVE NO CLUE ON HOW TO BUILD THE JETBRAINS PLUGIN]
