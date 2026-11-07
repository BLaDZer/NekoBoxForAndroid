package moe.matsuri.nb4a.hevtun

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder

class HevTunControllerTest {

    @get:Rule
    val tmp = TemporaryFolder()

    private class FakeBackend(var startResult: Boolean = true, var stopFails: Boolean = false) : HevTunBackend {
        val calls = mutableListOf<String>()
        override fun start(configPath: String, tunFd: Int): Boolean {
            calls += "start:$configPath:$tunFd"
            return startResult
        }

        override fun stop() {
            calls += "stop"
            if (stopFails) throw IllegalStateException("boom")
        }
    }

    private val config = HevTunConfig(mtu = 1500, ipv4 = "10.0.0.1", socksAddress = "127.0.0.1", socksPort = 1080)

    @Test
    fun startWritesConfigAndStartsBackend() {
        val backend = FakeBackend()
        val controller = HevTunController(backend)
        val file = tmp.root.resolve("nested/hev.yaml")

        controller.start(file, config, 42)

        assertEquals(config.toYaml(), file.readText())
        assertEquals(listOf("start:${file.absolutePath}:42"), backend.calls)
        assertTrue(controller.running)
    }

    @Test
    fun startFailureThrowsAndStaysStopped() {
        val backend = FakeBackend(startResult = false)
        val controller = HevTunController(backend)

        assertThrows(IllegalStateException::class.java) {
            controller.start(tmp.root.resolve("hev.yaml"), config, 3)
        }
        assertFalse(controller.running)
        controller.stop()
        assertEquals(1, backend.calls.size) // no stop for a tunnel that never started
    }

    @Test
    fun restartStopsPreviousTunnelFirst() {
        val backend = FakeBackend()
        val controller = HevTunController(backend)
        val file = tmp.root.resolve("hev.yaml")

        controller.start(file, config, 1)
        controller.start(file, config, 2)

        assertEquals(
            listOf("start:${file.absolutePath}:1", "stop", "start:${file.absolutePath}:2"),
            backend.calls
        )
    }

    @Test
    fun stopIsIdempotent() {
        val backend = FakeBackend()
        val controller = HevTunController(backend)
        controller.stop()
        assertTrue(backend.calls.isEmpty())

        controller.start(tmp.root.resolve("hev.yaml"), config, 1)
        controller.stop()
        controller.stop()
        assertEquals(2, backend.calls.size)
        assertFalse(controller.running)
    }

    @Test
    fun stopFailureStillMarksStopped() {
        val backend = FakeBackend(stopFails = true)
        val controller = HevTunController(backend)
        controller.start(tmp.root.resolve("hev.yaml"), config, 1)

        assertThrows(IllegalStateException::class.java) { controller.stop() }
        assertFalse(controller.running)
    }
}
