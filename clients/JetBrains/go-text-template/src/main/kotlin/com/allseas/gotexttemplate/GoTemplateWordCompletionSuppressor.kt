package com.allseas.gotexttemplate

import com.intellij.codeInsight.completion.CompletionContributor
import com.intellij.codeInsight.completion.CompletionParameters
import com.intellij.codeInsight.completion.CompletionResultSet
import com.redhat.devtools.lsp4ij.LanguageServersRegistry

/**
 * Completion contributor for the TextMate language that IntelliJ assigns to Go
 * text/template (`*.tmpl`) files (the file type LSP4IJ is mapped to via the
 * `fileNamePatternMapping patterns="*.tmpl"` extension in plugin.xml).
 *
 * LSP4IJ already registers its own `LSPCompletionContributor` (`language="any"`,
 * `order="first, before wordCompletion"`) which provides the language-server items.
 * The problem it leaves behind is that it never stops the completion chain, so when
 * the server returns nothing the platform's `WordCompletionContributor`
 * ("Suggest words from the editor") kicks in with noisy editor-word suggestions.
 *
 * This contributor is registered to run *after* LSP4IJ's contributor but *before*
 * `wordCompletion`. It does not add any items itself (so LSP items are never
 * duplicated); it simply calls [CompletionResultSet.stopHere] to halt the chain
 * before `WordCompletionContributor` runs.
 */
class GoTemplateWordCompletionSuppressor : CompletionContributor() {
    override fun fillCompletionVariants(
        parameters: CompletionParameters,
        result: CompletionResultSet,
    ) {
        val psiFile = parameters.originalFile

        // Only suppress word completion for files actually backed by an LSP server
        // (our *.tmpl mapping). Other TextMate files keep their default behavior.
        if (!LanguageServersRegistry.getInstance().isFileSupported(psiFile)) {
            return
        }

        // LSP4IJ's LSPCompletionContributor has already contributed its items at this
        // point. Stop the chain so WordCompletionContributor does not fall back to
        // "words from the editor" (especially when the LSP response is empty).
        result.stopHere()
    }
}
