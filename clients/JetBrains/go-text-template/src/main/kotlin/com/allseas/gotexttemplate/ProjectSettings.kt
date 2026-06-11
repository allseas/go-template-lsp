package com.allseas.gotexttemplate

import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.openapi.project.Project

@Service(Service.Level.PROJECT)
@State(name = "com.example.gotexttemplate.ProjectSettings", storages = [Storage("goTextTemplateSettings.xml")])
class ProjectSettings : PersistentStateComponent<ProjectSettings.State> {
    data class State(
        var enableHoverOverride: Boolean? = null,
        var enableDefinitionOverride: Boolean? = null,
        var enableDiagnosticsOverride: Boolean? = null,
        var diagnosticsSyntaxErrorOverride: Boolean? = null,
        var diagnosticsVariableRedeclarationOverride: Boolean? = null,
        var diagnosticsIncorrectFunctionOverride: Boolean? = null,
        var enableAutocompletionOverride: Boolean? = null,
        var traceServerOverride: AppSettings.TraceLevel? = null,
    )

    private var state = State()

    override fun getState(): State = state

    override fun loadState(state: State) {
        this.state = state
    }

    /**
     * Returns the effective configuration by merging application-level settings
     * with project-level overrides. Project settings take precedence.
     */
    fun getEffectiveState(): AppSettings.State {
        val appState = AppSettings.getInstance().state
        return AppSettings.State(
            enableHover = state.enableHoverOverride ?: appState.enableHover,
            enableDefinition = state.enableDefinitionOverride ?: appState.enableDefinition,
            enableDiagnostics = state.enableDiagnosticsOverride ?: appState.enableDiagnostics,
            diagnosticsSyntaxError = state.diagnosticsSyntaxErrorOverride ?: appState.diagnosticsSyntaxError,
            diagnosticsVariableRedeclaration = state.diagnosticsVariableRedeclarationOverride ?: appState.diagnosticsVariableRedeclaration,
            diagnosticsIncorrectFunction = state.diagnosticsIncorrectFunctionOverride ?: appState.diagnosticsIncorrectFunction,
            enableAutocompletion = state.enableAutocompletionOverride ?: appState.enableAutocompletion,
            traceServer = state.traceServerOverride ?: appState.traceServer,
        )
    }

    companion object {
        fun getInstance(project: Project): ProjectSettings = project.getService(ProjectSettings::class.java)
    }
}
