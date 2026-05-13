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

            row("Enable language server:") {
                comboBox(listOf(null, true, false), NullableBooleanRenderer())
                    .bindItem(settings.state::enableServerOverride)
                    .comment("Leave empty to use the application-level default")
            }
            row("Trace level:") {
                comboBox(listOf(null) + AppSettings.TraceLevel.entries, NullableTraceLevelRenderer())
                    .bindItem(settings.state::traceServerOverride)
                    .comment("Leave empty to use the application-level default")
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
