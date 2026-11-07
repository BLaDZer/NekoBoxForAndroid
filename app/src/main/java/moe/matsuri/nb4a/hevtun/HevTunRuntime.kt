package moe.matsuri.nb4a.hevtun

import android.content.Context
import io.nekohasekai.sagernet.IPv6Mode
import io.nekohasekai.sagernet.bg.VpnService
import io.nekohasekai.sagernet.database.DataStore
import io.nekohasekai.sagernet.fmt.LOCALHOST
import java.io.File

/**
 * In-process hev-socks5-tunnel runtime. The tunnel reads packets from the VPN
 * file descriptor and forwards TCP/UDP connections into the sing-box mixed
 * inbound on loopback.
 */
object HevTunRuntime {

    private const val CONFIG_FILE = "hev-socks5-tunnel.yaml"

    private val controller by lazy { HevTunController(NativeHevTunBackend) }

    fun start(context: Context, tunFd: Int) {
        controller.start(File(context.filesDir, CONFIG_FILE), buildConfig(), tunFd)
    }

    fun stop() {
        // Never load the native library just to stop something that never started.
        if (controller.running) controller.stop()
    }

    fun buildConfig(): HevTunConfig {
        return HevTunConfig(
            mtu = DataStore.mtu,
            ipv4 = VpnService.PRIVATE_VLAN4_CLIENT,
            ipv6 = if (DataStore.ipv6Mode != IPv6Mode.DISABLE) VpnService.PRIVATE_VLAN6_CLIENT else null,
            socksAddress = LOCALHOST,
            socksPort = DataStore.mixedPort,
            mapDns = if (DataStore.enableFakeDns) HevTunConfig.MapDns(
                address = VpnService.PRIVATE_VLAN4_ROUTER,
                network = VpnService.HEV_MAPDNS_VLAN4,
                netmask = VpnService.HEV_MAPDNS_NETMASK,
            ) else null,
        )
    }
}
