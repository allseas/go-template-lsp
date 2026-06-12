# JetBrains Plugin Architecture

This document describes the architecture of the JetBrains plugin and provides guides for extending it with new features.

## Overview

The JetBrains plugin integrates text/template language support into JetBrains IDEs (IntelliJ IDEA, PyCharm, GoLand, etc.) via the Language Server Protocol. It communicates with the shared Go language server to provide IDE features across the entire JetBrains ecosystem.

## Architecture

### Directory Structure

```files
clients/JetBrains/go-text-template/
    │
    ├─ build.gradle.kts ............. Build configuration
    ├─ gradle.properties ............. Gradle properties
    ├─ settings.gradle.kts ........... Settings
    ├─ gradle/ ....................... Gradle wrapper
    └─ src/
        ├─ main/
        │   ├─ kotlin/ ............... Main plugin code
        │   │   └─ com/example/gotexttemplate/
        │   │       ├─ GoTextTemplatePlugin.kt
        │   │       ├─ TemplateTextMateBundleProvider.kt
        │   │       ├─ GoTextTemplateContext.kt
        │   │       ├─ TextTemplateLspServerFactory.kt
        │   │       ├─ settings/
        │   │       │   ├─ AppSettings.kt
        │   │       │   └─ ProjectSettings.kt
        │   │       └─ MyToolWindowFactory.kt
        │   └─ resources/
        │       ├─ META-INF/
        │       │   └─ plugin.xml ... Plugin manifest
        │       ├─ messages/
        │       │   └─ MyMessageBundle.properties
        │       └─ liveTemplates/
        │           └─ GoTemplate.xml
        └─ test/
            ├─ kotlin/ .............. Test code
            └─ resources/ ........... Test resources
```

### Key Components

#### Plugin Manifest (`plugin.xml`)

Located at `src/main/resources/META-INF/plugin.xml`, this file defines the plugin's basic information and integration points:

```xml
<!-- Plugin Configuration File: https://plugins.jetbrains.com/docs/intellij/plugin-configuration-file.html -->
<idea-plugin>
    <id>com.example.go-text-template</id>

    <name>Go-text-template</name>

    <description>Language support for go text/template package</description>
    <vendor/>

    <!-- Dependencies on other plugins and IDE modules -->
    <!-- ... -->

    <!-- Extensions defined by the plugin -->
    <extensions defaultExtensionNs="com.intellij">
        <!-- Tool window for UI components -->
        <toolWindow id="MyToolWindow" factoryClass="com.example.gotexttemplate.MyToolWindowFactory"
                    icon="AllIcons.Toolwindows.ToolWindowPalette"/>
        
        <!-- TextMate syntax highlighting support -->
        <textmate.bundleProvider implementation="com.example.gotexttemplate.TemplateTextMateBundleProvider" />

        <!-- Live templates (code snippets) -->
        <defaultLiveTemplates file="liveTemplates/GoTemplate.xml"/>
        <liveTemplateContext
                contextId="GOTMPL"
                implementation="com.example.gotexttemplate.GoTextTemplateContext"/>
    </extensions>
    
    <!-- LSP server integration (uses LSP4IJ plugin for actual LSP handling) -->
    <extensions defaultExtensionNs="com.redhat.devtools.lsp4ij">
        <server factoryClass="com.example.gotexttemplate.TextTemplateLspServerFactory" id="go-text-template-lsp"/>
        <fileNamePatternMapping patterns=".tmpl" serverId="go-text-template-lsp"/>
    </extensions>
</idea-plugin>
```

#### Settings Architecture

The plugin uses a two-level configuration system:

```kotlin
// Application-level settings (global IDE configuration)
@Service(Service.Level.APP)
class AppSettings : PersistentStateComponent<AppSettings.State> {
    data class State(
        var enableServer: Boolean = true,
        var traceServer: TraceLevel = TraceLevel.MESSAGES,
    )
}

// Project-level settings (per-project overrides)
@Service(Service.Level.PROJECT)
class ProjectSettings(val project: Project) : PersistentStateComponent<ProjectSettings.State> {
    data class State(
        var enableServerOverride: Boolean? = null,
        var traceServerOverride: AppSettings.TraceLevel? = null,
    )
    
    fun getEffectiveState(): AppSettings.State {
        val appState = AppSettings.getInstance().state
        return AppSettings.State(
            enableServer = state.enableServerOverride ?: appState.enableServer,
            traceServer = state.traceServerOverride ?: appState.traceServer,
        )
    }
}
```

This design allows:

- Global defaults that apply everywhere
- Per-project customization when needed
- Clean fallback behavior

#### LSP Server Integration (`TextTemplateLspServerFactory.kt`)

The plugin integrates with the language server via LSP4IJ (the LSP support library for JetBrains). The factory creates and configures the LSP server connection. Configuration is sent to the server during initialization and when settings change via the workspace configuration mechanism.

The actual LSP protocol communication is handled by LSP4IJ, which:

- Launches the server binary as a subprocess
- Manages stdio communication
- Handles document synchronization
- Displays diagnostics and completions from the server

## Design Decisions

### Kotlin over Java

Kotlin is used for the plugin because:

- **Modern language** - Better null safety and extension functions
- **Concise syntax** - Less boilerplate than Java
- **Standard in JetBrains** - All modern JetBrains plugins use Kotlin
- **IDE support** - Excellent tooling and refactoring in IntelliJ

