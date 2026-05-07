import com.intellij.openapi.vfs.newvfs.impl.VfsRootAccess
import com.intellij.testFramework.fixtures.BasePlatformTestCase
import java.io.File

abstract class CustomPlatformTestCase : BasePlatformTestCase() {
    override fun getTestDataPath(): String = "src/test/resources"

    override fun setUp() {
        super.setUp()

        val absoluteTestDataPath = File(testDataPath).absolutePath

        VfsRootAccess.allowRootAccess(testRootDisposable, absoluteTestDataPath)
    }
}
