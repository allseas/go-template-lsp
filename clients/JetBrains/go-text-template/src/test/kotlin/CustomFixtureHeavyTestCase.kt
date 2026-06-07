import com.intellij.testFramework.UsefulTestCase
import com.intellij.testFramework.fixtures.CodeInsightTestFixture
import com.intellij.testFramework.fixtures.IdeaTestFixtureFactory

abstract class CustomFixtureHeavyTestCase : UsefulTestCase() {
    protected lateinit var myFixture: CodeInsightTestFixture

    override fun setUp() {
        super.setUp()
        val factory = IdeaTestFixtureFactory.getFixtureFactory()
        // Use a single param constructor to build a heavy fixture
        val builder = factory.createFixtureBuilder(name)

        myFixture = factory.createCodeInsightFixture(builder.fixture)
        myFixture.testDataPath = "../../../test/resources/templ-tests"
        myFixture.setUp()

        val tempDir = myFixture.tempDirFixture.getFile("")?.path
        if (tempDir != null) {
            System.setProperty("lsp.working.directory", tempDir)
        }

        // Ensure VFS is accessible if needed
        com.intellij.openapi.vfs.newvfs.impl.VfsRootAccess.allowRootAccess(
            myFixture.testRootDisposable,
            java.io.File(myFixture.testDataPath).absolutePath,
        )
    }

    override fun tearDown() {
        try {
            System.clearProperty("lsp.working.directory")
            myFixture.tearDown()
        } catch (e: Throwable) {
            addSuppressedException(e)
        } finally {
            super.tearDown()
        }
    }
}
