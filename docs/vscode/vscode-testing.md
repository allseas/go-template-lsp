# VS Code Extension Testing Guide

This document describes how to create and run tests for the VS Code extension in this project.

## Overview

The VS Code extension uses the `@vscode/test-cli` and `@vscode/test-electron` testing framework, which is the recommended approach for testing VS Code extensions. Tests are written in TypeScript using Mocha as the test framework and Node's `assert` module for assertions.

Most LSP-feature tests (completion, definition, diagnostics) are data-driven: they load shared JSON files from `test/testcases/` and execute one Mocha test per entry. See the [main testing guide](../testing.md) for the full JSON schema and instructions on adding new cases.

## Test File Structure

```files
clients/VSCode/src/test/
├── utils.ts              ← shared helpers (document lifecycle, polling, grammar)
├── completion.test.ts    ← loads test/testcases/completion.json
├── definition.test.ts    ← loads test/testcases/definition.json
├── diagnostics.test.ts   ← loads test/testcases/diagnostics.json
├── dotFieldsSuggestions.test.ts
├── snippets.test.ts
└── syntax.test.ts
```

Test files follow the naming convention `*.test.ts`.

## Helpers vs. Test Logic

Client code should consist mostly of **helpers** in `utils.ts`, not inline test logic. Test files load the JSON cases, iterate over them, and delegate assertions to helpers. This keeps the client thin - adding or modifying a test case only requires editing the shared JSON file.

**Helpers in `utils.ts`** cover:

- `createDocument(fileName, content)` - writes a `.tmpl` file under `test/resources/` and opens it in the editor.
- `cleanupDocument(uri)` - closes the active editor and deletes the temporary file.
- `getGrammar()` / `getScopes()` / `assertScope()` - TextMate grammar utilities for syntax tests.

**Test files** should only contain:

- A `suite` block that loads the JSON file once.
- A `for` loop that registers one `test()` per case.
- Calls to helpers for all VS Code API interactions and polling.

Example structure for a new feature test file:

```typescript
import * as assert from "assert";
import * as fs from "fs";
import { after } from "mocha";
import * as path from "path";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const testCasesDir = path.join(__dirname, "../../../../test/testcases");

interface MyFeatureTestCase {
    name: string;
    content: string;
    vscodeOnly?: boolean;
    expected: { /* ... */ };
}

// Helper: keeps test logic out of the suite loop
async function runMyFeature(uri: vscode.Uri, position: vscode.Position) {
    return vscode.commands.executeCommand("vscode.executeMyFeatureProvider", uri, position);
}

suite("My Feature Test Suite", () => {
    after(() => vscode.window.showInformationMessage("All my-feature tests done!"));

    const testCases: MyFeatureTestCase[] = JSON.parse(
        fs.readFileSync(path.join(testCasesDir, "my-feature.json"), "utf-8"),
    );

    for (const tc of testCases) {
        test(tc.name, async () => {
            const fileName = `my-feature-${tc.name.toLowerCase().replace(/[^a-z0-9]+/g, "-")}.tmpl`;
            const { tmplUri } = await createDocument(fileName, tc.content);
            try {
                const result = await runMyFeature(tmplUri, new vscode.Position(0, 0));
                assert.ok(result, "Expected a result");
            } finally {
                cleanupDocument(tmplUri);
            }
        });
    }
});
```

## Adding New JSON-Driven Tests

1. Add an entry to the appropriate file in `test/testcases/` (see the [main testing guide](../testing.md#adding-new-test-cases)).
2. Run the tests - the new case is picked up automatically.
3. No changes to client test files are required for standard cases.

## Running Tests

```bash
cd clients/VSCode
npm test
```

This compiles TypeScript, launches VS Code in test mode, runs all test files, and exits with an appropriate exit code.

## Testing Syntax Highlighting

Syntax highlighting is tested at the TextMate grammar layer, not through the VS Code UI.

VS Code does not expose a stable public API or CLI command for reading TextMate scopes at a cursor position, so automated tests cannot reliably ask the editor “what scope is here?”. Internal commands such as `_workbench.captureSyntaxTokens` are undocumented and unstable across VS Code versions.

Because of that, syntax highlighting tests use `vscode-textmate` together with `vscode-oniguruma` to tokenize sample content directly. This runs the same TextMate grammar engine that VS Code uses internally, while keeping the tests deterministic and CI-friendly.

These tests verify that the grammar produces the expected scopes, which is the contract that the editor theme uses to render highlighting.

## VS Code Test Framework Resources

- [VS Code Testing Extension Guide](https://code.visualstudio.com/api/working-with-extensions/testing-extension)
- [Mocha Documentation](https://mochajs.org/)
- [Node.js Assert Module](https://nodejs.org/api/assert.html)
- [VS Code Extension API](https://code.visualstudio.com/api)
