package com.allseas.gotexttemplate

import com.intellij.codeInsight.editorActions.TypedHandlerDelegate
import com.intellij.openapi.editor.Editor
import com.intellij.openapi.fileTypes.FileType
import com.intellij.openapi.project.Project
import com.intellij.psi.PsiDocumentManager
import com.intellij.psi.PsiFile

/**
 * Fixes the "{{}}}" autoclose bug for Go text/template files.
 *
 * When the user types "{{" in a .tmpl file, the IDE sometimes autocloses
 * the single "{" into "{}" (generic brace matching) AND then the TextMate
 * pair "{{" -> "}}" also fires, producing "{{}}}" with the cursor inside.
 *
 * This handler intercepts the second "{" when the buffer is in the state
 * "{|}" (caret between an existing "{" and the unwanted "}") and rewrites
 * it to "{{|}}" deterministically.
 */
class GoTemplateTypedHandler : TypedHandlerDelegate() {
    override fun beforeCharTyped(
        c: Char,
        project: Project,
        editor: Editor,
        file: PsiFile,
        fileType: FileType,
    ): Result {
        if (c != '{') return Result.CONTINUE
        if (!isGoTemplateFile(file)) return Result.CONTINUE

        val caret = editor.caretModel.offset
        val document = editor.document
        val text = document.charsSequence

        if (caret <= 0 || caret >= text.length) return Result.CONTINUE
        if (text[caret - 1] != '{') return Result.CONTINUE
        if (text[caret] != '}') return Result.CONTINUE

        // Replace the unwanted single "}" with "{}}", landing the caret
        // between the resulting "{{" and "}}".
        document.replaceString(caret, caret + 1, "{}}")
        editor.caretModel.moveToOffset(caret + 1)
        PsiDocumentManager.getInstance(project).commitDocument(document)
        return Result.STOP
    }

    private fun isGoTemplateFile(file: PsiFile): Boolean {
        val name = file.viewProvider.virtualFile.name
        return name.endsWith(".tmpl")
    }
}
