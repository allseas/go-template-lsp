# Config File Support

You can configure the language server project-wide by creating a `gotmpl.config.json` file in the root directory of your project. This configuration applies to all IDEs (VS Code, JetBrains, etc.) and takes precedence over IDE-specific settings.

See [Configuration](../configuration.md) for the full reference.

## Notes

- A restart is needed for changes to apply.
- The configuration in `gotmpl.config.json` takes precedence over IDE-specific settings and user preferences.
- Individual diagnostic options (`diagnostics.*`) only take effect when `enableDiagnostics` is `true`.
