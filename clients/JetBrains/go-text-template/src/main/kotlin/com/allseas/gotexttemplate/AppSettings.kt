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
        var diagnostics: MutableMap<String, String> =
            mutableMapOf(
                "invalidField" to "error",
                "invalidFunction" to "warning",
                "invalidCommand" to "error",
                "invalidRange" to "error",
                "invalidIf" to "error",
                "invalidWith" to "error",
                "undeclaredVariable" to "error",
                "doubleDeclaredVariable" to "warning",
                "invalidTemplateArg" to "error",
                "argumentNumberMismatch" to "error",
                "unknownType" to "information",
                "syntaxError" to "error",
                "hintLoadFailure" to "warning",
                "unknownRangeType" to "warning",
                "emptyDefineName" to "warning",
            ),
        var enableAutocompletion: Boolean = true,
        var traceServer: TraceLevel = TraceLevel.MESSAGES,
        var chainServer: ChainLevel = ChainLevel.FULL
    )

    enum class DiagnosticSeverity(
        val value: String,
    ) {
        DISABLED("disabled"),
        ERROR("error"),
        WARNING("warning"),
        INFORMATION("information"),
        HINT("hint"),
        ;

        companion object {
            fun fromValue(value: String): DiagnosticSeverity = entries.firstOrNull { it.value == value } ?: ERROR
        }
    }

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

    enum class ChainLevel(
        val value: String,
    ) {
        OFF("off"),
        FULL("full"),
        STEP("step"),
        ;

        companion object {
            fun fromValue(value: String): ChainLevel = entries.firstOrNull { it.value == value } ?: FULL
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
