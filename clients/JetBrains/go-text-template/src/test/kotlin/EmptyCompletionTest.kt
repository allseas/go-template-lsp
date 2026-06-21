import com.intellij.codeInsight.completion.CompletionType

/**
 * Regression tests for GoTemplateWordCompletionSuppressor.
 *
 * IntelliJ opens *.tmpl files with the TextMate language. The suppressor runs after
 * LSP4IJ's completion contributor (which provides the language-server items) and
 * before the platform's WordCompletionContributor, where it calls result.stopHere().
 * That stops the editor-word "Suggest words from the editor" fallback from adding
 * noise once the language server has had its say.
 *
 * These tests guard two things:
 *  - the suppressor must NOT drop the language-server completions (it runs *after*
 *    LSP4IJ, never before it), and
 *  - when the server returns nothing, no editor words leak into the suggestions.
 */
class EmptyCompletionTest : CustomPlatformTestCase() {
    fun testSuppressorDoesNotDropLspCompletions() {
        // With the suppressor wired after LSP4IJ, the language-server items must survive.
        // (If it were mis-registered before LSP4IJ, stopHere() would swallow these.)
        myFixture.configureByText("lsp-still-works.txt.tmpl", "{{<caret>}}")
        myFixture.complete(CompletionType.BASIC)
        val lspCompletions = myFixture.lookupElementStrings ?: emptyList()

        assertTrue(
            "Language-server completions must still be offered, got: $lspCompletions",
            lspCompletions.contains("$") && lspCompletions.contains("."),
        )
    }

    fun testEditorWordsNotSuggestedWhenLspReturnsNothing() {
        // "Zqxuniqueword" is a plain-text word in the document but is not a Go template
        // variable, builtin or function, so the language server returns nothing for the
        // "Zqx" prefix. Without suppression, WordCompletionContributor would offer it.
        myFixture.configureByText(
            "empty-completion.txt.tmpl",
            "Zqxuniqueword is only plain text\n{{ Zqx<caret> }}",
        )
        myFixture.complete(CompletionType.BASIC)
        val completions = myFixture.lookupElementStrings ?: emptyList()

        assertFalse(
            "Editor word 'Zqxuniqueword' should not be suggested (word completion must be suppressed), got: $completions",
            completions.contains("Zqxuniqueword"),
        )
    }

    fun testEditorWordsNotSuggestedInsidePlainText() {
        // Outside of an action ({{ }}) the language server has nothing to offer either,
        // so the only possible suggestion would be the editor word - which must be gone.
        myFixture.configureByText(
            "empty-completion-plain.txt.tmpl",
            "Zqxuniqueword appears here\nZqx<caret>",
        )
        myFixture.complete(CompletionType.BASIC)
        val completions = myFixture.lookupElementStrings ?: emptyList()

        assertFalse(
            "Editor word 'Zqxuniqueword' should not be suggested in plain text, got: $completions",
            completions.contains("Zqxuniqueword"),
        )
    }
}
