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
 * Two independent layers used to insert a closing "}" for the same "{":
 *   1. The IDE's generic single-brace matcher autocloses a lone "{" into "{}".
 *   2. The TextMate `autoClosingPairs` declares "{{" -> "}}".
 *
 * When the user typed "{{", both fired and produced "{{}}}" with the caret in
 * the wrong place. Rather than try to patch up the buffer *after* the fact
 * (which only worked for one specific caret state), this handler takes full
 * control of typing "{" inside `.tmpl` files and ALWAYS returns
 * [Result.STOP]. Because we stop the typing action, neither the generic brace
 * matcher nor the TextMate pair ever runs, so no layer can double-insert.
 *
 * Behaviour:
 *   * Typing the *first* "{" inserts a single literal "{" (no autoclosed "}"),
 *     because a lone "{" is not a Go template delimiter.
 *   * Typing the *second* "{" (completing the "{{" action opener) produces
 *     "{{}}" with the caret between the delimiters, consuming any stray "}"
 *     that an earlier layer may already have inserted.
 */
class GoTemplateTypedHandler : TypedHandlerDelegate() {
    override fun beforeCharTyped(
        c: Char,
        project: Project,
        editor: Editor,
        file: PsiFile,
        fileType: FileType,
    ): Result {
        if (c != '{' && c != '-' && c != '*') return Result.CONTINUE
        if (!isGoTemplateFile(file)) return Result.CONTINUE
        // Leave selection-surrounding to the platform.
        if (editor.selectionModel.hasSelection()) return Result.CONTINUE

        val document = editor.document
        val caret = editor.caretModel.offset
        val text = document.charsSequence

        val prevChar = if (caret > 0) text[caret - 1] else null
        val nextChar = if (caret < text.length) text[caret] else null
        val charBeforePrev = if (caret >= 2) text[caret - 2] else null
        val charAfterNext = if (caret + 1 < text.length) text[caret + 1] else null

        if (c == '-') {
            // Same double-close bug as "{{": when the caret is sitting between
            // "{{" and "}}" (i.e. right after we auto-inserted "{{}}"), the
            // TextMate `{{- -}}` pair adds its own "-}}" on top of the existing
            // "}}", producing "{{--}}}}". Handle it ourselves so only one pair
            // is inserted.
            val insideEmptyDelimiter =
                prevChar == '{' && charBeforePrev == '{' &&
                    nextChar == '}' && charAfterNext == '}'
            if (!insideEmptyDelimiter) return Result.CONTINUE

            // Replace the trailing "}}" with "-  -}}" so the buffer becomes
            // "{{-  -}}" with the caret between the two spaces. The spaces are
            // part of Go template trim-marker syntax ("{{- x -}}") so we insert
            // them eagerly — this also sidesteps a follow-up double-close where
            // typing the space after "{{-" would trigger the TextMate
            // "{{- " -> " -}}" pair on top of our existing "}}".
            document.replaceString(caret, caret + 2, "-  -}}")
            editor.caretModel.moveToOffset(caret + 2)
            PsiDocumentManager.getInstance(project).commitDocument(document)
            return Result.STOP
        }

        if (c == '*') {
            // Same double-close bug for comments: after "{{|}}" the user types
            // "/" (buffer becomes "{{/|}}"), then "*". The TextMate `{{/*` pair
            // adds "*/}}" on top of the existing "}}", producing "{{/**/}}}}".
            val charBeforeBeforePrev = if (caret >= 3) text[caret - 3] else null
            val insideCommentOpener =
                prevChar == '/' && charBeforePrev == '{' && charBeforeBeforePrev == '{' &&
                    nextChar == '}' && charAfterNext == '}'
            if (insideCommentOpener) {
                // Replace the trailing "}}" with "**/}}" so the buffer becomes
                // "{{/**/}}" with the caret between the two stars.
                document.replaceString(caret, caret + 2, "**/}}")
                editor.caretModel.moveToOffset(caret + 1)
                PsiDocumentManager.getInstance(project).commitDocument(document)
                return Result.STOP
            }

            // Trim-marker variant: caret sits in "{{- /|  -}}" (i.e. the user
            // typed "{{-" then " /" inside the auto-inserted "{{-  -}}"). We
            // want the buffer to become "{{- /*  */ -}}" with the caret
            // between the two spaces of the comment body.
            val fiveBack = if (caret >= 5) text.subSequence(caret - 5, caret).toString() else null
            val fourAhead = if (caret + 4 <= text.length) text.subSequence(caret, caret + 4).toString() else null
            if (fiveBack == "{{- /" && fourAhead == " -}}") {
                document.replaceString(caret, caret + 4, "*  */ -}}")
                editor.caretModel.moveToOffset(caret + 2)
                PsiDocumentManager.getInstance(project).commitDocument(document)
                return Result.STOP
            }

            return Result.CONTINUE
        }

        // Are we completing a fresh "{{" delimiter? Only when the char right
        // before the caret is "{" and the one before that is NOT "{" (so we do
        // not turn "{{" into "{{{...}}}" when typing a third brace).
        val completingDelimiter = prevChar == '{' && charBeforePrev != '{'

        if (completingDelimiter) {
            // We want the buffer to become "{{}}" with the caret between the
            // delimiters. Consume a stray autoclosed "}" if one is already
            // sitting under the caret so we don't end up with "{{}}}".
            val end = if (nextChar == '}') caret + 1 else caret
            document.replaceString(caret, end, "{}}")
            editor.caretModel.moveToOffset(caret + 1)
        } else {
            // Lone opening brace: insert it literally and suppress the generic
            // single-brace autoclose so we never get an unwanted "}".
            document.insertString(caret, "{")
            editor.caretModel.moveToOffset(caret + 1)
        }

        PsiDocumentManager.getInstance(project).commitDocument(document)
        return Result.STOP
    }

    private fun isGoTemplateFile(file: PsiFile): Boolean {
        val name = file.viewProvider.virtualFile.name
        return name.endsWith(".tmpl")
    }
}
