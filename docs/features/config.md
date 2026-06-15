# Config File Support

You can configure the language server project-wide by creating a `gotmpl.config.json` file in the root directory of your project. This configuration applies to all IDEs (VS Code, JetBrains, etc.) and takes precedence over IDE-specific settings.

## Example Configuration

```json
{
  "enableHover": true,
  "enableDefinition": true,
  "enableDiagnostics": true,
  "diagnostics": {
    "syntaxError": true,
    "variableRedeclaration": true,
    "incorrectFunction": true
  },
  "enableAutocompletion": true,
  "trace": {
    "server": "messages"
  }
}
```

## Configuration Options

| Option                                | Type      | Default    | Description                                                       |
|---------------------------------------|-----------|------------|-------------------------------------------------------------------|
| `enableHover`                         | `boolean` | `true`     | Enable/disable hover information                                  |
| `enableDefinition`                    | `boolean` | `true`     | Enable/disable go-to-definition                                   |
| `enableDiagnostics`                   | `boolean` | `true`     | Enable/disable all diagnostics                                    |
| `diagnostics.syntaxError`             | `boolean` | `true`     | Report syntax errors                                              |
| `diagnostics.variableRedeclaration`   | `boolean` | `true`     | Report duplicate variable declarations                            |
| `diagnostics.incorrectFunction`       | `boolean` | `true`     | Report unknown or incorrectly used functions                      |
| `enableAutocompletion`                | `boolean` | `true`     | Enable/disable autocompletion                                     |
| `trace.server`                        | `string`  | `messages` | Trace communication: `off`, `messages`, or `verbose`             |

## Notes

- A restart is needed for changes to apply.
- The configuration in `gotmpl.config.json` takes precedence over IDE-specific settings and user preferences.
- Individual diagnostic options (`diagnostics.*`) only take effect when `enableDiagnostics` is `true`.
