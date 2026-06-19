# Testing

The project has three independent test suites - one per component - each with its own test runner.

| Component                               | Language   | Framework                    | How to run                                                |
| --------------------------------------- | ---------- | ---------------------------- | --------------------------------------------------------- |
| Language server (`server/`)             | Go         | `testing` + `testify`        | `cd server && go test ./...`                              |
| VS Code extension (`clients/VSCode/`)   | TypeScript | Mocha via `@vscode/test-cli` | `cd clients/VSCode && npm run test`                       |
| JetBrains plugin (`clients/JetBrains/`) | Kotlin     | JUnit via Gradle             | `cd clients/JetBrains/go-text-template && ./gradlew test` |

Test fixtures shared across the suites live in `test/resources/`.

## Platform-Specific Guides

- [VS Code Testing](vscode/vscode-testing.md)
- [JetBrains Testing](jetbrains/jetbrains-testing.md)
