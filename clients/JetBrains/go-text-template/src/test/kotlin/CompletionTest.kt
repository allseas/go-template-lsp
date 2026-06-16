import com.intellij.codeInsight.completion.CompletionType

class CompletionTest : CustomPlatformTestCase() {
    fun testAllCompletionCases() {
        val testCases = loadCompletionTestCases()
        for (tc in testCases) {
            // Cases that require a resolved Go type run in DotFieldsSuggestionsTest
            // (heavy fixture with the Go model project copied in).
            if (tc.vscodeOnly == true || requiresGoProject(tc.content)) continue
            val fileName = "completion-${tc.name.lowercase().replace(Regex("[^a-z0-9]+"), "-")}.txt.tmpl"
            myFixture.configureByText(fileName, toCaret(tc.content))
            myFixture.complete(CompletionType.BASIC)
            val completions = myFixture.lookupElementStrings ?: emptyList()

            for (expected in tc.expectedIncludes) {
                assertTrue(
                    "[${ tc.name}] Expected '$expected' in completions, got: $completions",
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
