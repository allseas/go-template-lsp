package com.allseas.gotexttemplate

import com.intellij.openapi.fileTypes.LanguageFileType
import javax.swing.Icon

object TmplFileType : LanguageFileType(TmplLanguage) {
    override fun getName(): String = "Go Template"

    override fun getDescription(): String = "Go text/template file"

    override fun getDefaultExtension(): String = "tmpl"

    override fun getIcon(): Icon = TextTemplateIcons.FileIcon
}
