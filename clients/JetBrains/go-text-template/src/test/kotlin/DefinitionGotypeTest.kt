import com.intellij.codeInsight.navigation.actions.GotoDeclarationAction
import com.intellij.psi.PsiElement

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

            val targets: Array<PsiElement> =
                GotoDeclarationAction.findAllTargetElements(
                    myFixture.project,
                    myFixture.editor,
                    myFixture.caretOffset,
                )

            if (tc.expected.noResult == true) {
                assertTrue(
                    "[${tc.name}] Expected no definition targets, got ${targets.size}",
                    targets.isEmpty(),
                )
                continue
            }

            assertNotNull("[${tc.name}] Definition targets should not be null", targets)

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
                val actualFile = targets[0].containingFile?.virtualFile?.name
                assertTrue(
                    "[${tc.name}] Expected definition in $expectedFile, got $actualFile",
                    actualFile != null && actualFile.endsWith(expectedFile),
                )
            }
            tc.expected.targetLine?.let { expectedLine ->
                val containingFile = targets[0].containingFile
                val document = myFixture.getDocument(containingFile)
                val targetLine = document.getLineNumber(targets[0].textOffset)
                assertEquals(
                    "[${tc.name}] Expected definition on line $expectedLine, got $targetLine",
                    expectedLine,
                    targetLine,
                )
            }
        }
    }
}
