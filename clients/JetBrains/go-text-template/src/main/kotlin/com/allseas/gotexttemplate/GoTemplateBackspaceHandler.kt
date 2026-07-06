package com.allseas.gotexttemplate
import com.intellij.codeInsight.editorActions.BackspaceHandlerDelegate
import com.intellij.openapi.editor.Editor
import com.intellij.psi.PsiDocumentManager
import com.intellij.psi.PsiFile

/**
 * Mirror image of [GoTemplateTypedHandler]: when the user backspaces inside an
 * auto-inserted Go template delimiter pair, everything the typed handler
 * inserted should disappear so that insert and one backspace are exact
 * inverses of each other.
 *
 * The typed handler synthesizes four different "empty" shapes; each of them
 * has a matching post-deletion pattern below. Outside of those exact shapes
 * both hooks fall through to the platform's default behaviour, so normal
 * single-brace / single-char backspace is preserved.
 *
 * The delegate is also scoped to `*.tmpl` files via [isGoTemplateFile] (the
 * same check used by [GoTemplateTypedHandler]) so it can never fire in
 * unrelated languages even though `backspaceHandlerDelegate` itself is a
 * non-language-scoped extension point.
 */
class GoTemplateBackspaceHandler : BackspaceHandlerDelegate() {
    /**
     * Each pattern describes the state of the buffer *after* the platform has
     * deleted one character in response to a backspace, when the caret was
     * sitting inside a freshly auto-inserted empty template of the
     * corresponding shape. [deleted] is the character the platform removed;
     * [left] must match the text immediately before the caret and [right] the
     * text immediately after.
     */
    private data class Collapse(
        val deleted: Char,
        val left: String,
        val right: String,
    )

    private val collapses =
        listOf(
            // `{{|}}`  --backspace-->  `{|}}`
            Collapse('{', "{", "}}"),
            // `{{- |}}`  --backspace-->  `{{-|}}` (deletes the trailing space)
            Collapse(' ', "{{-", "}}"),
            // `{{/**/}}`  --backspace-->  `{{/|*/}}` (deletes one `*`; the
            // surviving `*` shifts left so the right side reads `*/}}`)
            Collapse('*', "{{/", "*/}}"),
            // `{{- /*  |  */ -}}`  --backspace-->  `{{- /* | */ -}}` (one space)
            Collapse(' ', "{{- /*", " */ -}}"),
        )

    override fun beforeCharDeleted(
        c: Char,
        file: PsiFile,
        editor: Editor,
    ) {
        // Nothing to do: we cannot safely mutate the document here without
        // confusing the platform's own caret bookkeeping. All cleanup is
        // performed in [charDeleted], which runs after the platform has
        // removed the char under the caret.
    }

    /**
     * Returning `true` tells the platform that we have already taken care of
     * any auto-pair cleanup, so it must NOT invoke its default "delete the
     * matching closing brace" logic (which would remove one extra `}` and
     * leave the document in an inconsistent state).
     */
    override fun charDeleted(
        c: Char,
        file: PsiFile,
        editor: Editor,
    ): Boolean {
        if (!isGoTemplateFile(file)) return false
        if (editor.selectionModel.hasSelection()) return false

        val document = editor.document
        val caret = editor.caretModel.offset
        val text = document.charsSequence

        for (collapse in collapses) {
            if (c != collapse.deleted) continue
            val start = caret - collapse.left.length
            val end = caret + collapse.right.length
            if (start < 0 || end > text.length) continue
            if (!regionMatches(text, start, collapse.left)) continue
            if (!regionMatches(text, caret, collapse.right)) continue
            // Guard against longer brace runs adjacent to the delimiter, e.g.
            // `{{{|}}` or `{{|}}}` (user-authored code that just happens to
            // contain matching braces). Those must fall through to the
            // platform default rather than collapsing here.
            if (start > 0 && text[start - 1] == '{') continue
            if (end < text.length && text[end] == '}') continue

            document.deleteString(start, end)
            editor.caretModel.moveToOffset(start)
            PsiDocumentManager.getInstance(file.project).commitDocument(document)
            return true
        }
        return false
    }

    private fun regionMatches(
        text: CharSequence,
        offset: Int,
        expected: String,
    ): Boolean {
        for (i in expected.indices) {
            if (text[offset + i] != expected[i]) return false
        }
        return true
    }

    private fun isGoTemplateFile(file: PsiFile): Boolean {
        val name = file.viewProvider.virtualFile.name
        return name.endsWith(".tmpl")
    }
}