### Gradle Build System

Gradle is used because:

- **Official** - The recommended build system for JetBrains plugins
- **gradle-intellij-plugin** - Automates many setup tasks
- **Plugin Verifier** - Built-in testing against multiple IDE versions

## Adding a New Feature

### Example: Adding a Code Inspection

Inspections detect problems in templates and suggest fixes.

#### 1. Create Inspection Class

In `src/main/kotlin/.../inspections/UnusedVariableInspection.kt`:

```kotlin
class UnusedVariableInspection : LocalInspectionTool() {
    override fun getShortName(): String = "UnusedTemplateVariable"
    
    override fun getDisplayName(): String = "Unused template variable"
    
    override fun getGroupDisplayName(): String = "Go text/template"
    
    override fun isEnabledByDefault(): Boolean = true
    
    override fun buildVisitor(
        holder: ProblemsHolder,
        isOnTheFly: Boolean,
        session: LocalInspectionToolSession
    ): PsiElementVisitor {
        return object : PsiElementVisitor() {
            override fun visitElement(element: PsiElement) {
                // Check for unused variables
                if (isUnused(element)) {
                    holder.registerProblem(
                        element,
                        "Variable '${element.text}' is never used",
                        ProblemHighlightType.WARNING,
                        FixUnusedVariableName()
                    )
                }
            }
        }
    }
}

private class FixUnusedVariableName : LocalQuickFix {
    override fun getFamilyName(): String = "Remove unused variable"
    
    override fun applyFix(project: Project, descriptor: ProblemDescriptor) {
        // Apply fix
    }
}
```

#### 2. Register in plugin.xml

```xml
<extensions defaultExtensionNs="com.intellij">
    <localInspection
        language="Go Template"
        shortName="UnusedTemplateVariable"
        displayName="Unused template variable"
        groupName="Go text/template"
        enabledByDefault="true"
        implementationClass="com.example.plugin.inspections.UnusedVariableInspection"
    />
</extensions>
```

#### 3. Test the Inspection

In `src/test/kotlin/.../UnusedVariableInspectionTest.kt`:

```kotlin
class UnusedVariableInspectionTest : BasePlatformTestCase() {
    fun testDetectsUnusedVariable() {
        myFixture.enableInspections(UnusedVariableInspection::class.java)
        
        myFixture.configureByText("test.tmpl", """
            {{ ${'$'}unused := "value" }}
            {{ . }}
        """.trimIndent())
        
        val problems = myFixture.doHighlighting()
        assertEquals(1, problems.size)
        assertEquals("Unused template variable", problems[0].description)
    }
}
```

### Example: Adding Syntax Highlighting

Create a syntax highlighter:

```kotlin
class GoTemplateSyntaxHighlighter : SyntaxHighlighter {
    override fun getHighlightingLexer() = GoTemplateLexer()
    
    override fun getTokenHighlights(tokenType: IElementType?): Array<TextAttributesKey> {
        return when (tokenType) {
            GoTemplateTokenTypes.LBRACE -> arrayOf(BRACES)
            GoTemplateTokenTypes.KEYWORD -> arrayOf(KEYWORD)
            GoTemplateTokenTypes.STRING -> arrayOf(STRING)
            else -> TextAttributesKey.EMPTY_ARRAY
        }
    }
    
    companion object {
        private val BRACES = TextAttributesKey.createTextAttributesKey(
            "GOTMPL_BRACES",
            TextAttributes(Color(0xFF6A00), null, null, null, Font.PLAIN)
        )
        
        private val KEYWORD = TextAttributesKey.createTextAttributesKey(
            "GOTMPL_KEYWORD",
            TextAttributes(Color(0x0033B3), null, null, null, Font.BOLD)
        )
        
        private val STRING = TextAttributesKey.createTextAttributesKey(
            "GOTMPL_STRING",
            TextAttributes(Color(0x067D17), null, null, null, Font.PLAIN)
        )
    }
}
```

Register in `plugin.xml`:

```xml
<extensions defaultExtensionNs="com.intellij.lang">
    <syntaxHighlighterFactory
        language="Go Template"
        implementationClass="com.example.plugin.highlighting.GoTemplateSyntaxHighlighterFactory"
    />
</extensions>
```

## Configuration

Read [jetbrains-config.md](jetbrains-configuration.md) for documentation about config in the JetBrains plugin.

## Testing

Read [jetbrains-testing.md](jetbrains-testing.md) for a guide on testing the JetBrains plugin.

## Building and Packaging

Read the main [README](../../README.md) section about how to run and build the extension.

## Resources

- [JetBrains Plugin Development Guide](https://plugins.jetbrains.com/docs/intellij/welcome.html)
- [Language Server Protocol Integration](https://plugins.jetbrains.com/docs/intellij/language-server-protocol.html)
- [UI Design Guide](https://plugins.jetbrains.com/docs/intellij/ui-components.html)
- [Kotlin UI DSL](https://plugins.jetbrains.com/docs/intellij/kotlin-ui-dsl-version-2.html)
- [Gradle IntelliJ Plugin](https://plugins.jetbrains.com/docs/intellij/gradle-build-system.html)
- [Sample Plugins](https://github.com/JetBrains/intellij-sdk-code-samples)
