# Config File Support

You can configure the language server project-wide by creating a `gotmpl.config.json` file in the root directory of your project. This configuration applies to all IDEs (VS Code, JetBrains, etc.) and takes precedence over IDE-specific settings.

## Example Configuration

```json
{
  "enableServer": true,
  "trace": {
    "server": "messages" 
  }
}
```

## Configuration Options

| Option         | Type      | Default    | Description                                          |
|----------------|-----------|------------|------------------------------------------------------|
| `enableServer` | `boolean` | `true`     | Enable/disable the language server                   |
| `trace.server` | `string`  | `messages` | Trace communication: `off`, `messages`, or `verbose` |

## Notes

- A restart is needed for changes to apply.
- The configuration in `gotmpl.config.json` takes precedence over IDE-specific settings and user preferences.