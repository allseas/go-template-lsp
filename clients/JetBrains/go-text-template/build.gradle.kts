
plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.1.20"
    id("org.jetbrains.intellij.platform") version "2.10.2"
    id("org.jlleitschuh.gradle.ktlint") version "14.2.0"
}

group = "com.example"
version = "1.0-SNAPSHOT"

repositories {
    mavenCentral()
    intellijPlatform {
        defaultRepositories()
    }
}
ktlint {
    verbose.set(true)
    outputToConsole.set(true)
}
// Read more: https://plugins.jetbrains.com/docs/intellij/tools-intellij-platform-gradle-plugin.html
dependencies {
    intellijPlatform {
        intellijIdea("2025.2.4")
        testFramework(org.jetbrains.intellij.platform.gradle.TestFrameworkType.Platform)

        // Add plugin dependencies for compilation here:
        bundledPlugin("com.intellij.properties")
        bundledPlugin("com.intellij.modules.json")
        bundledPlugin("org.jetbrains.plugins.textmate")
        plugin("com.redhat.devtools.lsp4ij", "0.19.3")
    }
}

intellijPlatform {
    pluginConfiguration {
        ideaVersion {
            sinceBuild = "252.25557"
        }

        changeNotes =
            """
            Initial version
            """.trimIndent()
    }
}

tasks {
    // Set the JVM compatibility versions
    withType<JavaCompile> {
        sourceCompatibility = "21"
        targetCompatibility = "21"
    }

    prepareSandbox {
        from("src/main/resources/textmate/go-text-template") {
            into(pluginName.map { "$it/textmate/go-text-template" })
        }
    }
}

tasks.register<Exec>("compileServer") {
    workingDir = rootDir.resolve("..").resolve("..")
    val npmCommand = if (System.getProperty("os.name").lowercase().contains("windows")) "npm.cmd" else "npm"
    commandLine(npmCommand, "run", "build:server")
}

tasks.register<Copy>("copyServerBin") {
    dependsOn("compileServer")
    from(
        rootDir
            .resolve("..")
            .resolve("..")
            .resolve("..")
            .resolve("dist")
            .resolve("server"),
    )
    include("**")

    into(
        rootDir
            .resolve("src")
            .resolve("main")
            .resolve("resources")
            .resolve("bin")
            .resolve("language-server"),
    )
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_21)
    }
}
tasks.build {
    dependsOn("addKtlintCheckGitPreCommitHook")
}

tasks.processResources {
    dependsOn("copyServerBin")
}
