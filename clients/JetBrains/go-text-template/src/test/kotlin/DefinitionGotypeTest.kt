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

    fun testGotypeDefinitionCases() {
        val testCases =
            loadDefinitionTestCases().filter {
                it.vscodeOnly != true && requiresGoProject(it.content)
            }
        for (tc in testCases) {
            val fileName = "definition-${tc.name.lowercase().replace(Regex("[^a-z0-9]+"), "-")}.txt.tmpl"
            myFixture.configureByText(fileName, toCaret(tc.content))

            var targets: Array<PsiElement> = findTargets()

            // The LSP definition response is asynchronous. For polling cases the
            // first request can return before the server resolved the target, in
            // which case GotoDeclaration falls back to the element under the
            // caret (the template file itself). Retry, pumping the IDE event
            // queue so background LSP responses are applied between attempts,
            // until a cross-file target appears - mirroring VSCode's
            // pollDefinitions helper.
            if (tc.poll == true && tc.expected.noResult != true) {
                var attempts = 0
                while (attempts < 40 &&
                    targets.none { it.containingFile?.virtualFile?.name != fileName }
                ) {
                    PlatformTestUtil.dispatchAllInvocationEventsInIdeEventQueue()
                    Thread.sleep(200)
                    PlatformTestUtil.dispatchAllInvocationEventsInIdeEventQueue()
                    targets = findTargets()
                    attempts++
                }
            }

            if (tc.expected.noResult == true) {
                assertTrue(
                    "[${tc.name}] Expected no definition targets, got ${targets.size}",
                    targets.isEmpty(),
                )
                continue
            }

            assertNotNull("[${tc.name}] Definition targets should not be null", targets)

            // Prefer a cross-file target (the real definition) over a fallback to
            // the template element under the caret.
            val target =
                targets.firstOrNull { it.containingFile?.virtualFile?.name != fileName }
                    ?: targets[0]

            tc.expected.count?.let { count ->
                assertEquals(
                    "[${tc.name}] Expected $count definition targets, got ${targets.size}",
                    count,
                    targets.size,
                )
            }
            tc.expected.minCount?.let { minCount ->
                assertTrue(
                    "[${tc.name}] Expected at least $minCount definition targets, got ${targets.size}",
                    targets.size >= minCount,
                )
            }
            tc.expected.targetFile?.let { expectedFile ->
                val actualFile = target.containingFile?.virtualFile?.name
                assertTrue(
                    "[${tc.name}] Expected definition in $expectedFile, got $actualFile",
                    actualFile != null && actualFile.endsWith(expectedFile),
                )
            }
            tc.expected.targetLine?.let { expectedLine ->
                val containingFile = target.containingFile
                val document = myFixture.getDocument(containingFile)
                val targetLine = document.getLineNumber(target.textOffset)
                assertEquals(
                    "[${tc.name}] Expected definition on line $expectedLine, got $targetLine",
                    expectedLine,
                    targetLine,
                )
            }
        }
    }

    private fun findTargets(): Array<PsiElement> =
        GotoDeclarationAction.findAllTargetElements(
            myFixture.project,
            myFixture.editor,
            myFixture.caretOffset,
        )
}
