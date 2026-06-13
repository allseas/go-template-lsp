import com.google.gson.Gson
import com.google.gson.reflect.TypeToken

data class CompletionTestCase(
    val name: String,
    val content: String,
    val expectedIncludes: List<String>,
    val expectedExcludes: List<String>,
    val expectedIncludesExactlyOnce: List<String>? = null,
)

data class DefinitionTestCase(
    val name: String,
    val content: String,
    val expected: DefinitionExpected,
)

data class DefinitionExpected(
    val targetLine: Int? = null,
    val count: Int? = null,
    val minCount: Int? = null,
    val noResult: Boolean? = null,
)

private val gson = Gson()

private fun loadResource(resourcePath: String): String =
    TestCaseLoader::class.java.getResourceAsStream(resourcePath)
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
