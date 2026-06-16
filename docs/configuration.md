# Configuration

The Go Text Template LSP supports consistent configuration across VS Code and JetBrains IDEs. Configuration options are the same on all platforms, but the storage and UI differ.

## Platform-Specific Guides

- **[VS Code Configuration](vscode/vscode-configuration.md)** - Settings stored in `settings.json`, read via the Settings UI or config files
- **[JetBrains Configuration](jetbrains/jetbrains-configuration.md)** - Application-level + project-level settings, stored in IDE/project configs

## Configuration Options

These options are supported on all platforms:

| Option              | Type      | Default    | Description                                          |
|---------------------|-----------|------------|------------------------------------------------------|
| `enableHover`       | `boolean` | `true`     | Enable/disable hover information                     |
| `enableDefinition`  | `boolean` | `true`     | Enable/disable go-to-definition                      |
| `enableDiagnostics` | `boolean` | `true`     | Enable/disable all diagnostics                       |
| `diagnostics`       | `object`  | see below  | Per-diagnostic severity levels (see table below)     |
| `enableAutocompletion` | `boolean` | `true`  | Enable/disable autocompletion                        |
| `trace.server`      | `string`  | `messages` | Trace communication: `off`, `messages`, or `verbose` |

### `diagnostics` keys

Each key in the `diagnostics` object controls a specific check. The value must be one of:
`"disabled"`, `"error"`, `"warning"`, `"information"`, `"hint"`.

| Key                    | Default       | Description                                               |
|------------------------|---------------|-----------------------------------------------------------|
| `syntaxError`          | `"error"`     | Syntax errors reported by the parser                      |
| `invalidField`         | `"error"`     | Field or method lookup failed                             |
| `invalidFunction`      | `"warning"`   | Unknown or incorrectly called function                    |
| `invalidCommand`       | `"error"`     | Command type mismatch                                     |
| `invalidRange`         | `"error"`     | Range over a non-rangeable type                           |
| `invalidIf`            | `"error"`     | If condition is not boolean                               |
| `invalidWith`          | `"error"`     | With dot is not a struct/interface                        |
| `undeclaredVariable`   | `"error"`     | Variable used without declaration                         |
| `doubleDeclaredVariable` | `"warning"` | Variable declared more than once in the same scope        |
| `invalidTemplateArg`   | `"error"`     | Template called with an argument of the wrong type        |
| `argumentNumberMismatch` | `"error"`   | Function called with the wrong number of arguments        |
| `unknownType`          | `"information"` | Type information is missing or could not be resolved    |
| `hintLoadFailure`      | `"warning"`   | A `gotype` hint type could not be loaded or resolved      |
| `unknownRangeType`     | `"warning"`   | Range over a value whose type could not be determined     |
| `emptyDefineName`      | `"warning"`   | Define block has an empty name                            |

## Configuration Hierarchy

Settings follow this precedence (highest to lowest):

1. **Project File** - `gotmpl.config.json` in project root (applies to all IDEs)
2. **IDE Project/Workspace** - `.vscode/settings.json` (VS Code) or `.idea/` config (JetBrains)
3. **User** - Global IDE user settings
4. **Defaults** - Plugin defaults (all servers enabled, trace at `messages` level)

## Project Configuration File

You can create a `gotmpl.config.json` file in your project root to configure the language server for your entire project. This configuration applies across all IDEs (VS Code, JetBrains, etc.):

```json
{
  "enableHover": true,
  "enableDefinition": true,
  "enableDiagnostics": true,
  "diagnostics": {
    "syntaxError": "error",
    "invalidField": "error",
    "invalidFunction": "warning",
    "invalidCommand": "error",
    "invalidRange": "error",
    "invalidIf": "error",
    "invalidWith": "error",
    "undeclaredVariable": "error",
    "doubleDeclaredVariable": "warning",
    "invalidTemplateArg": "error",
    "argumentNumberMismatch": "error",
    "unknownType": "information",
    "hintLoadFailure": "warning",
    "unknownRangeType": "warning",
    "emptyDefineName": "warning"
  },
  "enableAutocompletion": true,
  "trace": {
    "server": "messages"
  }
}
```

The project configuration takes precedence over IDE-specific settings and user preferences.

## Adding New Options

To add a new configuration option to both clients:

1. Follow the [VS Code Configuration](vscode/vscode-configuration.md#how-to-add-a-new-configuration-option) guide to add to VS Code
2. Follow the [JetBrains Configuration](jetbrains/jetbrains-configuration.md#how-to-add-a-new-configuration-option) guide to add to JetBrains
3. Add to the LSP server's `Config` struct in `server/handlers/configuration.go`
