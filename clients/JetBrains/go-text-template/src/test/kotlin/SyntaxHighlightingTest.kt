import org.jetbrains.plugins.textmate.language.syntax.lexer.TextMateElementType

/**
 * Tests for static syntax highlighting using the TextMate grammar.
 * Translated from the VSCode syntax.test.ts with the use of ai.
 */
class SyntaxHighlightingTest : CustomPlatformTestCase() {
    /**
     * Gets the full TextMate scope string at the given character offset.
     * Uses the editor highlighter which produces TextMateElementType tokens for .tmpl files.
     */
    private fun getScopeAt(
        content: String,
        offset: Int,
    ): String {
        myFixture.configureByText("test.tmpl", content)
        val editor = myFixture.editor
        val highlighter = editor.highlighter
        val iterator = highlighter.createIterator(offset)
        val tokenType = iterator.tokenType ?: return ""
        if (tokenType is TextMateElementType) {
            return tokenType.scope.toString()
        }
        return tokenType.toString()
    }

    private fun assertScopeAt(
        content: String,
        offset: Int,
        expectedScope: String,
    ) {
        val actualScope = getScopeAt(content, offset)
        assertTrue(
            "Expected scope containing '$expectedScope' at offset $offset in '$content', but got '$actualScope'",
            actualScope.contains(expectedScope),
        )
    }

    private fun assertNoGotmplScope(
        content: String,
        offset: Int,
    ) {
        val actualScope = getScopeAt(content, offset)
        assertFalse(
            "Plain text should not have gotmpl-specific scopes, got: '$actualScope'",
            actualScope.contains("gotmpl") && !actualScope.contains("source.gotmpl"),
        )
    }

    // --- Action delimiters ---

    fun testActionDelimiterBeginIsHighlighted() {
        assertScopeAt("{{ .Foo }}", 0, "punctuation.definition.embedded.begin.gotmpl")
    }

    fun testActionDelimiterEndIsHighlighted() {
        assertScopeAt("{{ .Foo }}", 8, "punctuation.definition.embedded.end.gotmpl")
    }

    // --- Trim marker delimiters ---

    fun testTrimMarkerDelimiterBeginIsHighlighted() {
        assertScopeAt("{{- .Foo -}}", 0, "punctuation.definition.embedded.begin.gotmpl")
    }

    fun testTrimMarkerDelimiterEndIsHighlighted() {
        assertScopeAt("{{- .Foo -}}", 9, "punctuation.definition.embedded.end.gotmpl")
    }

    // --- Keywords ---

