import com.intellij.codeInsight.navigation.actions.GotoDeclarationAction
import com.intellij.psi.PsiElement

class DefinitionTest : CustomPlatformTestCase() {
    fun testVariableDefinitionJumpsToDeclaration() {
        myFixture.configureByText(
            "test.txt.tmpl",
            "{{ \$test := 0 }}\n{{ \$te<caret>st }}",
        )

        val targets: Array<PsiElement>? =
            GotoDeclarationAction.findAllTargetElements(
                project,
                myFixture.editor,
                myFixture.caretOffset,
            )

        assertNotNull("Definition targets should not be null", targets)
        assertTrue(
            "Expected at least 1 definition target, got ${targets?.size ?: 0}",
            targets != null && targets.isNotEmpty(),
        )
    }

    fun testRedeclaredVariableShowsMultipleDefinitions() {
        myFixture.configureByText(
            "test-redecl.txt.tmpl",
            "{{ \$test := 0 }}\n{{ \$test }}\n{{ \$test := 1 }}\n{{ \$te<caret>st }}",
        )

        val targets: Array<PsiElement>? =
            GotoDeclarationAction.findAllTargetElements(
                project,
                myFixture.editor,
                myFixture.caretOffset,
            )

        assertNotNull("Definition targets should not be null", targets)
        assertEquals(
            "Expected 2 definitions for redeclared variable",
            2,
            targets?.size ?: 0,
        )
    }

    fun testDotInRangeJumpsToRangePipe() {
        myFixture.configureByText(
            "test-dot.txt.tmpl",
            "{{- range .Join }}\n{{ <caret>. }}\n{{- end }}",
        )

        val targets: Array<PsiElement>? =
            GotoDeclarationAction.findAllTargetElements(
                project,
                myFixture.editor,
                myFixture.caretOffset,
            )

        assertNotNull("Definition targets should not be null", targets)
        assertTrue(
            "Expected at least 1 definition for dot in range, got ${targets?.size ?: 0}",
            targets != null && targets.isNotEmpty(),
        )
    }

    fun testDotAtTopLevelHasNoDefinition() {
        myFixture.configureByText(
            "test-dot-top.txt.tmpl",
            "{{ <caret>. }}",
        )

        val targets: Array<PsiElement>? =
            GotoDeclarationAction.findAllTargetElements(
                project,
                myFixture.editor,
                myFixture.caretOffset,
            )

        assertTrue(
            "Expected no definitions for top-level dot",
            targets == null || targets.isEmpty(),
        )
    }
}
