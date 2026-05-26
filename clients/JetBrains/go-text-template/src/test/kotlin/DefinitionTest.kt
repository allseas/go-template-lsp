/**
 * Tests for Go to Definition functionality provided by the LSP server.
 *
 * Note: These tests verify the IDE integration side. The actual definition logic
 * is tested in the server-side Go tests (server/handlers/definition_test.go).
 * These tests require the LSP server to be running to pass in a full integration environment.
 */
class DefinitionTest : CustomPlatformTestCase() {
    fun testVariableDefinitionFileRecognized() {
        // Verify .tmpl files are recognized correctly for definition support
        val file =
            myFixture.configureByText(
                "test.txt.tmpl",
                "{{ \$test := 0 }}\n{{ \$test }}",
            )
        assertNotNull(file)
        assertEquals("Go Template", file.virtualFile.fileType.name)
    }

    fun testRedeclaredVariableFileRecognized() {
        val file =
            myFixture.configureByText(
                "test-redecl.txt.tmpl",
                "{{ \$test := 0 }}\n{{ \$test }}\n{{ \$test := 1 }}\n{{ \$test }}",
            )
        assertNotNull(file)
        assertEquals("Go Template", file.virtualFile.fileType.name)
    }

    fun testDotInRangeFileRecognized() {
        val file =
            myFixture.configureByText(
                "test-dot.txt.tmpl",
                "{{- range .Join }}\n{{ . }}\n{{- end }}",
            )
        assertNotNull(file)
        assertEquals("Go Template", file.virtualFile.fileType.name)
    }
}
