package com.example.gotexttemplate

import com.intellij.openapi.project.Project
import com.redhat.devtools.lsp4ij.LanguageServerFactory
import com.redhat.devtools.lsp4ij.client.LanguageClientImpl
import com.redhat.devtools.lsp4ij.server.StreamConnectionProvider

class TextTemplateLspServerFactory : LanguageServerFactory {
    override fun createConnectionProvider(project: Project): StreamConnectionProvider = TextTemplateLspServerConnectionProvider(project)

    override fun createLanguageClient(project: Project): LanguageClientImpl = TextTemplateLspLanguageClient(project)
}
