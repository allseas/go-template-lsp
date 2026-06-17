# JetBrains Plugin Configuration

See [Configuration](../configuration.md) for the general configuration overview and option descriptions. This document covers JetBrains-specific implementation details.

## Architecture

Configuration is split into two levels:

| Level       | Class             | Storage                                          |
|-------------|-------------------|--------------------------------------------------|
| Application | `AppSettings`     | `GoTextTemplateSettings.xml` (global IDE config) |
| Project     | `ProjectSettings` | `.idea/goTextTemplateSettings.xml`               |

Project-level settings override application-level defaults. If a project setting is `null`, the application-level value is used.

### Settings UI

The plugin settings UI uses Kotlin UI DSL (`com.intellij.ui.dsl.builder`) via `BoundConfigurable`.

## How to Add a New Configuration Option

### 1. Add the field to `AppSettings.State`

In `AppSettings.kt`, add the new field to the `State` data class with a default value:

```kotlin
data class State(
    var enableHover: Boolean = true,
    var enableDefinition: Boolean = true,
    var enableDiagnostics: Boolean = true,
    var diagnostics: MutableMap<String, String> = mutableMapOf(
        "syntaxError" to "error",
        "invalidField" to "error",
        // ... other entries ...
    ),
    var enableAutocompletion: Boolean = true,
    var traceServer: TraceLevel = TraceLevel.MESSAGES,
    var pipeChainCompletion: ChainLevel = ChainLevel.FULL,
    var myNewOption: String = "default",  // <-- add here
)
```

### 2. Add a nullable override in `ProjectSettings.State`

In `ProjectSettings.kt`, add a nullable version so projects can optionally override. For simple scalar fields, add a nullable field; for the `diagnostics` map the project-level `diagnosticsOverride` map is merged on top of app defaults.

```kotlin
data class State(
    var enableHoverOverride: Boolean? = null,
    var enableDefinitionOverride: Boolean? = null,
    var enableDiagnosticsOverride: Boolean? = null,
    var diagnosticsOverride: MutableMap<String, String> = mutableMapOf(),
    var enableAutocompletionOverride: Boolean? = null,
    var traceServerOverride: AppSettings.TraceLevel? = null,
    var chainServerOverride: AppSettings.ChainLevel? = null,
    var myNewOptionOverride: String? = null,  // <-- add here
)
```

Then update `getEffectiveState()` to merge the override:

```kotlin
fun getEffectiveState(): AppSettings.State {
    val appState = AppSettings.getInstance().state
    return AppSettings.State(
        enableHover = state.enableHoverOverride ?: appState.enableHover,
        enableDefinition = state.enableDefinitionOverride ?: appState.enableDefinition,
        enableDiagnostics = state.enableDiagnosticsOverride ?: appState.enableDiagnostics,
        diagnostics = (appState.diagnostics + state.diagnosticsOverride).toMutableMap(),
        enableAutocompletion = state.enableAutocompletionOverride ?: appState.enableAutocompletion,
        traceServer = state.traceServerOverride ?: appState.traceServer,
        myNewOption = state.myNewOptionOverride ?: appState.myNewOption,  // <-- add here
    )
}
```

### 3. Add UI controls

**Application-level** - in `AppSettingsConfigurable.kt`:

```kotlin
override fun createPanel() = panel {
    val settings = AppSettings.getInstance()
    // ... existing rows ...
    row("My new option:") {
        textField()
            .bindText(settings.state::myNewOption)
    }
}
```

**Project-level** - in `ProjectSettingsConfigurable.kt`:

```kotlin
override fun createPanel() = panel {
    val settings = ProjectSettings.getInstance(project)
    // ... existing rows ...
    row("My new option:") {
        textField()
            .bindText({ settings.state.myNewOptionOverride ?: "" }, { settings.state.myNewOptionOverride = it.ifEmpty { null } })
            .comment("Leave empty to use the application-level default")
    }
}
```

### 4. Send the new option to the LSP server

In `TextTemplateLspLanguageClient.kt`, add the field to the JSON:

```kotlin
override fun createSettings(): Any {
    val config = ProjectSettings.getInstance(project).getEffectiveState()
    val settings = JsonObject().apply {
        addProperty("enableHover", config.enableHover)
        addProperty("enableDefinition", config.enableDefinition)
        addProperty("enableDiagnostics", config.enableDiagnostics)
        add("diagnostics", JsonObject().apply {
            config.diagnostics.forEach { (key, value) ->
                addProperty(key, value)
            }
        })
        addProperty("enableAutocompletion", config.enableAutocompletion)
        addProperty("myNewOption", config.myNewOption)  // <-- add here
        add("trace", JsonObject().apply {
            addProperty("server", config.traceServer.value)
        })
    }
    // Settings must be wrapped under "goTmplSupport" so lsp4ij can find them
    // when the server requests workspace/configuration for section "goTmplSupport".
    return JsonObject().apply {
        add("goTmplSupport", settings)
    }
}
```
