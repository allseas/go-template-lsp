class SnippetsTest : CustomPlatformTestCase() {
    fun testLiveTemplateExpandsOnTab() {
        myFixture.configureByText("test.txt.tmpl", "")

        myFixture.type("if")
        myFixture.type('\t')
        myFixture.type("someStatement")
        myFixture.type('\t')
        myFixture.type('\n')

        myFixture.checkResult(
            "{{ if someStatement }}\n" +
                "{{ end }}",
        )
    }

    fun testLiveTemplateDoesntExpandOutsideTmpl() {
        myFixture.configureByText("test.html", "")

        myFixture.type("if")
        myFixture.type('\t')
        myFixture.type("someStatement")
        myFixture.type('\t')
        myFixture.type('\n')

        myFixture.checkResult("if  someStatement   \n")
    }
}
