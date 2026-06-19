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
                val diagnosticEntries =
                    listOf(
                        "syntaxError" to "Syntax errors",
                        "invalidField" to "Invalid field access",
                        "invalidFunction" to "Invalid function call",
                        "invalidCommand" to "Invalid command",
                        "invalidRange" to "Invalid range",
                        "invalidIf" to "Invalid if condition",
                        "invalidWith" to "Invalid with expression",
                        "undeclaredVariable" to "Undeclared variable",
                        "doubleDeclaredVariable" to "Duplicate variable declaration",
                        "invalidTemplateArg" to "Invalid template argument",
                        "argumentNumberMismatch" to "Argument count mismatch",
                        "unknownType" to "Unknown type",
                        "hintLoadFailure" to "Type hint load failure",
                        "unknownRangeType" to "Unknown range type",
                        "emptyDefineName" to "Empty define name",
                    )
                for ((key, label) in diagnosticEntries) {
                    val k = key
                    row("$label:") {
                        comboBox(AppSettings.DiagnosticSeverity.entries)
                            .bindItem(
                                { AppSettings.DiagnosticSeverity.fromValue(settings.state.diagnostics[k] ?: "error") },
                                { v -> settings.state.diagnostics[k] = v?.value ?: "error" },
                            )
                    }
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
