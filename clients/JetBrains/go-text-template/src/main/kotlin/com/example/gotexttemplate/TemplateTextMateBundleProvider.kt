package com.example.gotexttemplate

import com.intellij.openapi.application.PluginPathManager
import com.intellij.openapi.diagnostic.thisLogger
import org.jetbrains.plugins.textmate.api.TextMateBundleProvider
import java.io.File

class TemplateTextMateBundleProvider : TextMateBundleProvider {
    override fun getBundles(): List<TextMateBundleProvider.PluginBundle> {
        val directory: File? = PluginPathManager.getPluginResource(this.javaClass, "textmate/go-text-template")
        if (directory == null) {
            thisLogger().warn("Could not find the text/template TextMate bundle")
            return mutableListOf()
        }
        return mutableListOf(TextMateBundleProvider.PluginBundle("Go text/template", directory.toPath()))
    }
}
