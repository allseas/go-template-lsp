<!-- omit in toc -->
# Usage Guide

This guide is for developers using the GoTemplate Support extension. It covers what the extension does, how to get the most out of its features, and how to configure it.

<!-- omit in toc -->
## Table of Contents

- [Installation](#installation)
  - [VS Code](#vs-code)
  - [JetBrains](#jetbrains)
- [Bug Reporting](#bug-reporting)
  - [Server Logs in VS Code](#server-logs-in-vs-code)
  - [Server Logs in JetBrains](#server-logs-in-jetbrains)
- [File Association](#file-association)
- [Features](#features)
  - [Syntax Highlighting](#syntax-highlighting)
  - [Completions](#completions)
  - [Hover Tooltips](#hover-tooltips)
  - [Go to Definition](#go-to-definition)
  - [Find References](#find-references)
  - [Diagnostics](#diagnostics)
  - [Snippets and Code Actions](#snippets-and-code-actions)
- [Type Hints](#type-hints)
  - [Basic Usage](#basic-usage)
  - [Multiple Define Blocks](#multiple-define-blocks)
  - [Template Type Checking](#template-type-checking)
- [Function Hints](#function-hints)
- [Configuration](#configuration)
  - [Project Config File](#project-config-file)
  - [VS Code Settings](#vs-code-settings)
  - [JetBrains Settings](#jetbrains-settings)
  - [Configuration Options Reference](#configuration-options-reference)

---

## Installation

### VS Code

1. Obtain the `.vsix` package from a project release (or build it yourself with `npm run package:vscode`).
2. Open the *Extensions* sidebar (<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>X</kbd>).
3. Click the `…` menu in the top-right corner of the sidebar and choose **Install from VSIX…**
4. Select the `.vsix` file and confirm. VS Code will activate the extension immediately.

### JetBrains

1. Obtain the `.zip` plugin archive from a project release (or build it yourself with `./gradlew build` inside `clients/JetBrains/go-text-template/`).
2. Open *Settings* (<kbd>Ctrl</kbd>+<kbd>Alt</kbd>+<kbd>S</kbd>) → **Plugins**.
3. Click the gear icon ⚙ → **Install Plugin from Disk…**
4. Select the `.zip` file and restart the IDE when prompted.

---

## Bug Reporting

If you encounter issues while using the extension please report them to us by creating a GitHub issue! You can create one easily here: <https://github.com/allseas/go-template-lsp/issues/new?template=bug.yml>.

The following sections explain how to find the relevant server logs.

### Server Logs in VS Code

To find the server logs you should go to the `Output` tab, you can do so by opening the command pallet (<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>P</kbd>) and then selecting `Output: Focus on Output View`. Then select `Go Template Language Server` and copy the logs into the bug report.

### Server Logs in JetBrains

To find the server logs you need to go the `Language Servers` tab, which you can open in the bottom-left menu. Then, you can navigate to `go-text-template-lsp` and click on the server process (it will be called `started pid:...`). You can copy the logs from `Traces` and `Logs`.

In order to see the logs you might need to not only change the options in the UI or config, but also in the `Language Servers` tab. To enable verbose logging here you need to select `go-text-template-lsp` and go into the `Debug` tab. Then select the `Trace` level to `verbose` and save.

---

## File Association

The extension activates for files with a double extension ending in `.tmpl`, for example:

```bash
page.html.tmpl
query.sql.tmpl
config.yaml.tmpl
```

Single-extension `.tmpl` files are also supported. The language ID used internally is `gotmpl`.

---

## Features

### Syntax Highlighting

Template tags (`{{ }}`) and their contents are highlighted separately from the surrounding text, regardless of what the host language is (HTML, SQL, YAML, etc.).

### Completions

Completions trigger automatically when you type `$` (variables) or `.` (dot fields/methods). You can also invoke them manually with <kbd>Ctrl</kbd>+<kbd>Space</kbd>.

What you get depends on where the cursor is:

| Cursor position                            | Suggestions                                                                                            |
| ------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `{{ $ }}`                                  | All variables in scope (`$`, `$i`, `$v`, …)                                                            |
| `{{ . }}`                                  | The `.` item, or all fields/methods of the current dot type (if a [type hint](#type-hints) is present) |
| `{{ .Address. }}`                          | Fields and methods of the element type (chained access)                                                |
| `{{ .Items \| len \| }}`                   | Only functions that accept `int` input                                                                 |
| `{{ .IsAdmin \| not \| }}`                 | Only functions that accept `bool` input                                                                |
| Inside `{{ if … }}`, `{{ range … }}`, etc. | All in-scope variables, dot fields/methods, and global functions                                       |

Pipe-aware filtering means the completion list is narrowed based on what the previous command in the pipe produces, so you won't be offered functions with incompatible input types.

### Hover Tooltips

Hovering over any node in a template file shows a tooltip. Examples:

| Hovered token          | Tooltip                                                                                                                    |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `if .Cond`             | If the value of the pipeline is empty, no output is generated; Otherwise, inside is executed.                              |
| `range .Items`         | Branch executed for each item in a collection.                                                                             |
| `with .Value`          | Branch executed with a new context.                                                                                        |
| `$i` in `range $i, $v` | `var $i int` - Serves as the index variable in the `range` loop, representing the current iteration count.                 |
| `$x`                   | `var $x Type` - variable declaration/type (or `(unknown)` if not resolvable)                                               |
| `.`                    | Returns the current context.                                                                                               |
| `.Name`                | `field .Name` - Accesses the `Name` field of the `.` context.                                                              |
| `end`                  | From `` `if` `` / `` `range` `` / `` `with` `` at line N.                                                                  |
| `else`                 | From `` `if` `` / `` `range` `` / `` `with` `` at line N.                                                                  |
| `len`                  | A built-in function that returns the length of its argument.                                                               |
| `and`                  | A built-in function that returns the first argument if it is false, and the last argument otherwise.                       |
| `or`                   | A built-in function that returns the first argument if it is true, and the last argument otherwise.                        |
| `not`                  | A built-in function that returns the boolean negation of its argument.                                                     |
| `nil`                  | `nil` is a predeclared identifier representing the zero value for a pointer, channel, func, interface, map, or slice type. |
| Other identifiers      | Represents an identifier in a command or action.                                                                           |

### Go to Definition

<kbd>Ctrl</kbd>+Click (or <kbd>F12</kbd>) on a symbol jumps to its definition:

| Symbol                                              | Behaviour                                                                                                                                                               |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `$x` (variable)                                     | Jumps to all `:=` declarations of `$x` in the file (VS Code shows all; JetBrains navigates to the first - see [client differences](features/definition.md#variables-x)) |
| `.` inside `range` or `with`                        | Jumps to the `range`/`with` pipe that redefines dot                                                                                                                     |
| `.FieldName`                                        | Jumps to the field or method declaration in the Go source file (requires a [type hint](#type-hints))                                                                    |
| `.Address.City`                                     | Jumps to whichever identifier the cursor is on                                                                                                                          |
| `upper`, `formatDate`, etc. (user-defined function) | Jumps to the function declaration in the Go source file (requires a [function hint](#function-hints))                                                                   |

### Find References

Right-click -> *Find All References* (or <kbd>Shift</kbd>+<kbd>F12</kbd>) lists every occurrence of a variable or identifier within the current file.

Supported symbols:

| Symbol                            | Behaviour                                    |
| --------------------------------- | -------------------------------------------- |
| `$x` (variable)                   | All uses of `$x` in the file                 |
| `upper`, `len`, etc. (identifier) | All uses of that identifier name in the file |

Field access nodes (`.FieldName`) are not supported by find references.

### Diagnostics

Errors appear as squiggly underlines as you type. The extension reports:

| Situation                                    | Message                                                                   |
| -------------------------------------------- | ------------------------------------------------------------------------- |
| Invalid character in a tag                   | `undefined variable: bad character U+003F '?'`                            |
| Empty action                                 | `missing value for command`                                               |
| Undeclared variable                          | `undefined variable: $x`                                                  |
| Unknown function                             | `unsupported function or unregistered command: foo`                       |
| Variable redeclared in the same scope        | `variable $x already declared in this scope`                              |
| Variable redeclared inside a `range` body    | `variable $x already declared in this scope`                              |
| `range` over a struct or other non-rangeable | `cannot range over value of type models.Order`                            |
| Invalid `gotype` hint (type not found)       | `could not load type "pkg.NoSuchType": …`                                 |
| Template called with wrong type              | `template "T" expects argument of type models.User, but got models.Order` |

### Snippets and Code Actions

- **Snippets** - common template patterns available via the completion menu
- **Wrap in comment** - wraps the selected text in `{{- /* … */ -}}`
- **Wrap in block** - wraps the selection in a block (`{{- if … }} … {{- end }}`), using snippets

---

## Type Hints

By default the extension knows the Go built-in template functions and the variables in scope, but it does not know the type of `.` (the dot context). Adding a type hint unlocks field and method completions, hover types, go-to-definition for fields, and type checking on template calls.

### Basic Usage

Place a comment on the **first line** of the template file (or immediately after `{{define "name"}}` for named blocks):

```go
{{- /*gotype: github.com/example/myapp/models.User*/ -}}
```

The format is `gotype: <import-path>.<TypeName>`. For a type in the same module you can use the package name directly:

```go
{{/*gotype: models.User*/}}
```

After saving, the extension loads the type from the Go source using `go list`. This takes a second or two on the first load; subsequent opens are instant because the result is cached.

With the hint in place:

- `{{ . }}` completes with all exported fields and methods of `User`
- `{{ .Address. }}` completes with the fields of whatever `Address` is
- Hovering `.Name` shows the field type
- <kbd>Ctrl</kbd>+Click on `.Name` jumps to the `Name` field in the Go source

### Multiple Define Blocks

Each `{{define}}` block can carry its own independent hint:

```go
{{- /*gotype: models.Address*/ -}}
Street: {{ .Street }}, {{ .City }}

{{define "OrderBlock"}}
{{- /*gotype: models.Order*/ -}}
Order #{{ .ID }} by {{ .Customer.Name }}
{{end}}

{{define "NoHintBlock"}}
{{ . }}   {{/* no type-aware features here */}}
{{end}}
```

### Template Type Checking

If a called template has a type hint, the extension checks that the argument you pass matches the expected type:

```go
{{- /*gotype: models.Order*/ -}}

{{/* correct - passing the whole Order to OrderBlock */}}
{{ template "OrderBlock" . }}

{{/* error - UserBlock expects models.User, not models.Order */}}
{{ template "UserBlock" . }}

{{/* correct - passing the User field */}}
{{ template "UserBlock" .Customer }}
```

The error appears as a diagnostic underline on the bad `{{ template … }}` call.

---

## Function Hints

If your Go code registers custom functions into a `template.FuncMap`, you can expose them to the extension by adding a `//tmpl:func "global"` comment above the function that returns the map:

```go
//tmpl:func "global"
func TemplateFuncs() template.FuncMap {
    return template.FuncMap{
        "upper":   strings.ToUpper,
        "repeat":  strings.Repeat,
        "formatDate": formatDate,
    }
}
```

Once annotated, `upper`, `repeat`, and `formatDate` will:

- appear in the completion list alongside built-in functions
- not produce "unsupported function" diagnostics
- be usable in pipe filtering (if the signature is resolvable)

The extension picks up changes automatically when any `.go` file in the workspace is saved - no restart needed.

---

## Configuration

### Project Config File

Create a `gotmpl.config.json` file in the root of your project to configure the language server for everyone working in that repo, regardless of which IDE they use:

```json
{
  "enableHover": true,
  "enableDefinition": true,
  "enableDiagnostics": true,
  "diagnostics": {
    "syntaxError": "error",
    "doubleDeclaredVariable": "warning",
    "invalidFunction": "warning"
  },
  "enableAutocompletion": true,
  "trace": {
    "server": "messages"
  }
}
```

This file takes precedence over any IDE-specific settings. A server restart is required for changes to take effect.

### VS Code Settings

Open *Settings* (<kbd>Ctrl</kbd>+<kbd>,</kbd>) and search for `Go Template`, or add entries directly to `.vscode/settings.json`:

```json
{
  "goTmplSupport.enableDiagnostics": true,
  "goTmplSupport.diagnostics": {
    "invalidFunction": "disabled"
  },
  "goTmplSupport.trace.server": "off"
}
```

### JetBrains Settings

Go to *Settings -> Tools -> Go Text Template*. Application-level settings apply globally; project-level settings (stored in `.idea/`) override them for the current project.

### Configuration Options Reference

Common options:

| Option                | Type    | Default      | Description                                             |
| --------------------- | ------- | ------------ | ------------------------------------------------------- |
| `enableHover`         | boolean | `true`       | Show hover tooltips                                     |
| `enableDefinition`    | boolean | `true`       | Enable go-to-definition                                 |
| `enableDiagnostics`   | boolean | `true`       | Enable all diagnostics                                  |
| `enableAutocompletion`| boolean | `true`       | Enable completions                                      |
| `pipeChainCompletion`                      | string  | `"full"`     | Nested field completions: `"off"`, `"full"`, or `"step"` |
| `trace.server`        | string  | `"messages"` | LSP trace level: `"off"`, `"messages"`, or `"verbose"`  |

Setting `trace.server` to `"verbose"` logs the full LSP traffic and is useful when debugging why a feature isn't working. The output appears in the *Output* panel (VS Code) or the LSP4IJ console (JetBrains).

The `diagnostics` object controls severity per check. For example:

```json
"diagnostics": {
  "invalidFunction": "warning",
  "doubleDeclaredVariable": "disabled"
}
```

For the full list of diagnostic keys and their defaults see [docs/configuration.md](configuration.md).

When a specific type in the template should be provided, `pipeChainCompletion` allows users to see valid fields/methods differently. For example, if a `string` should be provided, when `pipeChainCompletion` is set to `full`, we see all the available children fields/methods like `.Address.City` or `.Address.Info.Desc.Info1` up to the depth of 8. If it is set to `step`, only immediate fields are suggested in case they or their children match the type. When it is `off`, nested fields are not suggested.