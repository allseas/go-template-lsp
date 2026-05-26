package com.allseas.gotexttemplate

import com.google.gson.JsonObject
import com.intellij.openapi.project.Project
import com.redhat.devtools.lsp4ij.client.LanguageClientImpl

class TextTemplateLspLanguageClient(
    project: Project,
) : LanguageClientImpl(project) {
    override fun createSettings(): Any {
        val config = ProjectSettings.getInstance(project).getEffectiveState()
        val json =
            JsonObject().apply {
                addProperty("enableServer", config.enableServer)
                add(
                    "trace",
                    JsonObject().apply {
                        addProperty("server", config.traceServer.value)
                    },
                )
            }
        return json
    }
}
