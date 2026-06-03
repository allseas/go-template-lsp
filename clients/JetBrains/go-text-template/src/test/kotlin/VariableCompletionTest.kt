import com.intellij.codeInsight.completion.CompletionType
import com.intellij.codeInsight.lookup.Lookup

class VariableCompletionTest : CustomPlatformTestCase() {

    fun testDollarSignAlwaysSuggested() {
        myFixture.configureByText("test1.txt.tmpl", "{{<caret>}}")
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, "$")
    }

    fun testDotAlwaysSuggested() {
        myFixture.configureByText("test2.txt.tmpl", "{{<caret>}}")
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, ".")
    }

    fun testMultipleVariablesSuggested() {
        myFixture.configureByText(
            "test3.txt.tmpl", $$"""{{$a := a}}
            {{$b := b}}
            {{$c := c}}
            {{$d := d}}
            {{$e := e}}
            {{<caret>}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, $$"$a", $$"$b", $$"$c", $$"$d", $$"$e")
    }

    fun testVariablesAreSuggestedInScope() {
        myFixture.configureByText(
            "test4.txt.tmpl", $$"""
            {{range $a, $b := .}}
            {{<caret>}}
            {{end}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, $$"$a", $$"$b")
    }

    fun testVariablesAreNotSuggestedOutsideScope() {
        myFixture.configureByText(
            "test5.txt.tmpl", $$"""
            
            {{range $a, $b := .}}
            {{end}}
            {{<caret>}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertDoesntContain(suggestedCompletions!!, $$"$a", $$"$b")
    }

    fun testVariableIsSuggestedWithDollarSignExclusivelyAndFilledCorrectlyOnPartialMatch() {
        myFixture.configureByText(
            "test6.txt.tmpl", $$"""
            {{$variable := 123}}
            {{$b := b}}
            {{$va<caret>}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertOneElement(suggestedCompletions!!)
        assertContainsElements(suggestedCompletions, "variable")

        myFixture.finishLookup(Lookup.NORMAL_SELECT_CHAR)

        myFixture.checkResult($$"""
            {{$variable := 123}}
            {{$b := b}}
            {{$variable}}
        """.trimIndent())
    }

    fun testVariableIsSuggestedExclusivelyWithoutDollarSignAndFilledCorrectlyOnPartialMatch() {
        myFixture.configureByText(
            "test7.txt.tmpl", $$"""
            {{$variable := 123}}
            {{$b := b}}
            {{va<caret>}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertOneElement(suggestedCompletions!!)
        assertContainsElements(suggestedCompletions, $$"$variable")

        myFixture.finishLookup(Lookup.NORMAL_SELECT_CHAR)

        myFixture.checkResult($$"""
            {{$variable := 123}}
            {{$b := b}}
            {{$variable}}
        """.trimIndent())
    }

    fun testVariablesAreSuggestedInWithAndIfScopes() {
        myFixture.configureByText(
            "test_with_if.txt.tmpl", $$"""
            {{with $x := .Field}}
                {{<caret>}}
            {{end}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        var suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, $$"$x")

        myFixture.configureByText(
            "test_with_if2.txt.tmpl", $$"""
            {{if $y := .Field}}
                {{<caret>}}
            {{end}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, $$"$y")
    }

    fun testVariablesAreNotSuggestedOutsideWithAndIfScopes() {
        myFixture.configureByText(
            "test_with_if_outside.txt.tmpl", $$"""
            {{with $x := .Field}}{{end}}
            {{if $y := .Field}}{{end}}
            {{<caret>}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertDoesntContain(suggestedCompletions!!, $$"$x", $$"$y")
    }

    fun testMultiAssignmentVariables() {
        myFixture.configureByText(
            "test_multi_assignment.txt.tmpl", $$"""
            {{$a, $b := myFunc}}
            {{<caret>}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, $$"$a", $$"$b")
    }

    fun testVariablesAreNotSuggestedInsideComments() {
        myFixture.configureByText(
            "test_comments.txt.tmpl", $$"""
            {{$va := 1}}
            {{/* {{$v<caret>}} */}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        if (suggestedCompletions != null) {
            assertDoesntContain(suggestedCompletions, $$"$va")
        }
    }

    fun testNestedScopeDeduplication() {
        myFixture.configureByText(
            "test_nested_scope.txt.tmpl", $$"""
            {{$a := 1}}
            {{range .}}
                {{$a := 2}}
                {{<caret>}}
            {{end}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        val count = suggestedCompletions!!.count { it == $$"$a" }
        assertEquals(1, count)
    }

    fun testPipelineAndFunctionArgumentCompletions() {
        myFixture.configureByText(
            "test_pipeline_args.txt.tmpl", $$"""
            {{$myVar := 123}}
            {{ "foo" | myFunc $<caret> }}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        var suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, "myVar")

        myFixture.configureByText(
            "test_func_args.txt.tmpl", $$"""
            {{$myVar := 123}}
            {{ myFunc 123 "test" $<caret> }}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertContainsElements(suggestedCompletions!!, "myVar")
    }

    fun testGlobalVsLocalIsolationInDefineBlocks() {
        myFixture.configureByText(
            "test_define_blocks.txt.tmpl", $$"""
            {{define "foo"}}
               {{$fooVar := 1}}
            {{end}}

            {{define "bar"}}
               {{<caret>}}
            {{end}}
        """.trimIndent()
        )
        myFixture.complete(CompletionType.BASIC)
        val suggestedCompletions = myFixture.lookupElementStrings
        assertNotNull(suggestedCompletions)
        assertDoesntContain(suggestedCompletions!!, $$"$fooVar")
    }
}