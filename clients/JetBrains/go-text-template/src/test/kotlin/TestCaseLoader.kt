import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import java.io.File

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

fun loadCompletionTestCases(testCasesDir: String): List<CompletionTestCase> {
    val json = File("$testCasesDir/completion.json").readText()
    val type = object : TypeToken<List<CompletionTestCase>>() {}.type
    return gson.fromJson(json, type)
}

fun loadDefinitionTestCases(testCasesDir: String): List<DefinitionTestCase> {
    val json = File("$testCasesDir/definition.json").readText()
    val type = object : TypeToken<List<DefinitionTestCase>>() {}.type
    return gson.fromJson(json, type)
}

/**
 * Converts the <cursor> marker in content to the IntelliJ <caret> marker.
 * Returns the converted content string ready for use with myFixture.configureByText.
 */
fun toCaret(content: String): String = content.replace("<cursor>", "<caret>")
