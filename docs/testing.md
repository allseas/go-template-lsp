# Testing

The project has three independent test suites - one per component - each with its own test runner.

| Component                               | Language   | Framework                    | How to run                                                |
| --------------------------------------- | ---------- | ---------------------------- | --------------------------------------------------------- |
| Language server (`server/`)             | Go         | `testing` + `testify`        | `cd server && go test ./...`                              |
| VS Code extension (`clients/VSCode/`)   | TypeScript | Mocha via `@vscode/test-cli` | `cd clients/VSCode && npm run test`                       |
| JetBrains plugin (`clients/JetBrains/`) | Kotlin     | JUnit via Gradle             | `cd clients/JetBrains/go-text-template && ./gradlew test` |

Test fixtures shared across the suites live in `test/resources/`.

## Shared JSON Test Cases

The client test suites share a common set of test cases defined in JSON files under `test/testcases/`:

| File                              | Feature tested                    |
| --------------------------------- | --------------------------------- |
| `test/testcases/completion.json`  | Completion suggestions            |
| `test/testcases/definition.json`  | Go-to-definition                  |
| `test/testcases/diagnostics.json` | Diagnostics (errors and warnings) |

Both the VS Code and JetBrains clients read these files at test time and iterate over the cases automatically. Adding a new entry to a JSON file is therefore sufficient to cover both clients - no client-side test code needs to change.

### Common Fields

All test case objects share these fields:

| Field        | Type    | Description                                                                                                                              |
| ------------ | ------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `name`       | string  | Human-readable name used as the test case label.                                                                                         |
| `content`    | string  | Template source content. Use `\n` for newlines.                                                                                          |
| `vscodeOnly` | boolean | When `true`, the JetBrains client skips the case.                                                                                        |
| `poll`       | boolean | When `true`, the client retries until the LSP resolves the result (needed for cases that require async Go type resolution via `gotype`). |

### Completion Cases (`completion.json`)

| Field                         | Type     | Description                                                                   |
| ----------------------------- | -------- | ----------------------------------------------------------------------------- |
| `expectedIncludes`            | string[] | Labels that must appear in the completion list.                               |
| `expectedExcludes`            | string[] | Labels that must not appear in the completion list.                           |
| `expectedIncludesExactlyOnce` | string[] | Labels that must appear exactly once in the completion list.                  |
| `expectedResult`              | string   | If set, the full resulting document after applying the first completion item. |

Place the cursor position in `content` using the `<cursor>` marker:

```json
{
  "name": "Dollar sign always suggested",
  "content": "{{<cursor>}}",
  "expectedIncludes": ["$"],
  "expectedExcludes": []
}
```

### Definition Cases (`definition.json`)

| Field                 | Type    | Description                                                            |
| --------------------- | ------- | ---------------------------------------------------------------------- |
| `expected.minCount`   | number  | Minimum number of definition targets expected.                         |
| `expected.count`      | number  | Exact number of definition targets expected.                           |
| `expected.targetLine` | number  | 0-based line number in the target file the first result must point to. |
| `expected.targetFile` | string  | Filename (basename) that the first result must point to.               |
| `expected.noResult`   | boolean | When `true`, asserts that no definition is returned.                   |

Place the cursor using `<cursor>` in `content`, the same as for completion cases.

### Diagnostics Cases (`diagnostics.json`)

| Field                        | Type   | Description                                                 |
| ---------------------------- | ------ | ----------------------------------------------------------- |
| `expected.count`             | number | Exact number of diagnostics expected.                       |
| `expected.minCount`          | number | Minimum number of diagnostics expected.                     |
| `expected.diagnostics`       | array  | Per-diagnostic assertions (see below).                      |
| diagnostic `index`           | number | 0-based index into the diagnostics list for this assertion. |
| diagnostic `severity`        | string | `"error"` or `"warning"`.                                   |
| diagnostic `message`         | string | Full expected diagnostic message.                           |
| diagnostic `messageContains` | string | Substring that the diagnostic message must contain.         |
| diagnostic `source`          | string | Expected `source` field of the diagnostic.                  |
| diagnostic `rangeStart`      | object | `{ line, character }` - expected start position.            |
| diagnostic `rangeEnd`        | object | `{ line, character }` - expected end position.              |
| diagnostic `rangeStartLine`  | number | Short-hand to assert only the start line.                   |

## Adding New Test Cases

1. Open the relevant JSON file in `test/testcases/`.
2. Append a new object to the top-level array.
3. Fill in `name`, `content`, and the feature-specific expected fields described above.
4. If the test requires a resolved Go type (i.e. `content` contains a `gotype` annotation), set `"poll": true` so both clients wait for async type resolution.
5. If behaviour intentionally differs between clients, set `"vscodeOnly": true` to restrict the case to VS Code.

No changes to client code are needed for standard cases.

## Client Code: Helpers vs. Tests

Client code in `clients/VSCode/src/test/` and `clients/JetBrains/.../src/test/kotlin/` should consist mostly of **helpers** rather than test logic. The actual test assertions are driven entirely by the shared JSON files.

- **Helpers** encapsulate reusable operations: creating/cleaning up documents, executing LSP commands, polling for async results, converting the `<cursor>` marker, and loading the JSON test cases.
- **Test files** load the relevant JSON file, iterate over the cases, and delegate each case to a helper. They should contain minimal inline logic.

This separation keeps the clients thin and ensures that adding or modifying a test case requires editing only the shared JSON file.

## Platform-Specific Guides

- [VS Code Testing](vscode/vscode-testing.md)
- [JetBrains Testing](jetbrains/jetbrains-testing.md)
