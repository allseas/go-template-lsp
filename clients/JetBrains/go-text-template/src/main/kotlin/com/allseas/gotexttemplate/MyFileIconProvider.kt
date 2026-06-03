package com.allseas.gotexttemplate

import com.intellij.ide.FileIconProvider
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.IconLoader
import com.intellij.openapi.vfs.VirtualFile
import javax.swing.Icon

object MyIcons {
    @JvmField
    val FILE = IconLoader.getIcon("/icons/icon.svg", MyIcons::class.java)
}

class MyFileIconProvider : FileIconProvider {
    override fun getIcon(
        file: VirtualFile,
        flags: Int,
        project: Project?,
    ): Icon? = if (file.extension == "tmpl") MyIcons.FILE else null
}
