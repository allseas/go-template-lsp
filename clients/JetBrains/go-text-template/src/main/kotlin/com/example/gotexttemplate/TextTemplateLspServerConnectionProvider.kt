package com.example.gotexttemplate

import com.intellij.openapi.application.PathManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.SystemInfo
import com.redhat.devtools.lsp4ij.server.ProcessStreamConnectionProvider
import java.io.File
import java.nio.file.Files

class TextTemplateLspServerConnectionProvider(
    project: Project,
) : ProcessStreamConnectionProvider() {
    init {
        val binary = getBinary()
        commands = listOf(binary.absolutePath, "--stdio")
        workingDirectory = project.basePath
    }

    private fun getBinary(): File {
        val platform = detectPlatform()
        val binaryName = "gotmpl-server$platform"
        val resource = javaClass.classLoader.getResourceAsStream("bin/language-server/$binaryName")
        val cacheDir = File(PathManager.getSystemPath(), "go-text-template-lsp").also { it.mkdirs() }

        val binaryFile = File(cacheDir, binaryName)
        if (!binaryFile.exists()) {
            resource.use { Files.copy(it, binaryFile.toPath()) }
        } else {
            // Check if the existing binary is the same as the resource, if not, replace it
            val existingBytes = Files.readAllBytes(binaryFile.toPath())
            val resourceBytes = resource.use { it.readAllBytes() }
            if (!existingBytes.contentEquals(resourceBytes)) {
                Files.copy(resourceBytes.inputStream(), binaryFile.toPath(), java.nio.file.StandardCopyOption.REPLACE_EXISTING)
            }
        }
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
