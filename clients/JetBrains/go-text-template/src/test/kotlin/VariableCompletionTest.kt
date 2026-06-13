import com.intellij.codeInsight.completion.CompletionType

class VariableCompletionTest : CustomPlatformTestCase() {
    private val testCasesDir = "../../../../test/testcases"

    fun testAllCompletionCases() {
        val testCases = loadCompletionTestCases(testCasesDir)
        for (tc in testCases) {
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
