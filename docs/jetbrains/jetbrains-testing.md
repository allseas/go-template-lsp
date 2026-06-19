
# JetBrains Testing

This document describes how to create and run tests for the JetBrains plugin in this project.

## Overview

The JetBrains plugin uses JUnit via Gradle. Most LSP-feature tests (completion, definition) are data-driven: they load shared JSON files from `test/testcases/` and execute one assertion block per entry. See the [main testing guide](../testing.md) for the full JSON schema and instructions on adding new cases.

## Test File Structure

```files
clients/JetBrains/go-text-template/src/test/kotlin/
├── TestCaseLoader.kt          ← loads JSON files and defines shared data classes
├── CustomPlatformTestCase.kt  ← base class wiring up the IntelliJ test fixture
├── CustomFixtureHeavyTestCase.kt
├── CompletionTest.kt          ← loads test/testcases/completion.json
├── DefinitionTest.kt          ← loads test/testcases/definition.json
├── DotFieldsSuggestionsTest.kt
├── DefinitionGotypeTest.kt
├── SnippetsTest.kt
├── SyntaxHighlightingTest.kt
├── BackspaceHandlerTest.kt
└── TypedHandlerTest.kt
```

## Helpers vs. Test Logic

Client code should consist mostly of **helpers**, not inline test logic. Test classes call helpers for loading cases, converting markers, and filtering cases. The actual assertions are driven by the shared JSON files.

**`TestCaseLoader.kt`** provides:

- `loadCompletionTestCases()` / `loadDefinitionTestCases()` - read the JSON files from classpath resources.
- Data classes (`CompletionTestCase`, `DefinitionTestCase`, …) that mirror the JSON schema.
- `toCaret(content)` - converts the `<cursor>` marker used in the JSON to IntelliJ's `<caret>` marker required by `myFixture.configureByText`.
- `requiresGoProject(content)` - returns `true` when a case contains a `gotype` annotation and therefore requires the heavy fixture with the Go model project.

**`CustomPlatformTestCase`** (base class) sets up the IntelliJ test fixture and configures `VfsRootAccess` for the test data path.

**Test classes** should only:

- Call the appropriate `loadXxxTestCases()` function.
- Filter out `vscodeOnly` cases and cases that `requiresGoProject` (those run in the separate heavy-fixture test class).
- Use `myFixture.configureByText(fileName, toCaret(tc.content))` to configure the fixture.
- Execute the IntelliJ action under test and assert against `tc.expected`.

## Adding New JSON-Driven Tests

1. Add an entry to the appropriate file in `test/testcases/` (see the [main testing guide](../testing.md#adding-new-test-cases)).
2. Run the tests - the new case is picked up automatically.
3. No changes to client Kotlin files are required for standard cases.
4. If a case uses a `gotype` annotation (needs the Go model project), add it to the relevant heavy-fixture test class (e.g., `DefinitionGotypeTest`, `DotFieldsSuggestionsTest`) instead of filtering it out.

## Running Tests

```bash
cd clients/JetBrains/go-text-template
./gradlew test
```

## Diagnostics

Diagnostics should be tested with `checkHighlighting` or `doHighlighting`. However, language server diagnostics are not currently visible in tests with `myFixture`, even though they appear correctly in the IDE.

Documentation: <https://plugins.jetbrains.com/docs/intellij/testing-highlighting.html#special-cases>

Test data syntax for highlighting annotations:

```xml
<error>{{ }}</error>
```

The test data generator described at <https://plugins.jetbrains.com/docs/intellij/testing-highlighting.html#generating-test-data> can be used to produce these files automatically.
