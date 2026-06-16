import com.intellij.codeInsight.completion.CompletionType

// This class tests dot-field/method suggestions that depend on a resolved Go type.
// The shared completion test cases (test/testcases/completion.json) that require a
// Go project are driven here, since this fixture copies the Go model into the project.
// See issues 101 and 102: some assertions may need updating once those are fixed.
class DotFieldsSuggestionsTest : CustomFixtureHeavyTestCase() {
    override fun setUp() {
        super.setUp()
        myFixture.copyDirectoryToProject("", "")
    }

    fun testGotypeCompletionCases() {
        val testCases =
            loadCompletionTestCases().filter {
                it.vscodeOnly != true && requiresGoProject(it.content)
            }
        for (tc in testCases) {
            val fileName = "completion-${tc.name.lowercase().replace(Regex("[^a-z0-9]+"), "-")}.txt.tmpl"
            myFixture.configureByText(fileName, toCaret(tc.content))
            myFixture.complete(CompletionType.BASIC)
            val completions = myFixture.lookupElementStrings ?: emptyList()

            for (expected in tc.expectedIncludes) {
                assertTrue(
                    "[${tc.name}] Expected '$expected' in completions, got: $completions",
                    completions.contains(expected),
                )
            }
            for (excluded in tc.expectedExcludes) {
                assertFalse(
                    "[${tc.name}] Expected '$excluded' to NOT be in completions",
                    completions.contains(excluded),
                )
            }
            for (once in tc.expectedIncludesExactlyOnce ?: emptyList()) {
                val count = completions.count { it == once }
                assertEquals(
                    "[${tc.name}] Expected '$once' exactly once in completions",
                    1,
                    count,
                )
            }
        }
    }

    fun testPartialFieldNameFiltering() {
        myFixture.configureByText(
            "test_partial.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{.Cus<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        // Only one suggestion, so returns null and autocompletes
        assertNull(suggestions)
        myFixture.checkResult(
            """
            {{/*gotype: cg/model.Order*/}}
            {{.CustomerName}}
            """.trimIndent(),
        )
    }
}
