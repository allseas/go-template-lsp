package com.allseas.gotexttemplate

import com.intellij.openapi.options.BoundConfigurable
import com.intellij.ui.dsl.builder.bindItem
import com.intellij.ui.dsl.builder.bindSelected
import com.intellij.ui.dsl.builder.panel
import com.intellij.ui.dsl.builder.toNullableProperty

class AppSettingsConfigurable : BoundConfigurable("Go Text Template Support") {
    override fun createPanel() =
        panel {
            val settings = AppSettings.getInstance()

            group("Features") {
                row {
                    checkBox("Enable hover information")
                        .bindSelected(settings.state::enableHover)
                }
                row {
                    checkBox("Enable go-to-definition")
                        .bindSelected(settings.state::enableDefinition)
                }
                row {
                    checkBox("Enable autocompletion")
                        .bindSelected(settings.state::enableAutocompletion)
                }
            }
            group("Diagnostics") {
                row {
                    checkBox("Enable diagnostics")
                        .bindSelected(settings.state::enableDiagnostics)
                }
                row {
                    checkBox("Report syntax errors")
                        .bindSelected(settings.state::diagnosticsSyntaxError)
                }
                row {
                    checkBox("Report duplicate variable declarations")
                        .bindSelected(settings.state::diagnosticsVariableRedeclaration)
                }
                row {
                    checkBox("Report unknown or incorrectly used functions")
                        .bindSelected(settings.state::diagnosticsIncorrectFunction)
                }
            }
            group("Advanced") {
                row("Trace level:") {
                    comboBox(AppSettings.TraceLevel.entries)
                        .bindItem(settings.state::traceServer.toNullableProperty())
                }
            }
        }
}
