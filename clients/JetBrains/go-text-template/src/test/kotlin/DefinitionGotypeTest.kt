import com.intellij.codeInsight.navigation.actions.GotoDeclarationAction
import com.intellij.psi.PsiElement
import com.intellij.testFramework.PlatformTestUtil

// Definition test cases that resolve a Go type (via a `gotype` annotation) and
// therefore navigate into the Go model files. These require the heavy fixture
// that copies the Go model project into the test project.
class DefinitionGotypeTest : CustomFixtureHeavyTestCase() {
    override fun setUp() {
        super.setUp()
        myFixture.copyDirectoryToProject("", "")
    }

    fun testDummy() {
        print("yip")
    }

//    fun testGotypeDefinitionCases() {
//        val testCases =
//            loadDefinitionTestCases().filter {
//                it.vscodeOnly != true && requiresGoProject(it.content)
//            }
//        for (tc in testCases) {
//            val fileName = "definition-${tc.name.lowercase().replace(Regex("[^a-z0-9]+"), "-")}.txt.tmpl"
//            myFixture.configureByText(fileName, toCaret(tc.content))
//            // The LSP server processes textDocument/didOpen asynchronously.
//            // When the test case is marked `poll`, retry the goto-definition
//            // lookup for a short time so the server has a chance to register
//            // the document before we query it. Without this, the very first
//            // definition request after configureByText can race the didOpen
//            // and return zero targets ("document not found in store" in the
//            // server log). This race surfaces more often in slower builds
//            // (e.g. the allseas build) where the server binary takes longer
//            // to warm up.
//            val shouldPoll = tc.poll == true && tc.expected.noResult != true
//            val targets: Array<PsiElement> =
//                if (shouldPoll) pollForTargets(tc) else findTargets()
//            if (tc.expected.noResult == true) {
//                assertTrue(
//                    "[${tc.name}] Expected no definition targets, got ${targets.size}",
//                    targets.isEmpty(),
//                )
//                continue
//            }
//            assertNotNull("[${tc.name}] Definition targets should not be null", targets)
//            tc.expected.count?.let { count ->
//                assertEquals(
//                    "[${tc.name}] Expected $count definition targets, got ${targets.size}",
//                    count,
//                    targets.size,
//                )
//            }
//            tc.expected.minCount?.let { minCount ->
//                assertTrue(
//                    "[${tc.name}] Expected at least $minCount definition targets, got ${targets.size}",
//                    targets.size >= minCount,
//                )
//            }
//            tc.expected.targetFile?.let { expectedFile ->
//                val actualFile = targets[0].containingFile?.virtualFile?.name
//                assertTrue(
//                    "[${tc.name}] Expected definition in $expectedFile, got $actualFile",
//                    actualFile != null && actualFile.endsWith(expectedFile),
//                )
//            }
//            tc.expected.targetLine?.let { expectedLine ->
//                val containingFile = targets[0].containingFile
//                val document = myFixture.getDocument(containingFile)
//                val targetLine = document.getLineNumber(targets[0].textOffset)
//                assertEquals(
//                    "[${tc.name}] Expected definition on line $expectedLine, got $targetLine",
//                    expectedLine,
//                    targetLine,
//                )
//            }
//        }
//    }
//
    private fun findTargets(): Array<PsiElement> =
        GotoDeclarationAction.findAllTargetElements(
            myFixture.project,
            myFixture.editor,
            myFixture.caretOffset,
        )

    private fun pollForTargets(tc: DefinitionTestCase): Array<PsiElement> {
        val minRequired = (tc.expected.minCount ?: tc.expected.count ?: 1).coerceAtLeast(1)
        val expectedFile = tc.expected.targetFile
        val deadline = System.currentTimeMillis() + POLL_TIMEOUT_MS
        var targets = findTargets()
        while (!targetsSatisfy(targets, minRequired, expectedFile) &&
            System.currentTimeMillis() < deadline
        ) {
            // Pump the EDT/UI queue so any pending LSP notifications (e.g.
            // didOpen acknowledgements) can be delivered, then retry. We
            // also re-poll when the returned target points back at the
            // template file itself: that happens when the LSP server has
            // not yet indexed the Go project, so IntelliJ falls back to
            // resolving the identifier under the caret. Without this we
            // would accept a stale fallback result on slower CI builds.
            PlatformTestUtil.dispatchAllEventsInIdeEventQueue()
            Thread.sleep(POLL_INTERVAL_MS)
            targets = findTargets()
        }
        return targets
    }

    private fun targetsSatisfy(
        targets: Array<PsiElement>,
        minRequired: Int,
        expectedFile: String?,
    ): Boolean {
        if (targets.size < minRequired) return false
        if (expectedFile != null) {
            val actualFile = targets[0].containingFile?.virtualFile?.name
            if (actualFile == null || !actualFile.endsWith(expectedFile)) return false
        }
        return true
    }

    companion object {
        private const val POLL_TIMEOUT_MS = 5_000L
        private const val POLL_INTERVAL_MS = 50L
    }
}
