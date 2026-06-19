import com.intellij.codeInsight.navigation.actions.GotoDeclarationAction
import com.intellij.psi.PsiElement

class DefinitionTest : CustomPlatformTestCase() {
    fun testAllDefinitionCases() {
        val testCases = loadDefinitionTestCases()
        for (tc in testCases) {
            if (tc.vscodeOnly == true) continue
            // Cases that resolve a Go type run in DefinitionGotypeTest
            // (heavy fixture with the Go model project copied in).
            if (requiresGoProject(tc.content)) continue
            val fileName = "definition-${tc.name.lowercase().replace(Regex("[^a-z0-9]+"), "-")}.txt.tmpl"
            myFixture.configureByText(fileName, toCaret(tc.content))

            val targets: Array<PsiElement> =
                GotoDeclarationAction.findAllTargetElements(
                    project,
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
            tc.expected.targetLine?.let { expectedLine ->
                val targetOffset = targets[0].textOffset
                val targetLine = myFixture.editor.document.getLineNumber(targetOffset)
                assertEquals(
                    "[${tc.name}] Expected definition on line $expectedLine, got $targetLine",
                    expectedLine,
                    targetLine,
                )
            }
        }
    }
}
