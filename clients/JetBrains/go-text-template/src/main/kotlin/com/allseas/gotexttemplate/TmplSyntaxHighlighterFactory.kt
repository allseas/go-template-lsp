package com.allseas.gotexttemplate

import com.intellij.openapi.diagnostic.thisLogger
import com.intellij.openapi.fileTypes.SyntaxHighlighter
import com.intellij.openapi.fileTypes.SyntaxHighlighterFactory
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.textmate.joni.JoniRegexFactory
import org.jetbrains.plugins.textmate.TextMateService
import org.jetbrains.plugins.textmate.language.syntax.highlighting.TextMateHighlighter
import org.jetbrains.plugins.textmate.language.syntax.lexer.TextMateHighlightingLexer
import org.jetbrains.plugins.textmate.language.syntax.lexer.TextMateSyntaxMatcherImpl
import org.jetbrains.plugins.textmate.language.syntax.lexer.caching
import org.jetbrains.plugins.textmate.language.syntax.selector.TextMateSelectorWeigherImpl
import org.jetbrains.plugins.textmate.language.syntax.selector.caching
import org.jetbrains.plugins.textmate.regex.CaffeineCachingRegexProvider
import org.jetbrains.plugins.textmate.regex.RememberingLastMatchRegexFactory

class TmplSyntaxHighlighterFactory : SyntaxHighlighterFactory() {
    override fun getSyntaxHighlighter(
        project: Project?,
        virtualFile: VirtualFile?,
    ): SyntaxHighlighter {
        thisLogger().warn("TmplSyntaxHighlighterFactory called for: ${virtualFile?.name}")
        val service = TextMateService.getInstance()
        val byFile = virtualFile?.let { service?.getLanguageDescriptorByFileName(it.name) }
        val byExt = service?.getLanguageDescriptorByExtension("tmpl")
        thisLogger().warn("descriptor by filename: $byFile, by extension: $byExt")
        val descriptor = byFile ?: byExt
        if (descriptor != null) {
            return TextMateHighlighter(TextMateHighlightingLexer(descriptor, syntaxMatcher, 20000))
        }
        thisLogger().warn("No TextMate descriptor found, returning plain highlighter")
        return TextMateHighlighter(null)
    }

    companion object {
        private val regexProvider = CaffeineCachingRegexProvider(RememberingLastMatchRegexFactory(JoniRegexFactory()))
        private val weigher = TextMateSelectorWeigherImpl().caching()
        private val syntaxMatcher = TextMateSyntaxMatcherImpl(regexProvider, weigher).caching()
    }
}