    fun testKeywordIfIsHighlighted() {
        assertScopeAt("{{ if }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordElseIsHighlighted() {
        assertScopeAt("{{ else }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordEndIsHighlighted() {
        assertScopeAt("{{ end }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordRangeIsHighlighted() {
        assertScopeAt("{{ range }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordWithIsHighlighted() {
        assertScopeAt("{{ with }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordBlockIsHighlighted() {
        assertScopeAt("{{ block }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordDefineIsHighlighted() {
        assertScopeAt("{{ define }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordTemplateIsHighlighted() {
        assertScopeAt("{{ template }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordBreakIsHighlighted() {
        assertScopeAt("{{ break }}", 3, "keyword.control.gotmpl")
    }

    fun testKeywordContinueIsHighlighted() {
        assertScopeAt("{{ continue }}", 3, "keyword.control.gotmpl")
    }

    // --- Comments ---

    fun testCommentIsHighlighted() {
        assertScopeAt("{{/* a comment */}}", 0, "comment.block.gotmpl")
    }

    fun testTrimmedCommentIsHighlighted() {
        assertScopeAt("{{- /* a comment */ -}}", 0, "comment.block.gotmpl")
    }

    fun testCommentFollowedByTrailingWhitespaceAndNewlineEnds() {
        val content = "{{/* a comment */}} \nhello\n{{ .Foo }}\nmore text\n"
        val actualScope = getScopeAt(content, content.length - 5)
        assertFalse(
            "Text after comment+trailing-whitespace+newline should not be comment, got: '$actualScope'",
            actualScope.contains("comment"),
        )
    }

    // --- Variables ---

    fun testVariableDeclarationIsHighlighted() {
        assertScopeAt("{{ \$name := 0 }}", 3, "variable.other.gotmpl")
    }

    fun testVariableDeclarationOperatorIsHighlighted() {
        assertScopeAt("{{ \$name := 0 }}", 9, "keyword.operator.assignment.gotmpl")
    }

    fun testVariableAssignmentIsHighlighted() {
        assertScopeAt("{{ \$x = 1 }}", 3, "variable.other.gotmpl")
    }

    fun testVariableAssignmentOperatorIsHighlighted() {
        assertScopeAt("{{ \$x = 1 }}", 6, "keyword.operator.assignment.gotmpl")
    }

    fun testVariableReferenceIsHighlighted() {
        assertScopeAt("{{ \$myVar }}", 3, "variable.other.gotmpl")
    }

    fun testBareDollarSignIsHighlightedAsVariable() {
        assertScopeAt("{{ \$ }}", 3, "variable.other.gotmpl")
    }

    // --- Dot and field access ---

    fun testStandaloneDotIsHighlighted() {
        assertScopeAt("{{ . }}", 3, "variable.language.dot.gotmpl")
    }

    fun testFieldAccessIsHighlighted() {
        assertScopeAt("{{ .Name }}", 3, "variable.other.member.gotmpl")
    }

    // --- Builtin functions ---

    fun testBuiltinAndIsHighlighted() {
        assertScopeAt("{{ and }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinCallIsHighlighted() {
        assertScopeAt("{{ call }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinHtmlIsHighlighted() {
        assertScopeAt("{{ html }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinIndexIsHighlighted() {
        assertScopeAt("{{ index }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinSliceIsHighlighted() {
        assertScopeAt("{{ slice }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinJsIsHighlighted() {
        assertScopeAt("{{ js }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinLenIsHighlighted() {
        assertScopeAt("{{ len }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinNotIsHighlighted() {
        assertScopeAt("{{ not }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinOrIsHighlighted() {
        assertScopeAt("{{ or }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinPrintIsHighlighted() {
        assertScopeAt("{{ print }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinPrintfIsHighlighted() {
        assertScopeAt("{{ printf }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinPrintlnIsHighlighted() {
        assertScopeAt("{{ println }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinUrlqueryIsHighlighted() {
        assertScopeAt("{{ urlquery }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinEqIsHighlighted() {
        assertScopeAt("{{ eq }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinNeIsHighlighted() {
        assertScopeAt("{{ ne }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinLtIsHighlighted() {
        assertScopeAt("{{ lt }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinLeIsHighlighted() {
        assertScopeAt("{{ le }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinGtIsHighlighted() {
        assertScopeAt("{{ gt }}", 3, "support.function.gotmpl")
    }

    fun testBuiltinGeIsHighlighted() {
        assertScopeAt("{{ ge }}", 3, "support.function.gotmpl")
    }

    // --- Pipe operator ---

    fun testPipeOperatorIsHighlighted() {
        assertScopeAt("{{ .Name | html }}", 9, "keyword.operator.pipe.gotmpl")
    }

    // --- Boolean literals ---

    fun testBooleanTrueIsHighlighted() {
        assertScopeAt("{{ true }}", 3, "constant.language.boolean.gotmpl")
    }

    fun testBooleanFalseIsHighlighted() {
        assertScopeAt("{{ false }}", 3, "constant.language.boolean.gotmpl")
    }

    // --- Nil ---

    fun testNilIsHighlighted() {
        assertScopeAt("{{ nil }}", 3, "constant.language.nil.gotmpl")
    }

    // --- Numbers ---

    fun testIntegerNumberIsHighlighted() {
        assertScopeAt("{{ 42 }}", 3, "constant.numeric.gotmpl")
    }

    fun testFloatNumberIsHighlighted() {
        assertScopeAt("{{ 3.14 }}", 3, "constant.numeric.gotmpl")
    }

    fun testHexNumberIsHighlighted() {
        assertScopeAt("{{ 0xFF }}", 3, "constant.numeric.gotmpl")
    }

    // --- Strings ---

    fun testDoubleQuotedStringIsHighlighted() {
        assertScopeAt("{{ \"hello\" }}", 3, "string.quoted.double.gotmpl")
    }

    fun testRawStringIsHighlighted() {
        assertScopeAt("{{ `raw` }}", 3, "string.quoted.other.raw.gotmpl")
    }

    fun testCharLiteralIsHighlighted() {
        assertScopeAt("{{ 'a' }}", 3, "string.quoted.single.gotmpl")
    }

    // --- Escape sequence ---

    fun testEscapeSequenceInsideStringIsHighlighted() {
        assertScopeAt("{{ \"\\n\" }}", 4, "constant.character.escape.gotmpl")
    }

    // --- Template names ---

    fun testTemplateNameAfterDefineIsHighlighted() {
        assertScopeAt("{{ define \"myTemplate\" }}", 10, "entity.name.function.gotmpl")
    }

    fun testTemplateNameAfterTemplateIsHighlighted() {
        assertScopeAt("{{ template \"myTemplate\" }}", 12, "entity.name.function.gotmpl")
    }

    // --- Parentheses ---

    fun testOpeningParenthesisIsHighlighted() {
        assertScopeAt("{{ if (eq .A .B) }}", 6, "punctuation.section.parens.begin.gotmpl")
    }

    fun testClosingParenthesisIsHighlighted() {
        assertScopeAt("{{ if (eq .A .B) }}", 15, "punctuation.section.parens.end.gotmpl")
    }

    // --- Plain text ---

    fun testPlainTextOutsideActionsHasNoGotmplTokenScopes() {
        assertNoGotmplScope("hello world", 0)
    }

    fun testRangeOutsideBracketsIsNotHighlightedAsKeyword() {
        val content = "range"
        val actualScope = getScopeAt(content, 0)
        assertFalse(
            "Expected 'range' outside brackets to NOT have keyword.control.gotmpl scope, but got '$actualScope'",
            actualScope.contains("keyword.control.gotmpl"),
        )
    }
}
