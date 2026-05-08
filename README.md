# GoTemplate Support Extension

This repository contains the source code of the *GoTemplate Support* extension. This extension provides IDE support for text/template language via a language server. It is created for VS Code and JetBrains IDEs.

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

## Development

### Requirements

- Go: 1.26.2+
- Node: 24.11.0+
- Npm: 11.12.0+
- gowatch (<https://github.com/silenceper/gowatch>)

### Server

#### Tests

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

### VS Code extension

#### Prerequisites

Install the npm packages:

```bash
npm i
cd clients/VSCode
npm i
cd ../..
```

#### Running the extension with watching

1. Run the watcher for server and extension source code:

    ```bash
    npm run watch:vscode
    ```

2. Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

3. For logs from the server, look into `Output` in the new VS Code window.

#### Running the extension with static builds

1. Build the binaries and compile the extension:

    ```bash
    npm run build:vscode
    ```

2. You also need to manually copy the server binaries from `/dist/server` to the `out` folder where the extension is compiled.

3. Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

4. For logs from the server, look into `Output` in the new VS Code window.

#### Packaging the extension

1. Build the binaries and package the extension:

    ```bash
    npm run package:vscode
    ```

2. Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

3. For logs from the server, look into `Output` in the new VS Code window.
