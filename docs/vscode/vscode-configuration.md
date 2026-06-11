# VS Code Extension Configuration

See [Configuration](../configuration.md) for the general configuration overview and option descriptions. This document covers VS Code-specific implementation details.

## Architecture

VS Code configuration is unified at the workspace level. Settings are automatically read from:

- User settings (global IDE configuration)
- Workspace settings (`.vscode/settings.json`)
- Workspace folder settings (`.vscode/settings.json` per folder)

The extension reads settings via `workspace.getConfiguration()`, which respects this hierarchy with folder settings taking precedence over workspace settings, which take precedence over user settings.

## How to Add a New Configuration Option

### 1. Add to `package.json`

In `clients/VSCode/package.json`, add the new option under `contributes.configuration.properties`:

```json
{
  "contributes": {
    "configuration": {
      "properties": {
        "goTmplSupport.enableHover": {
          ...
        },
        "goTmplSupport.enableDefinition": {
          ...
        },
        "goTmplSupport.enableDiagnostics": {
          ...
        },
        "goTmplSupport.diagnostics.syntaxError": {
          ...
        },
        "goTmplSupport.diagnostics.variableRedeclaration": {
          ...
        },
        "goTmplSupport.diagnostics.incorrectFunction": {
          ...
        },
        "goTmplSupport.enableAutocompletion": {
          ...
        },
        "goTmplSupport.trace.server": {
          ...
        },
        "goTmplSupport.myNewOption": {
          "type": "string",
          "default": "default",
          "description": "Description of my new option."
        }
      }
    }
  }
}
```

Configuration property naming conventions:

- Use the section prefix: `goTmplSupport.*`
- Document the type, default value, and description
- For enums, use `enum: [...]`
