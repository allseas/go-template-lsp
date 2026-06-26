import org.gradle.api.tasks.testing.logging.TestLogEvent
import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.3.20"
    id("org.jetbrains.intellij.platform") version "2.12.0"
    id("org.jlleitschuh.gradle.ktlint") version "14.2.0"
}

group = "com.allseas"
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
sourceSets {
    create("integrationTest") {
        compileClasspath += sourceSets.main.get().output
        runtimeClasspath += sourceSets.main.get().output
    }
}

val integrationTestImplementation by configurations.getting {
    extendsFrom(configurations.testImplementation.get())
}
// Read more: https://plugins.jetbrains.com/docs/intellij/tools-intellij-platform-gradle-plugin.html
dependencies {
    intellijPlatform {
        intellijIdea("2026.1.1")
        testImplementation("org.junit.jupiter:junit-jupiter-api:6.0.3")
        testImplementation("org.junit.platform:junit-platform-launcher:6.0.3")
        testImplementation("junit:junit:4.13.2")
        testFramework(TestFrameworkType.Platform)
        // Add plugin dependencies for compilation here:
        bundledPlugin("com.intellij.properties")
        bundledPlugin("com.intellij.modules.json")
        bundledPlugin("org.jetbrains.plugins.textmate")
        plugin("com.redhat.devtools.lsp4ij", "0.19.3")
    }
    integrationTestImplementation("org.junit.jupiter:junit-jupiter:5.7.1")
    integrationTestImplementation("org.kodein.di:kodein-di-jvm:7.20.2")
    integrationTestImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-core-jvm:1.10.1")
}
val integrationTest by intellijPlatformTesting.testIdeUi.registering {
    task {
        val integrationTestSourceSet = sourceSets.getByName("integrationTest")
        testClassesDirs = integrationTestSourceSet.output.classesDirs
        classpath = integrationTestSourceSet.runtimeClasspath
        useJUnitPlatform()

        systemProperty("ide.browser.jcef.enabled", "false")

        testLogging {
            showStandardStreams = true
            exceptionFormat = org.gradle.api.tasks.testing.logging.TestExceptionFormat.FULL
        }

        if (org.gradle.internal.os.OperatingSystem
                .current()
                .isLinux
        ) {
            systemProperty("awt.toolkit.name", "XToolKit")
            environment("WAYLAND_DISPLAY", "")
            environment("XDG_SESSION_TYPE", "x11")
        }
    }
}

tasks.test {
    testLogging {
        showStandardStreams = true
        showCauses = true
        showExceptions = true
        exceptionFormat = org.gradle.api.tasks.testing.logging.TestExceptionFormat.FULL
        events(
            TestLogEvent.FAILED,
            TestLogEvent.PASSED,
            TestLogEvent.SKIPPED,
            TestLogEvent.STANDARD_OUT,
        )
    }
}

tasks.processTestResources {
    from(sourceSets.main.get().resources)
    from(rootProject.file("../../../test/testcases")) {
        into("testcases")
    }
    dependsOn("copyServerBin")
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
        from("src/main/server") {
            into(pluginName.map { "$it/server" })
        }
    }
    prepareTestSandbox {
        from("src/main/resources/textmate/go-text-template") {
            into(pluginName.map { "$it/textmate/go-text-template" })
        }
        from("src/main/server") {
            into(pluginName.map { "$it/server" })
        }
    }
}

tasks.register<Exec>("compileServer") {
    workingDir = rootDir.resolve("..").resolve("..")
    val npmCommand = if (System.getProperty("os.name").lowercase().contains("windows")) "npm.cmd" else "npm"
    val buildTarget = if (project.hasProperty("allseas")) "build:server:allseas" else "build:server"
    commandLine(npmCommand, "run", buildTarget)
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
            .resolve("server"),
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

tasks.named("buildSearchableOptions") {
    enabled = false
}

tasks.processResources {
    dependsOn("copyServerBin")
}
