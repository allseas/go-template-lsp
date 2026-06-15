package com.allseas.gotexttemplate

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage

@Service(Service.Level.APP)
@State(name = "com.example.gotexttemplate.AppSettings", storages = [Storage("GoTextTemplateSettings.xml")])
class AppSettings : PersistentStateComponent<AppSettings.State> {
    data class State(
        var enableHover: Boolean = true,
        var enableDefinition: Boolean = true,
        var enableDiagnostics: Boolean = true,
        var diagnosticsSyntaxError: Boolean = true,
        var diagnosticsVariableRedeclaration: Boolean = true,
        var diagnosticsIncorrectFunction: Boolean = true,
        var enableAutocompletion: Boolean = true,
        var traceServer: TraceLevel = TraceLevel.MESSAGES,
    )

    enum class TraceLevel(
        val value: String,
    ) {
        OFF("off"),
        MESSAGES("messages"),
        VERBOSE("verbose"),
        ;

        companion object {
            fun fromValue(value: String): TraceLevel = entries.firstOrNull { it.value == value } ?: MESSAGES
        }
    }

    private var state = State()

    override fun getState(): State = state

    override fun loadState(state: State) {
        this.state = state
    }

    companion object {
        fun getInstance(): AppSettings = ApplicationManager.getApplication().getService(AppSettings::class.java)
    }
}
