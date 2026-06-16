package com.allseas.gotexttemplate

import com.intellij.openapi.options.BoundConfigurable
import com.intellij.openapi.project.Project
import com.intellij.ui.SimpleListCellRenderer
import com.intellij.ui.dsl.builder.bindItem
import com.intellij.ui.dsl.builder.panel
import javax.swing.JList

class ProjectSettingsConfigurable(
    private val project: Project,
) : BoundConfigurable("Go Text Template Support") {
    override fun createPanel() =
        panel {
            val settings = ProjectSettings.getInstance(project)

            group("Features") {
                row("Enable hover information:") {
                    comboBox(listOf(null, true, false), NullableBooleanRenderer())
                        .bindItem(settings.state::enableHoverOverride)
                        .comment("Leave empty to use the application-level default")
                }
                row("Enable go-to-definition:") {
                    comboBox(listOf(null, true, false), NullableBooleanRenderer())
                        .bindItem(settings.state::enableDefinitionOverride)
                        .comment("Leave empty to use the application-level default")
                }
                row("Enable autocompletion:") {
                    comboBox(listOf(null, true, false), NullableBooleanRenderer())
                        .bindItem(settings.state::enableAutocompletionOverride)
                        .comment("Leave empty to use the application-level default")
                }
            }
            group("Diagnostics") {
                row("Enable diagnostics:") {
                    comboBox(listOf(null, true, false), NullableBooleanRenderer())
                        .bindItem(settings.state::enableDiagnosticsOverride)
                        .comment("Leave empty to use the application-level default")
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
                        "unknownType" to "Unknown type",
                        "hintLoadFailure" to "Type hint load failure",
                        "unknownRangeType" to "Unknown range type",
                        "emptyDefineName" to "Empty define name",
                    )
                for ((key, label) in diagnosticEntries) {
                    val k = key
                    row("$label:") {
                        comboBox(listOf(null) + AppSettings.DiagnosticSeverity.entries, NullableDiagnosticSeverityRenderer())
                            .bindItem(
                                { settings.state.diagnosticsOverride[k]?.let { AppSettings.DiagnosticSeverity.fromValue(it) } },
                                { v ->
                                    if (v !=
                                        null
                                    ) {
                                        settings.state.diagnosticsOverride[k] = v.value
                                    } else {
                                        settings.state.diagnosticsOverride.remove(k)
                                    }
                                },
                            ).comment("Leave empty to use the application-level default")
                    }
                }
            }
            group("Advanced") {
                row("Trace level:") {
                    comboBox(listOf(null) + AppSettings.TraceLevel.entries, NullableTraceLevelRenderer())
                        .bindItem(settings.state::traceServerOverride)
                        .comment("Leave empty to use the application-level default")
                }
            }
        }
}

private class NullableBooleanRenderer : SimpleListCellRenderer<Boolean?>() {
    override fun customize(
        list: JList<out Boolean?>,
        value: Boolean?,
        index: Int,
        selected: Boolean,
        hasFocus: Boolean,
    ) {
        text =
            when (value) {
                null -> "(use default)"
                true -> "Enabled"
                false -> "Disabled"
            }
    }
}

private class NullableTraceLevelRenderer : SimpleListCellRenderer<AppSettings.TraceLevel?>() {
    override fun customize(
        list: JList<out AppSettings.TraceLevel?>,
        value: AppSettings.TraceLevel?,
        index: Int,
        selected: Boolean,
        hasFocus: Boolean,
    ) {
        text = value?.value ?: "(use default)"
    }
}

private class NullableDiagnosticSeverityRenderer : SimpleListCellRenderer<AppSettings.DiagnosticSeverity?>() {
    override fun customize(
        list: JList<out AppSettings.DiagnosticSeverity?>,
        value: AppSettings.DiagnosticSeverity?,
        index: Int,
        selected: Boolean,
        hasFocus: Boolean,
    ) {
        text = value?.value ?: "(use default)"
    }
}
