
# JetBrains Testing

## Diagnostics

Diagnostics should be tested with `checkHighlighting` or `doHighlighting`. However, we were not able to see the language server diagnostics in the tests with `myFixture`, even though they can be seen in the JetBrains IDE.

Documentation: <https://plugins.jetbrains.com/docs/intellij/testing-highlighting.html#special-cases>

The syntax for creating test cases should be easy to create: <https://plugins.jetbrains.com/docs/intellij/testing-highlighting.html#generating-test-data>

Example:

```xml
<error>{{ }}</error>
```
