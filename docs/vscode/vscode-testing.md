# VS Code Extension Testing Guide

This document describes how to create and run tests for the VS Code extension in this project.

## Overview

The VS Code extension uses the `@vscode/test-cli` and `@vscode/test-electron` testing framework, which is the recommended approach for testing VS Code extensions. Tests are written in TypeScript using Mocha as the test framework and Node's assert module for assertions.

## Test File Structure

Test files are located in `clients/VSCode/src/test/` and should follow the naming convention `*.test.ts`.

### Basic Test File Template

```typescript
import * as assert from "assert";
import { after, before } from "mocha";
import * as vscode from "vscode";

suite("My Test Suite", () => {
    before(async () => {
        // Wait for extension activation
        await new Promise(resolve => setTimeout(resolve, 1000));
    });

    after(() => {
        vscode.window.showInformationMessage("Tests completed!");
    });

    test("My test case", async () => {
        // Test code here
        assert.strictEqual(1 + 1, 2);
    });
});
```

## Creating Tests

### 1. Async Tests with Documents

```typescript
test("Should open document", async () => {
    const uri = vscode.Uri.file("/path/to/file.tmpl");
    const document = await vscode.workspace.openTextDocument(uri);
    
    assert.strictEqual(document.languageId, "gotmpl");
});
```

### 2. Working with File System

```typescript
test("Should create and cleanup files", async () => {
    const uri = vscode.Uri.file("/path/to/test-file.txt");
    
    // Create file
    const edit = new vscode.WorkspaceEdit();
    edit.createFile(uri, { overwrite: true });
    edit.insert(uri, new vscode.Position(0, 0), "test content");
    await vscode.workspace.applyEdit(edit);
    
    try {
        const document = await vscode.workspace.openTextDocument(uri);
        assert.ok(document);
    } finally {
        // Cleanup
        const deleteEdit = new vscode.WorkspaceEdit();
        deleteEdit.deleteFile(uri);
        await vscode.workspace.applyEdit(deleteEdit);
    }
});
```

### 3. Testing Language-Specific Features

The text/template language is associated with `.tmpl` files. When testing language-specific features:

```typescript
test("Snippets in gotmpl files", async () => {
    const tmplUri = vscode.Uri.file("/path/to/file.tmpl");
    const edit = new vscode.WorkspaceEdit();
    edit.createFile(tmplUri, { overwrite: true });
    await vscode.workspace.applyEdit(edit);
    
    const document = await vscode.workspace.openTextDocument(tmplUri);
    
    // Verify language is recognized
    assert.strictEqual(document.languageId, "gotmpl");
});
```

## Running Tests

### Run All Tests

```bash
cd clients/VSCode
npm test
```

This command will:

1. Compile TypeScript files
2. Launch VS Code in test mode
3. Run all test files
4. Exit with appropriate exit code

## Common Patterns

### File Cleanup

Always clean up temporary test files in a `finally` block:

```typescript
try {
    // Test code
} finally {
    await vscode.commands.executeCommand("workbench.action.closeActiveEditor");
    const deleteEdit = new vscode.WorkspaceEdit();
    deleteEdit.deleteFile(uri);
    await vscode.workspace.applyEdit(deleteEdit);
}
```

## VS Code Test Framework Resources

- [VS Code Testing Extension Guide](https://code.visualstudio.com/api/working-with-extensions/testing-extension)
- [Mocha Documentation](https://mochajs.org/)
- [Node.js Assert Module](https://nodejs.org/api/assert.html)
- [VS Code Extension API](https://code.visualstudio.com/api)
