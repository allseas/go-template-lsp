import com.intellij.openapi.actionSystem.IdeActions

class BackspaceHandlerTest : CustomPlatformTestCase() {
    private fun backspace() {
        myFixture.performEditorAction(IdeActions.ACTION_EDITOR_BACKSPACE)
    }

    /**
     * Inverse of the primary `{{` insertion case: backspacing once inside a
     * freshly auto-paired empty delimiter must wipe ALL four braces.
     */
    fun testBackspaceInsideEmptyDelimiterDeletesAllFourBraces() {
        myFixture.configureByText("test.tmpl", "{{<caret>}}")
        backspace()
        assertEquals("", myFixture.editor.document.text)
        assertEquals(0, myFixture.editor.caretModel.offset)
    }

    /**
     * Round-trip sanity: typing `{{` then backspacing must return the buffer
     * to its initial state, confirming the typed-handler and backspace-handler
     * are exact inverses of each other.
     */
    fun testTypeDoubleBraceThenBackspaceRoundTrip() {
        myFixture.configureByText("test.tmpl", "foo<caret>bar")
        myFixture.type('{')
        myFixture.type('{')
        assertEquals("foo{{}}bar", myFixture.editor.document.text)
        backspace()
        assertEquals("foobar", myFixture.editor.document.text)
        assertEquals(3, myFixture.editor.caretModel.offset)
    }

    /**
     * Backspacing inside a single `{|}` brace pair must NOT trigger our
     * four-brace collapse (which would over-delete and corrupt the
     * document). The four-brace delegate's preconditions require two `{`
     * and two `}`, so it must stay out of the way here and let the
     * platform's normal single-char backspace run.
     */
    fun testBackspaceInsideSingleBracePairDoesNotCollapse() {
        myFixture.configureByText("test.tmpl", "{<caret>}")
        backspace()
        val text = myFixture.editor.document.text
        // Our delegate must not fire: it requires `{{|}}`. The platform's
        // default behaviour here removes only the `{` (and possibly the
        // closing `}` if it was tracked as auto-inserted). Either way, at
        // most 2 chars should be gone from the 2-char source.
        assertTrue(
            "Single-pair backspace produced unexpected text '$text'",
            text == "" || text == "}",
        )
    }

    /**
     * When the delimiter is NOT empty (`{{ expr |}}`), backspace must behave
     * like a normal single-char delete: it removes only the space before the
     * caret and leaves both delimiters intact. It must never collapse the
     * four braces.
     */
    fun testBackspaceInsideNonEmptyDelimiterDeletesOneChar() {
        myFixture.configureByText("test.tmpl", "{{ expr <caret>}}")
        backspace()
        val text = myFixture.editor.document.text
        assertEquals("{{ expr}}", text)
        // The four delimiter braces must all still be present.
        assertTrue("Outer braces collapsed unexpectedly: '$text'", text.startsWith("{{") && text.endsWith("}}"))
    }

//    /**
//     * Caret sitting OUTSIDE a complete delimiter (`{{}}|`) must perform a
//     * plain single-char backspace, removing only one `}` -- the delegate
//     * must not fire here because the caret is not between two `{` and two
//     * `}`.
//     */
//    fun testBackspaceOutsideDelimiterDeletesOnlyOneBrace() {
//        myFixture.configureByText("test.tmpl", "{{}}<caret>")
//        backspace()
//        assertEquals("{{}", myFixture.editor.document.text)
//    }

    /**
     * Triple-brace runs (e.g. `{{{|}}` or `{{|}}}`) are user-authored and
     * must NOT trigger the four-brace collapse. The platform's own backspace
     * machinery still runs and may remove an adjacent matching brace -- that
     * is acceptable; what is NOT acceptable is our delegate firing and
     * wiping all four braces.
     */
    fun testBackspaceInsideTripleOpeningBraceDoesNotCollapse() {
        myFixture.configureByText("test.tmpl", "{{{<caret>}}")
        backspace()
        val text = myFixture.editor.document.text
        // Our 4-brace collapse would leave <= 1 char behind; the platform
        // default removes at most 2 chars from the 5-char source.
        assertTrue(
            "Four-brace collapse fired unexpectedly on triple opener: '$text'",
            text.length >= 3,
        )
        assertTrue("Lost the inner `{{` delimiter: '$text'", text.contains("{{"))
    }

    fun testBackspaceInsideTripleClosingBraceDoesNotCollapse() {
        myFixture.configureByText("test.tmpl", "{{<caret>}}}")
        backspace()
        val text = myFixture.editor.document.text
        assertTrue(
            "Four-brace collapse fired unexpectedly on triple closer: '$text'",
            text.length >= 3,
        )
        assertTrue("Lost the trailing `}}` delimiter: '$text'", text.contains("}}"))
    }

