class TypedHandlerTest : CustomPlatformTestCase() {
    fun testTypingDoubleBraceProducesProperPair() {
        myFixture.configureByText("test.tmpl", "")
        myFixture.type('{')
        myFixture.type('{')
        assertEquals("{{}}", myFixture.editor.document.text)
        assertEquals(2, myFixture.editor.caretModel.offset)
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
}
