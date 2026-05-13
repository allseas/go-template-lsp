package com.allseas.gotexttemplate

import com.intellij.codeInsight.template.TemplateContextType
import com.intellij.psi.PsiFile

class GoTextTemplateContext : TemplateContextType("Go text/template") {
    override fun isInContext(
        file: PsiFile,
        offset: Int,
    ): Boolean = file.name.endsWith(".tmpl")
}
