package com.example.gotexttemplate

import com.intellij.codeInsight.template.TemplateContextType
import com.intellij.psi.PsiFile

class GoTextTemplateContext : TemplateContextType("Go text/template") {
    override fun isInContext(file: PsiFile, offset: Int): Boolean {
        return file.name.endsWith(".tmpl")
    }
}
