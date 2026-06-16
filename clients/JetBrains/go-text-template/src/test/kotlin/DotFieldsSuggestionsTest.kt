import com.intellij.codeInsight.completion.CompletionType

// This class has to test around issues 101 and 102. There might have to be some changes to some assertions once fixed
// However, these tests are still of value due to the fact that they verify the server can read the Go files and suggest accordingly
class DotFieldsSuggestionsTest : CustomFixtureHeavyTestCase() {
    override fun setUp() {
        super.setUp()
        myFixture.copyDirectoryToProject("", "")
    }

    fun testAllCompletionsRecommended() {
        myFixture.configureByText(
            "test2.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{<caret>}}
            """.trimIndent(),
        )

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
    }

    fun testNestedFieldCompletionsOnStructField() {
        myFixture.configureByText(
            "test_nested.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{.Address.<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            "Street",
            "City",
            "Country",
            "Zip",
        )
        assertDoesntContain(suggestions, "CustomerName", "Items", "Paid")
    }

    fun testNestedMethodCompletionsOnStructField() {
        myFixture.configureByText(
            "test_nested_methods.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{.Address.<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            "Line",
            "IsLocal",
            "ZipCode",
        )
        assertDoesntContain(suggestions, "DisplayName", "ItemCount", "IsLargeOrder")
    }

    fun testMethodsRecommended() {
        myFixture.configureByText(
            "test_methods.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            ".DisplayName",
            ".Summary",
            ".ItemCount",
            ".IsLargeOrder",
            ".Format",
        )
    }

    fun testInvalidMethodsExcluded() {
        myFixture.configureByText(
            "test_invalid_methods.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertDoesntContain(suggestions!!, ".badReturn", ".wrongSecond")
    }

//    Currently Broken, see issue 101
//    fun testNoCompletionsOnPrimitiveField() {
//        myFixture.configureByText(
//            "test_primitive.txt.tmpl",
//            """
//            {{/*gotype: cg/model.Order*/}}
//            {{.CustomerName.<caret>}}
//            """.trimIndent(),
//        )
//
//        myFixture.complete(CompletionType.BASIC)
//        val suggestions = myFixture.lookupElementStrings
//        if (suggestions != null) {
//            assertDoesntContain(suggestions, "ID", "CustomerName", "DisplayName", "Street")
//        }
//    }

    fun testDotFieldInsideIfBlock() {
        myFixture.configureByText(
            "test_if.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{if .Paid}}
                {{<caret>}}
            {{end}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            ".Address",
            ".CustomerName",
            ".Items",
            ".TotalAmount",
        )
    }

    fun testDotFieldInsideRangeBlock() {
        myFixture.configureByText(
            "test_range.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{range .Items}}
                {{<caret>}}
            {{end}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            ".SKU",
            ".Name",
            ".Qty",
            ".UnitPrice",
        )
    }

    fun testDotFieldInsideRangeBlockMethods() {
        myFixture.configureByText(
            "test_range_methods.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{range .Items}}
                {{<caret>}}
            {{end}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            ".Label",
            ".Total",
            ".IsExpensive",
            ".Describe",
        )
        assertDoesntContain(suggestions, ".DisplayName", ".ItemCount", ".IsLargeOrder")
    }

    fun testDotFieldInsideWithBlock() {
        myFixture.configureByText(
            "test_with.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{with .Address}}
                {{<caret>}}
            {{end}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            ".Street",
            ".City",
            ".Country",
            ".Zip",
        )
        assertDoesntContain(suggestions, ".CustomerName", ".Items", ".TotalAmount")
    }

    fun testPartialFieldNameFiltering() {
        myFixture.configureByText(
            "test_partial.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{.Cus<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        // Only one suggestion, so returns null and autocompletes
        assertNull(suggestions)
        myFixture.checkResult(
            """
            {{/*gotype: cg/model.Order*/}}
            {{.CustomerName}}
            """.trimIndent(),
        )
    }

    fun testNoBuiltinsInDotCompletions() {
        myFixture.configureByText(
            "test_no_builtins.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{.<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertDoesntContain(suggestions!!, "len", "eq", "html", "print", "not")
    }

    fun testNoCompletionsWithoutGotype() {
        myFixture.configureByText(
            "test_no_gotype.txt.tmpl",
            """
            {{.<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        if (suggestions != null) {
            assertDoesntContain(
                suggestions,
                "Address",
                "CustomerName",
                "Items",
                "DisplayName",
            )
        }
    }

    fun testDotFieldInPipeExpression() {
        myFixture.configureByText(
            "test_pipe.txt.tmpl",
            """
            {{/*gotype: cg/model.Order*/}}
            {{<caret> | len}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            ".Address",
            ".CustomerName",
            ".Items",
        )
    }

    fun testMultipleTemplatesWithDifferentTypes() {
        myFixture.configureByText(
            "test_multi_define.txt.tmpl",
            """
            {{/*gotype: cg/model.Address*/}}
            {{<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            ".Street",
            ".City",
            ".Country",
            ".Zip",
        )
        assertDoesntContain(suggestions, ".CustomerName", ".Items", ".TotalAmount")
    }

    fun testMultipleChainedDots() {
        myFixture.configureByText(
            "test_multiple_dots.txt.tmpl",
            """
            {{/*gotype: cg/model.Tree*/}}
            {{.Left.Left.Left.Left.Left.<caret>}}
            """.trimIndent(),
        )

        myFixture.complete(CompletionType.BASIC)
        val suggestions = myFixture.lookupElementStrings
        assertNotNull(suggestions)
        assertContainsElements(
            suggestions!!,
            "Left",
            "Right",
        )
    }
}
