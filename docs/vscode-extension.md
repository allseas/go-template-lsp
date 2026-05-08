# VS Code Extension Architecture

This document describes the architecture of the VS Code extension and provides guides for extending it with new features.

## Overview

The VS Code extension connects the editor to the language server via the Language Server Protocol (LSP). It handles:

- Launching the appropriate server binary for the platform
- Watching template files for changes
- Forwarding editor events to the server
- Displaying results (completions, hover info, etc.) to the user

## Architecture

### Components

```files
VS Code Extension (TypeScript)
    │
    ├─ extension.ts .................. Entry point, LSP client setup
    ├─ package.json .................. Manifest, scripts, dependencies
    ├─ language-configuration.json ... Bracket matching, commenting
    ├─ snippets/
    │   └─ snippets.json ............ Code snippets
    ├─ syntaxes/
    │   └─ gotmpl.tmLanguage.json ... Syntax highlighting definition
    └─ src/
        ├─ extension.ts ............ Main extension code
        └─ test/
            └─ sample.test.ts ...... Test files
```

### Platform-Specific Binaries

The extension supports multiple platforms by shipping with all server binaries:

```typescript
let binaryName: string;

if (process.platform === "win32") {
    binaryName = process.arch === "arm64" 
        ? "gotmpl-server-arm64.exe"
        : "gotmpl-server.exe";
} else if (process.platform === "darwin") {
    binaryName = process.arch === "arm64"
        ? "gotmpl-server-darwin-arm64"
        : "gotmpl-server-darwin-amd64";
} else {
    binaryName = process.arch === "arm64"
        ? "gotmpl-server-arm64"
        : "gotmpl-server";
}
```

### File Watching

The extension watches for changes to `.*.tmpl` files (e.g., `template.html.tmpl`):

```typescript
for (const folder of workspace.workspaceFolders) {
    const watcher = workspace.createFileSystemWatcher(
        new RelativePattern(folder, "**/*.*.tmpl"),
    );
    
    watcher.onDidCreate((uri) => console.log(`Created: ${uri.fsPath}`));
    watcher.onDidChange((uri) => console.log(`Changed: ${uri.fsPath}`));
    watcher.onDidDelete((uri) => console.log(`Deleted: ${uri.fsPath}`));
}
```

These events are forwarded to the server so it can track document changes.

## Activation

The extension activates when:

1. A workspace folder is opened
2. The first template file is edited

This is controlled by `activationEvents` in `package.json`.

## Design Decisions

### Binary Transport

The extension runs the language server as a subprocess and communicates via stdio (standard input/output). This is the recommended approach for LSP in VS Code.

## Adding a New Feature

### Example: Adding Hover Support

#### 1. Create a Command or Handler

If you're adding a feature that needs a command, add it to `extension.ts`:

```typescript
// In activate()
const hoverCommand = vscode.commands.registerCommand(
    'goTmplSupport.showHover',
    async () => {
        const editor = vscode.window.activeTextEditor;
        if (!editor) {
            return;
        }
        
        const document = editor.document;
        const position = editor.selection.active;
        
        // Server handles this if registered
        // Results are displayed automatically
    }
);

context.subscriptions.push(hoverCommand);
```

#### 2. Update Language Configuration (if needed)

In `language-configuration.json`, define editor behavior:

```json
{
  "comments": {
    "blockComment": ["{{/*", "*/}}"]
  },
  "brackets": [
    ["{{" ,"}}"],
    ["[", "]"],
    ["{", "}"]
  ]
}
```

#### 3. Update Syntax Highlighting (if needed)

If you're adding new syntax, update `syntaxes/gotmpl.tmLanguage.json`:

```json
{
  "name": "Go text/template",
  "scopeName": "text.template.gotmpl",
  "patterns": [
    {
      "include": "#template"
    }
  ],
  "repository": {
    "template": {
      "patterns": [
        {
          "match": "{{.*?}}",
          "name": "meta.template.gotmpl"
        }
      ]
    }
  }
}
```

#### 4. Add UI Bindings (if command-based)

In `package.json`, register the command:

```json
{
  "contributes": {
    "commands": [
      {
        "command": "goTmplSupport.showHover",
        "title": "Show Hover Information",
        "category": "Go Template"
      }
    ],
    "keybindings": [
      {
        "command": "goTmplSupport.showHover",
        "key": "ctrl+k ctrl+i",
        "mac": "cmd+k cmd+i",
        "when": "editorLangId == gotmpl"
      }
    ]
  }
}
```

#### 5. Test the Feature

```typescript
// src/test/hover.test.ts
import * as assert from "assert";
import { after, before } from "mocha";
import * as vscode from "vscode";

suite("Hover Support", () => {
    test("Should show hover information", async () => {
        const uri = vscode.Uri.file("/path/to/test.tmpl");
        
        // Create test file
        const edit = new vscode.WorkspaceEdit();
        edit.createFile(uri, { overwrite: true });
        edit.insert(uri, new vscode.Position(0, 0), "{{ .Name }}");
        await vscode.workspace.applyEdit(edit);
        
        // Open document
        const document = await vscode.workspace.openTextDocument(uri);
        const editor = await vscode.window.showTextDocument(document);
        
        // Verify hover works
        editor.selection = new vscode.Selection(
            new vscode.Position(0, 3),
            new vscode.Position(0, 3)
        );
        
        // Command would be executed here
        // Results would be verified
        
        assert.ok(true);
    });
});
```

### Example: Adding a Code Snippet

Snippets provide quick template generation. Add to `snippets/snippets.json`:

```json
{
  "If Block": {
    "prefix": "if",
    "body": [
      "{{- if ${1:.condition} }}",
      "${2:content}",
      "{{- end }}"
    ],
    "description": "Create an if block"
  },
  "Range Block": {
    "prefix": "range",
    "body": [
      "{{- range ${1:.items} }}",
      "${2:content}",
      "{{- end }}"
    ],
    "description": "Create a range block"
  }
}
```

## Configuration

Read [vscode-config.md](vscode-config.md) for documentation about config in the VS Code extension.

## Testing

Read [vscode-testing.md](vscode-testing.md) for a guide on testing the VS Code extension.

## Building and Packaging

Read the main [README](./../README.md) section about how to run and build the extension.

## Resources

- [VS Code Extension API](https://code.visualstudio.com/api)
- [LSP Client Documentation](https://github.com/microsoft/vscode-languageclient)
- [TextMate Grammar Reference](https://macromates.com/manual/en/language_grammars)
- [Example VS Code Extensions](https://github.com/microsoft/vscode-extension-samples)
