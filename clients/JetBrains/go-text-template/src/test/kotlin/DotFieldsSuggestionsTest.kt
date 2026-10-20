import com.intellij.codeInsight.completion.CompletionType
import com.intellij.codeInsight.lookup.LookupManager
import com.intellij.testFramework.PlatformTestUtil

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
            val suggestions = completeWithPolling(fileName, toCaret(tc.content), tc)

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

    private fun complete(
        fileName: String,
        text: String,
    ): List<String>? {
        myFixture.configureByText(fileName, text)
        myFixture.complete(CompletionType.BASIC)
        return myFixture.lookupElementStrings
    }

    // The LSP server processes textDocument/didOpen asynchronously, so the
    // first completion request after configureByText can race it and come
    // back empty ("document not found in store" in the server log), most
    // often on the first case of a fresh server. For cases marked `poll`,
    // retry until the expected items show up or the timeout expires. Each
    // retry reconfigures the file so a stale lookup or an auto-inserted item
    // from the previous attempt cannot leak into the next one.
    private fun completeWithPolling(
        fileName: String,
        text: String,
        tc: CompletionTestCase,
    ): List<String>? {
        var suggestions = complete(fileName, text)
        if (tc.poll != true || tc.expectedResult != null || tc.expectedIncludes.isEmpty()) {
            return suggestions
        }
        val deadline = System.currentTimeMillis() + POLL_TIMEOUT_MS
        while (!suggestions.orEmpty().containsAll(tc.expectedIncludes) &&
            System.currentTimeMillis() < deadline
        ) {
            LookupManager.hideActiveLookup(myFixture.project)
            PlatformTestUtil.dispatchAllEventsInIdeEventQueue()
            Thread.sleep(POLL_INTERVAL_MS)
            suggestions = complete(fileName, text)
        }
        return suggestions
    }

    companion object {
        private const val POLL_TIMEOUT_MS = 5_000L
        private const val POLL_INTERVAL_MS = 50L
    }
}
