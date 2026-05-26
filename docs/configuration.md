# Configuration

The Go Text Template LSP supports consistent configuration across VS Code and JetBrains IDEs. Configuration options are the same on all platforms, but the storage and UI differ.

## Platform-Specific Guides

- **[VS Code Configuration](vscode/vscode-configuration.md)** — Settings stored in `settings.json`, read via the Settings UI or config files
- **[JetBrains Configuration](jetbrains/jetbrains-configuration.md)** — Application-level + project-level settings, stored in IDE/project configs

## Configuration Options

These options are supported on all platforms:

| Option         | Type      | Default    | Description                                          |
|----------------|-----------|------------|------------------------------------------------------|
| `enableServer` | `boolean` | `true`     | Enable/disable the language server                   |
| `trace.server` | `string`  | `messages` | Trace communication: `off`, `messages`, or `verbose` |

## Configuration Hierarchy

Settings follow this precedence (highest to lowest):

1. **Project/Workspace** — `.vscode/settings.json` (VS Code) or `.idea/` config (JetBrains)
2. **User** — Global IDE user settings
3. **Defaults** — Plugin defaults (all servers enabled, trace at `messages` level)

## Adding New Options

To add a new configuration option to both clients:

1. Follow the [VS Code Configuration](vscode/vscode-configuration.md#how-to-add-a-new-configuration-option) guide to add to VS Code
2. Follow the [JetBrains Configuration](jetbrains/jetbrains-configuration.md#how-to-add-a-new-configuration-option) guide to add to JetBrains
3. Add to the LSP server's `Config` struct in `server/handlers/configuration.go`
