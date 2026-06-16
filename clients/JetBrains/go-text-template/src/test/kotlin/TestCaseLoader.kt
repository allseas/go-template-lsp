import com.google.gson.Gson
import com.google.gson.reflect.TypeToken

data class CompletionTestCase(
    val name: String,
    val content: String,
    val expectedIncludes: List<String>,
    val expectedExcludes: List<String>,
    val expectedIncludesExactlyOnce: List<String>? = null,
    val expectedResult: String? = null,
    val vscodeOnly: Boolean? = null,
    val poll: Boolean? = null,
)

data class DefinitionTestCase(
    val name: String,
    val content: String,
    val expected: DefinitionExpected,
    val vscodeOnly: Boolean? = null,
    val poll: Boolean? = null,
)

data class DefinitionExpected(
    val targetLine: Int? = null,
    val targetFile: String? = null,
    val count: Int? = null,
    val minCount: Int? = null,
    val noResult: Boolean? = null,
)

private val gson = Gson()

private fun loadResource(resourcePath: String): String =
    TestCaseLoader::class.java
        .getResourceAsStream(resourcePath)
        ?.bufferedReader()
        ?.readText()
        ?: error("Resource not found: $resourcePath")

object TestCaseLoader

fun loadCompletionTestCases(): List<CompletionTestCase> {
    val json = loadResource("/testcases/completion.json")
    val type = object : TypeToken<List<CompletionTestCase>>() {}.type
    return gson.fromJson(json, type)
}

fun loadDefinitionTestCases(): List<DefinitionTestCase> {
    val json = loadResource("/testcases/definition.json")
    val type = object : TypeToken<List<DefinitionTestCase>>() {}.type
    return gson.fromJson(json, type)
}

/**
 * Converts the <cursor> marker in content to the IntelliJ <caret> marker.
 * Returns the converted content string ready for use with myFixture.configureByText.
 */
fun toCaret(content: String): String = content.replace("<cursor>", "<caret>")

/**
 * Whether a test case relies on a resolved Go type (via a `gotype` annotation),
 * which requires the heavy fixture that copies the Go model project.
 */
fun requiresGoProject(content: String): Boolean = content.contains("gotype:")
