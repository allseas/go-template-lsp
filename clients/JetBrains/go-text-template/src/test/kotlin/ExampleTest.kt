// For more info on myFixture. Read https://github.com/JetBrains/intellij-community/blob/idea/261.23567.138/platform/testFramework/src/com/intellij/testFramework/fixtures/CodeInsightTestFixture.java
class ExampleTest : CustomPlatformTestCase() {
    fun testTest() {
        myFixture.configureByFile("exampleBefore.txt")
        myFixture.checkResultByFile("exampleAfter.txt")
    }
}
