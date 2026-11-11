package com.allseas.gotexttemplate

import com.intellij.codeInsight.completion.CompletionContributor
import com.intellij.codeInsight.completion.CompletionParameters
import com.intellij.codeInsight.completion.CompletionResult
import com.intellij.codeInsight.completion.CompletionResultSet
import com.intellij.codeInsight.completion.CompletionSorter
import com.intellij.codeInsight.completion.PrioritizedLookupElement
import com.intellij.codeInsight.lookup.LookupElement
import com.intellij.codeInsight.lookup.LookupElementWeigher
import com.redhat.devtools.lsp4ij.LanguageServersRegistry

/**
 * Runs before LSP4IJ and re-sorts its items purely by the server's sortText order.
 * LSP4IJ encodes that order as a [PrioritizedLookupElement] priority, but IntelliJ's default
 * sorter ranks start-matches first (e.g. `call` above `.CustomerName` for the prefix `C`).
 */
class GoTemplateCompletionSorter : CompletionContributor() {
    override fun fillCompletionVariants(
        parameters: CompletionParameters,
        result: CompletionResultSet,
    ) {
        if (!LanguageServersRegistry.getInstance().isFileSupported(parameters.originalFile)) {
            return
        }
        val sorter = CompletionSorter.emptySorter().weigh(ServerOrderWeigher)
        result.runRemainingContributors(parameters) { r ->
            CompletionResult.wrap(r.lookupElement, r.prefixMatcher, sorter)?.let(result::passResult)
        }
    }

    private object ServerOrderWeigher : LookupElementWeigher("goTemplateServerOrder", true, false) {
        override fun weigh(element: LookupElement): Comparable<*> =
            element.`as`(PrioritizedLookupElement.CLASS_CONDITION_KEY)?.priority ?: 0.0
    }
}
