import com.intellij.codeInsight.navigation.actions.GotoDeclarationAction
import com.intellij.psi.PsiElement

class DefinitionTest : CustomPlatformTestCase() {
    fun testVariableDefinitionJumpsToDeclaration() {
        myFixture.configureByText(
            "test.txt.tmpl",
            $$"{{ $test := 0 }}\n{{ $te<caret>st }}",
        )

        val targets: Array<PsiElement> =
            GotoDeclarationAction.findAllTargetElements(
                project,
                myFixture.editor,
                myFixture.caretOffset,
            )

        assertNotNull("Definition targets should not be null", targets)
        assertTrue(
            "Expected at least 1 definition target, got ${targets.size}",
            targets.isNotEmpty(),
        )

        val targetOffset = targets[0].textOffset
        val targetLine = myFixture.editor.document.getLineNumber(targetOffset)
        assertEquals("Definition should point to line 0 (declaration)", 0, targetLine)
    }

    fun testRedeclaredVariableShowsDefinitions() {
        myFixture.configureByText(
            "test-redecl.txt.tmpl",
            $$"{{ $test := 0 }}\n{{ $test }}\n{{ $test := 1 }}\n{{ $te<caret>st }}",
        )

        val targets: Array<PsiElement> =
            GotoDeclarationAction.findAllTargetElements(
                project,
                myFixture.editor,
                myFixture.caretOffset,
            )

        assertNotNull("Definition targets should not be null", targets)
        assertTrue(
            "Expected at least 1 definition for redeclared variable, got ${targets.size}",
            targets.isNotEmpty(),
        )

        val targetOffset = targets[0].textOffset
        val targetLine = myFixture.editor.document.getLineNumber(targetOffset)
        assertTrue(
            "Definition should point to a declaration line (0 or 2), got $targetLine",
            targetLine == 0 || targetLine == 2,
        )
    }

    fun testDotInRangeJumpsToRangePipe() {
        myFixture.configureByText(
            "test-dot.txt.tmpl",
            "{{- range .Join }}\n{{ <caret>. }}\n{{- end }}",
        )

        val targets: Array<PsiElement> =
            GotoDeclarationAction.findAllTargetElements(
                project,
                myFixture.editor,
                myFixture.caretOffset,
            )

        assertNotNull("Definition targets should not be null", targets)
        assertTrue(
            "Expected at least 1 definition for dot in range, got ${targets.size}",
            targets.isNotEmpty(),
        )

        val targetOffset = targets[0].textOffset
        val targetLine = myFixture.editor.document.getLineNumber(targetOffset)
        assertEquals("Definition should point to range pipe on line 0", 0, targetLine)
    }
}
