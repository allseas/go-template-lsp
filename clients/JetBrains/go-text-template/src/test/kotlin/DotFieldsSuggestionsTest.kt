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
            val suggestions = myFixture.lookupElementStrings

            // A single matching completion is auto-inserted, so the lookup is
            // null. Verify the resulting document text instead.
            val expectedResult = tc.expectedResult
            if (expectedResult != null) {
                assertNull(
                    "[${tc.name}] Expected a single completion to be auto-inserted",
                    suggestions,
                )
                myFixture.checkResult(expectedResult)
                continue
            }

            val completions = suggestions ?: emptyList()

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
}
