package com.allseas.gotexttemplate

import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.openapi.extensions.PluginId
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.SystemInfo
import com.redhat.devtools.lsp4ij.server.ProcessStreamConnectionProvider
import java.io.File
import java.io.FileNotFoundException
import java.nio.file.Files
import java.nio.file.Path

class TextTemplateLspServerConnectionProvider(
    project: Project,
) : ProcessStreamConnectionProvider() {
    init {
        val binary = getBinary(project)
        commands = listOf(binary.absolutePath, "--stdio")
        val testWorkingDir = System.getProperty("lsp.working.directory")
        workingDirectory = testWorkingDir ?: project.basePath
    }

    private fun getBinary(project: Project): File {
        val configuredPath =
            System.getProperty("lsp.server.path")
                ?: ProjectSettings.getInstance(project).getEffectiveState().serverPath
        val normalizedPath =
            configuredPath
                .trim()
                .removeSurrounding("\"")
                .trim()
                .filter { it.code >= 0x20 && it.category != CharCategory.FORMAT }
        if (normalizedPath.isNotBlank()) {
            val configuredFile = File(normalizedPath)
            if (!configuredFile.exists()) {
                throw FileNotFoundException("Could not find server binary at configured path: $normalizedPath")
            }
            configuredFile.setExecutable(true)
            return configuredFile
        }

        val pluginId = PluginId.getId("com.allseas.go-text-template") // Should match exactly with plugin.xml
        val pluginPath: Path? = PluginManagerCore.getPlugin(pluginId)?.pluginPath

        val platform = detectPlatform()
        val binaryName = "gotmpl-server$platform"
//        val resource = javaClass.classLoader.getResourceAsStream("server/$binaryName")

//        val cacheDir = File(PathManager.getSystemPath(), "go-text-template-lsp").also { it.mkdirs() }

        val binaryPath = pluginPath?.resolve("server/$binaryName")

        if (binaryPath == null || !Files.exists(binaryPath)) {
            throw FileNotFoundException("Could not find $binaryPath")
        }

//
//        val binaryFile = File(cacheDir, binaryName)
//        if (!binaryFile.exists()) {
//            resource.use { Files.copy(it, binaryFile.toPath()) }
//        } else {
//            // Check if the existinclassLoader.getResourceAsStream("server/$binaryNamg binary is the same as the resource, if not, replace it
//            val existingBytes = Files.readAllBytes(binaryFile.toPath())
//            val resourceBytes = resource.use { it.readAllBytes() }
//            if (!existingBytes.contentEquals(resourceBytes)) {
//                Files.copy(resourceBytes.inputStream(), binaryFile.toPath(), java.nio.file.StandardCopyOption.REPLACE_EXISTING)
//            }
//        }
        val binaryFile = binaryPath.toFile()
        binaryFile.setExecutable(true)
        return binaryFile
    }

    private fun detectPlatform(): String? =
        when {
            SystemInfo.isMac && SystemInfo.OS_ARCH == "aarch64" -> "-darwin-arm64"
            SystemInfo.isMac -> "-darwin-amd64"
            SystemInfo.isWindows && SystemInfo.OS_ARCH == "aarch64" -> "-arm64.exe"
            SystemInfo.isWindows -> ".exe"
            SystemInfo.isLinux && SystemInfo.OS_ARCH == "aarch64" -> "-arm64"
            SystemInfo.isLinux -> ""
            else -> null
        }
}
