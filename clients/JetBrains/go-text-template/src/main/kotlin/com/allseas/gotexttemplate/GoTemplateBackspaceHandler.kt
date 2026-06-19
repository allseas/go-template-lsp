package com.allseas.gotexttemplate
import com.intellij.codeInsight.editorActions.BackspaceHandlerDelegate
import com.intellij.openapi.editor.Editor
import com.intellij.psi.PsiDocumentManager
import com.intellij.psi.PsiFile

/**
 * Mirror image of [GoTemplateTypedHandler]: when the user backspaces inside an
 * auto-inserted Go template delimiter pair `{{|}}`, all four braces should
 * disappear so that insert (`{{`) and delete (one backspace) are exact
 * inverses of each other.
 *
 * The IntelliJ platform's default backspace machinery only knows how to remove
 * the single closer that the brace matcher inserted for the opener under the
 * caret. For the nested `{{ ... }}` pair we synthesize ourselves in
 * [GoTemplateTypedHandler], the platform leaves the outer `{` and the trailing
 * `}}` behind, producing the broken state `{|}}` after one backspace.
 *
 * We fix that by handling the post-deletion step ourselves when, and *only*
 * when, the surrounding text looks exactly like a freshly auto-paired empty
 * delimiter:
 *   * before the platform deletes the inner `{`, the buffer reads `{{|}}`
 *     (`text[caret-2]=='{' && text[caret-1]=='{' && text[caret]=='}' &&
 *      text[caret+1]=='}'`), and
 *   * neither side is part of a longer `{{{` / `}}}` run (those occur in
 *     legitimate user code such as `{{ printf "}}}" }}`).
 *
 * Outside of that exact shape both hooks fall through to the platform's
 * default behaviour, so normal single-brace backspace is preserved.
 *
 * The delegate is also scoped to `*.tmpl` files via [isGoTemplateFile] (the
 * same check used by [GoTemplateTypedHandler]) so it can never fire in
 * unrelated languages even though `backspaceHandlerDelegate` itself is a
 * non-language-scoped extension point.
 */
class GoTemplateBackspaceHandler : BackspaceHandlerDelegate() {
    override fun beforeCharDeleted(
        c: Char,
        file: PsiFile,
        editor: Editor,
    ) {
        // Nothing to do: we cannot safely mutate the document here without
        // confusing the platform's own caret bookkeeping. All cleanup is
        // performed in [charDeleted], which runs after the platform has
        // removed the inner `{` under the caret.
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
        if (c != '{') return false
        if (!isGoTemplateFile(file)) return false
        if (editor.selectionModel.hasSelection()) return false
        val document = editor.document
        val caret = editor.caretModel.offset
        val text = document.charsSequence
        // At this point the platform has already removed the inner `{`. If the
        // original buffer was `{{|}}` (caret offset N, with `{` at N-2/N-1 and
        // `}` at N/N+1), the caret has moved to N-1 and the buffer now reads
        // `{|}}` around it:
        //   text[caret - 1] == '{'   (the outer opener that survived)
        //   text[caret    ] == '}'   (inner closer, originally at N)
        //   text[caret + 1] == '}'   (outer closer, originally at N+1)
        if (caret < 1 || caret + 1 >= text.length) return false
        if (text[caret - 1] != '{') return false
        if (text[caret] != '}') return false
        if (text[caret + 1] != '}') return false
        // Guard against longer brace runs that are not a freshly auto-paired
        // empty delimiter: e.g. `{{{|}}` (third literal `{` before) or
        // `{{|}}}` (third literal `}` after) -- leave those to the default
        // single-char backspace behaviour.
        if (caret >= 2 && text[caret - 2] == '{') return false
        if (caret + 2 < text.length && text[caret + 2] == '}') return false
        // Wipe the surviving outer `{` and the trailing `}}` so the four-brace
        // pair vanishes as a whole.
        document.deleteString(caret - 1, caret + 2)
        editor.caretModel.moveToOffset(caret - 1)
        PsiDocumentManager.getInstance(file.project).commitDocument(document)
        return true
    }

    private fun isGoTemplateFile(file: PsiFile): Boolean {
        val name = file.viewProvider.virtualFile.name
        return name.endsWith(".tmpl")
    }
}
