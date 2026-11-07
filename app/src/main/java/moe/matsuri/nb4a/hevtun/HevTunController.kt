package moe.matsuri.nb4a.hevtun

import java.io.File

/** Abstraction over the hev-socks5-tunnel native library (faked in unit tests). */
interface HevTunBackend {
    fun start(configPath: String, tunFd: Int): Boolean
    fun stop()
}

object NativeHevTunBackend : HevTunBackend {
    override fun start(configPath: String, tunFd: Int) =
        HevTunNative.TProxyStartService(configPath, tunFd)

    override fun stop() {
        HevTunNative.TProxyStopService()
    }
}

/** Lifecycle handling around a [HevTunBackend]: at most one tunnel runs at a time. */
class HevTunController(private val backend: HevTunBackend) {

    var running = false
        private set

    /** Writes [config] to [configFile] and starts the tunnel on [tunFd], replacing any running one. */
    @Synchronized
    fun start(configFile: File, config: HevTunConfig, tunFd: Int) {
        stop()
        configFile.parentFile?.mkdirs()
        configFile.writeText(config.toYaml())
        check(backend.start(configFile.absolutePath, tunFd)) {
            "Failed to start hev-socks5-tunnel"
        }
        running = true
    }

    @Synchronized
    fun stop() {
        if (!running) return
        try {
            backend.stop()
        } finally {
            running = false
        }
    }
}