    /**
     * The delegate must be scoped to `.tmpl` files only -- backspacing inside
     * `{{|}}` in an unrelated file must NOT trigger the four-brace collapse.
     */
    fun testBackspaceInNonTmplFileIsNotHandledByUs() {
        myFixture.configureByText("test.html", "{{<caret>}}")
        backspace()
        // We don't touch non-tmpl files; the document must NOT become empty
        // due to *our* delegate. The platform may still do its own thing,
        // but it cannot collapse all four braces.
        val text = myFixture.editor.document.text
        assertFalse(
            "Our backspace delegate must not fire in non-tmpl files (text='$text')",
            text.isEmpty(),
        )
    }

    // Trim-marker delimiter: `{{-  |  -}}` collapses to empty on one backspace.
    fun testBackspaceInsideEmptyTrimDelimiterDeletesEverything() {
        myFixture.configureByText("test.tmpl", "{{- <caret> -}}")
        backspace()
        assertEquals("", myFixture.editor.document.text)
        assertEquals(0, myFixture.editor.caretModel.offset)
    }

    fun testTypeTrimDelimiterThenBackspaceRoundTrip() {
        myFixture.configureByText("test.tmpl", "foo<caret>bar")
        myFixture.type("{{-")
        assertEquals("foo{{-  -}}bar", myFixture.editor.document.text)
        backspace()
        assertEquals("foobar", myFixture.editor.document.text)
        assertEquals(3, myFixture.editor.caretModel.offset)
    }

    // Non-empty trim delimiter must not collapse: only the one space vanishes.
    fun testBackspaceInsideNonEmptyTrimDelimiterDeletesOneChar() {
        myFixture.configureByText("test.tmpl", "{{- expr <caret>-}}")
        backspace()
        assertEquals("{{- expr-}}", myFixture.editor.document.text)
    }

    // Comment delimiter: `{{/*|*/}}` collapses to empty on one backspace.
    fun testBackspaceInsideEmptyCommentDeletesEverything() {
        myFixture.configureByText("test.tmpl", "{{/*<caret>*/}}")
        backspace()
        assertEquals("", myFixture.editor.document.text)
        assertEquals(0, myFixture.editor.caretModel.offset)
    }

    fun testTypeCommentThenBackspaceRoundTrip() {
        myFixture.configureByText("test.tmpl", "foo<caret>bar")
        myFixture.type("{{/*")
        assertEquals("foo{{/**/}}bar", myFixture.editor.document.text)
        backspace()
        assertEquals("foobar", myFixture.editor.document.text)
        assertEquals(3, myFixture.editor.caretModel.offset)
    }

    // Non-empty comment must not collapse.
    fun testBackspaceInsideNonEmptyCommentDeletesOneChar() {
        myFixture.configureByText("test.tmpl", "{{/* hi <caret>*/}}")
        backspace()
        val text = myFixture.editor.document.text
        assertEquals("{{/* hi*/}}", text)
        assertTrue("Comment delimiters vanished: '$text'", text.startsWith("{{/*") && text.endsWith("*/}}"))
    }

    // Trim-comment delimiter: `{{- /*  |  */ -}}` collapses to empty.
    fun testBackspaceInsideEmptyTrimCommentDeletesEverything() {
        myFixture.configureByText("test.tmpl", "{{- /* <caret> */ -}}")
        backspace()
        assertEquals("", myFixture.editor.document.text)
        assertEquals(0, myFixture.editor.caretModel.offset)
    }

    fun testTypeTrimCommentThenBackspaceRoundTrip() {
        myFixture.configureByText("test.tmpl", "foo<caret>bar")
        myFixture.type("{{-/*") // The space is automatically inserted by the TypedHandler
        assertEquals("foo{{- /*  */ -}}bar", myFixture.editor.document.text)
        backspace()
        assertEquals("foobar", myFixture.editor.document.text)
        assertEquals(3, myFixture.editor.caretModel.offset)
    }

    // Non-empty trim-comment must not collapse.
    fun testBackspaceInsideNonEmptyTrimCommentDeletesOneChar() {
        myFixture.configureByText("test.tmpl", "{{- /* hi <caret>*/ -}}")
        backspace()
        val text = myFixture.editor.document.text
        assertEquals("{{- /* hi*/ -}}", text)
        assertTrue("Trim-comment delimiters vanished: '$text'", text.startsWith("{{- /*") && text.endsWith("*/ -}}"))
    }

    // Trim collapse must be scoped to `.tmpl` files.
    fun testBackspaceTrimInNonTmplFileIsNotHandledByUs() {
        myFixture.configureByText("test.html", "{{- <caret> -}}")
        backspace()
        val text = myFixture.editor.document.text
        assertFalse(
            "Our backspace delegate must not fire in non-tmpl files (text='$text')",
            text.isEmpty(),
        )
    }

    // Adjacent brace runs must still veto the trim/comment collapses.
    fun testBackspaceInsideTrimWithAdjacentBraceDoesNotCollapse() {
        myFixture.configureByText("test.tmpl", "{{{- <caret> -}}")
        backspace()
        val text = myFixture.editor.document.text
        assertTrue("Trim collapse fired unexpectedly on adjacent brace: '$text'", text.length >= 4)
    }
}
