package com.example.gotexttemplate

import com.intellij.openapi.options.BoundConfigurable
import com.intellij.ui.dsl.builder.bindItem
import com.intellij.ui.dsl.builder.bindSelected
import com.intellij.ui.dsl.builder.panel
import com.intellij.ui.dsl.builder.toNullableProperty

class AppSettingsConfigurable : BoundConfigurable("Go Text Template Support") {
    override fun createPanel() =
        panel {
            val settings = AppSettings.getInstance()

            row {
                checkBox("Enable language server")
                    .bindSelected(settings.state::enableServer)
            }
            row("Trace level:") {
                comboBox(AppSettings.TraceLevel.entries)
                    .bindItem(settings.state::traceServer.toNullableProperty())
            }
        }
}
