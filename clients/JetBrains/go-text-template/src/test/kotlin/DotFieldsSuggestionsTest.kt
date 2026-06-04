import com.intellij.codeInsight.completion.CompletionType

class DotFieldsSuggestionsTest : CustomFixtureHeavyTestCase() {
    override fun setUp() {
        super.setUp()
        myFixture.copyDirectoryToProject("DotFieldTestResources/", "")
    }
//
//    private fun printVfsTree(
//        dir: VirtualFile,
//        indent: String,
//    ) {
//        for (child in dir.children) {
//            println("$indent${child.name}${if (child.isDirectory) "/" else ""}")
//            if (child.isDirectory) {
//                printVfsTree(child, "$indent  ")
//            }
//        }
//    }

    fun testAllCompletionsRecommended() {
        myFixture.configureByText(
            "test2.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{<caret>}}
            """.trimIndent(),
        )
//        val root = myFixture.tempDirFixture.getFile("")!!
//        root.refresh(false, true)
//        println("=== VFS Tree ===")
//        println("Root path: ${root.path}")
//        println("Project base path: ${myFixture.project.basePath}")
//        println("File URI: ${myFixture.file.virtualFile.url}")
//        printVfsTree(root, "")
//        println("=== End VFS Tree ===")

        // Wait for language server indexing to catch up
//        Thread.sleep(2500)

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        System.err.println(suggestions)
        assertContainsElements(
            suggestions!!,
            ".Address",
            ".CustomerName",
            ".DisplayName",
            ".Items",
            ".Paid",
            ".TotalAmount",
        )
//        assertSize(6, suggestions)
    }
}
