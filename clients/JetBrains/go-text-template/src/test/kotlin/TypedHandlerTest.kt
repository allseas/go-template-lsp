class TypedHandlerTest : CustomPlatformTestCase() {
    private fun assertNoTripleBrace(text: String) {
        assertFalse(
            "Buggy '{{}}}' / triple-close pattern leaked into the document: '$text'",
            text.contains("}}}") || text.contains("{{}}}"),
        )
    }

    fun testTypingDoubleBraceProducesProperPair() {
        myFixture.configureByText("test.tmpl", "")
        myFixture.type('{')
        myFixture.type('{')
        assertEquals("{{}}", myFixture.editor.document.text)
        assertEquals(2, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingSingleBraceInsertsNoAutoclose() {
        // A lone "{" must not get an autoclosed "}" (that stray "}" was one of
        // the two layers that produced the "{{}}}" bug).
        myFixture.configureByText("test.tmpl", "")
        myFixture.type('{')
        assertEquals("{", myFixture.editor.document.text)
        assertEquals(1, myFixture.editor.caretModel.offset)
    }

    fun testTypingDoubleBraceInsideExistingText() {
        myFixture.configureByText("test.tmpl", "foo<caret>bar")
        myFixture.type('{')
        myFixture.type('{')
        assertEquals("foo{{}}bar", myFixture.editor.document.text)
        assertEquals(5, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingDoubleBraceAtEndOfText() {
        myFixture.configureByText("test.tmpl", "foo<caret>")
        myFixture.type('{')
        myFixture.type('{')
        assertEquals("foo{{}}", myFixture.editor.document.text)
        assertEquals(5, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingDoubleBraceBeforeExistingClosingBrace() {
        // Caret sits before a stray "}" — the classic "{|}" state. The second
        // brace must consume the stray "}" rather than add a third one.
        myFixture.configureByText("test.tmpl", "{<caret>}")
        myFixture.type('{')
        assertEquals("{{}}", myFixture.editor.document.text)
        assertEquals(2, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingDoubleBraceOnNewLine() {
        myFixture.configureByText("test.tmpl", "line1\n<caret>")
        myFixture.type('{')
        myFixture.type('{')
        assertEquals("line1\n{{}}", myFixture.editor.document.text)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingThirdBraceDoesNotCascade() {
        // Typing a third "{" inside a finished "{{}}" must not produce
        // "{{{}}}" runaway nesting; it is inserted literally.
        myFixture.configureByText("test.tmpl", "{{<caret>}}")
        myFixture.type('{')
        assertNoTripleBrace(myFixture.editor.document.text)
        assertEquals("{{{}}", myFixture.editor.document.text)
    }

    fun testTypingDoubleBraceInNonTmplFileIsNotHandledByUs() {
        myFixture.configureByText("test.html", "")
        myFixture.type('{')
        myFixture.type('{')
        // We don't intervene in non-tmpl files; just assert our handler
        // didn't produce the "{{}}}" pattern we are guarding against.
        val text = myFixture.editor.document.text
        assertFalse(
            "Our handler must not touch non-tmpl files, got: '$text'",
            text == "{{}}}",
        )
    }

    // Trim-marker delimiter: typing "{{-" must produce "{{-  -}}" with the
    // caret sitting between the two body spaces, and never leak a triple close.
    fun testTypingTrimDelimiterProducesProperPair() {
        myFixture.configureByText("test.tmpl", "")
        myFixture.type("{{-")
        assertEquals("{{-  -}}", myFixture.editor.document.text)
        assertEquals(4, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingTrimDelimiterInsideExistingText() {
        myFixture.configureByText("test.tmpl", "foo<caret>bar")
        myFixture.type("{{-")
        assertEquals("foo{{-  -}}bar", myFixture.editor.document.text)
        assertEquals(7, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    // A lone "-" outside a fresh "{{|}}" must insert literally.
    fun testTypingDashOutsideDelimiterInsertsLiteral() {
        myFixture.configureByText("test.tmpl", "foo<caret>")
        myFixture.type('-')
        assertEquals("foo-", myFixture.editor.document.text)
    }

    fun testTypingTrimDelimiterInNonTmplFileIsNotHandledByUs() {
        myFixture.configureByText("test.html", "")
        myFixture.type("{{-")
        val text = myFixture.editor.document.text
        assertFalse(
            "Our handler must not touch non-tmpl files, got: '$text'",
            text == "{{-  -}}",
        )
    }

    // Comment delimiter: typing "{{/*" must produce "{{/**/}}" with the caret
    // between the two stars.
    fun testTypingCommentProducesProperPair() {
        myFixture.configureByText("test.tmpl", "")
        myFixture.type("{{/*")
        assertEquals("{{/**/}}", myFixture.editor.document.text)
        assertEquals(4, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingCommentInsideExistingText() {
        myFixture.configureByText("test.tmpl", "foo<caret>bar")
        myFixture.type("{{/*")
        assertEquals("foo{{/**/}}bar", myFixture.editor.document.text)
        assertEquals(7, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    // A lone "*" outside a comment context must insert literally.
    fun testTypingStarOutsideCommentInsertsLiteral() {
        myFixture.configureByText("test.tmpl", "foo<caret>")
        myFixture.type('*')
        assertEquals("foo*", myFixture.editor.document.text)
    }

    fun testTypingCommentInNonTmplFileIsNotHandledByUs() {
        myFixture.configureByText("test.html", "")
        myFixture.type("{{/*")
        val text = myFixture.editor.document.text
        assertFalse(
            "Our handler must not touch non-tmpl files, got: '$text'",
            text == "{{/**/}}",
        )
    }

    // Trim-comment delimiter: typing "{{- /*" must produce "{{- /*  */ -}}"
    // with the caret between the two body spaces.
    fun testTypingTrimCommentProducesProperPair() {
        myFixture.configureByText("test.tmpl", "")
        myFixture.type("{{-/*")
        assertEquals("{{- /*  */ -}}", myFixture.editor.document.text)
        assertEquals(7, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingTrimCommentInsideExistingText() {
        myFixture.configureByText("test.tmpl", "foo<caret>bar")
        myFixture.type("{{-/*")
        assertEquals("foo{{- /*  */ -}}bar", myFixture.editor.document.text)
        assertEquals(10, myFixture.editor.caretModel.offset)
        assertNoTripleBrace(myFixture.editor.document.text)
    }

    fun testTypingTrimCommentInNonTmplFileIsNotHandledByUs() {
        myFixture.configureByText("test.html", "")
        myFixture.type("{{- /*")
        val text = myFixture.editor.document.text
        assertFalse(
            "Our handler must not touch non-tmpl files, got: '$text'",
            text == "{{- /*  */ -}}",
        )
    }
}
