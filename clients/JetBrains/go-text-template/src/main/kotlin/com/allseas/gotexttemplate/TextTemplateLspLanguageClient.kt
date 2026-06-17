package com.allseas.gotexttemplate

import com.google.gson.JsonObject
import com.intellij.openapi.project.Project
import com.redhat.devtools.lsp4ij.client.LanguageClientImpl

class TextTemplateLspLanguageClient(
    project: Project,
) : LanguageClientImpl(project) {
    override fun createSettings(): Any {
        val config = ProjectSettings.getInstance(project).getEffectiveState()
        val settings =
            JsonObject().apply {
                addProperty("enableHover", config.enableHover)
                addProperty("enableDefinition", config.enableDefinition)
                addProperty("enableDiagnostics", config.enableDiagnostics)
                add(
                    "diagnostics",
                    JsonObject().apply {
                        config.diagnostics.forEach { (key, value) ->
                            addProperty(key, value)
                        }
                    },
                )
                addProperty("enableAutocompletion", config.enableAutocompletion)
                add(
                    "trace",
                    JsonObject().apply {
                        addProperty("server", config.traceServer.value)
                    },
                )
                addProperty("pipeChainCompletion", config.pipeChainCompletion)
            }
        return JsonObject().apply {
            add("goTmplSupport", settings)
        }
    }
}
