package moe.matsuri.nb4a.hevtun

/**
 * Pure (Android-free) description of a hev-socks5-tunnel run and its YAML
 * rendering, so the generated config can be unit tested on the JVM.
 */
data class HevTunConfig(
    val mtu: Int,
    val ipv4: String,
    /** null disables IPv6 on the tunnel. */
    val ipv6: String? = null,
    val socksAddress: String,
    val socksPort: Int,
    /** Credentials are only written when both are set to something non-null. */
    val username: String? = null,
    val password: String? = null,
    val mapDns: MapDns? = null,
    val logLevel: String = "warn",
) {

    /** hev "mapdns": answers DNS locally with addresses from [network] and maps them back to domains. */
    data class MapDns(
        val address: String,
        val port: Int = 53,
        val network: String,
        val netmask: String,
        val cacheSize: Int = 10000,
    )

    fun toYaml(): String = buildString {
        appendLine("tunnel:")
        appendLine("  mtu: $mtu")
        appendLine("  ipv4: ${ipv4.yamlString()}")
        if (ipv6 != null) appendLine("  ipv6: ${ipv6.yamlString()}")
        appendLine("socks5:")
        appendLine("  port: $socksPort")
        appendLine("  address: ${socksAddress.yamlString()}")
        appendLine("  udp: 'udp'")
        appendLine("  pipeline: true")
        if (username != null && password != null) {
            appendLine("  username: ${username.yamlString()}")
            appendLine("  password: ${password.yamlString()}")
        }
        if (mapDns != null) {
            appendLine("mapdns:")
            appendLine("  address: ${mapDns.address.yamlString()}")
            appendLine("  port: ${mapDns.port}")
            appendLine("  network: ${mapDns.network.yamlString()}")
            appendLine("  netmask: ${mapDns.netmask.yamlString()}")
            appendLine("  cache-size: ${mapDns.cacheSize}")
        }
        appendLine("misc:")
        appendLine("  log-level: ${logLevel.yamlString()}")
    }

    companion object {
        /**
         * Renders [this] as a YAML double-quoted scalar. Double quotes (rather than
         * single) are used because single-quoted scalars fold line breaks, which
         * would silently corrupt a password containing a newline.
         */
        internal fun String.yamlString(): String = buildString {
            append('"')
            for (c in this@yamlString) {
                when (c) {
                    '\\' -> append("\\\\")
                    '"' -> append("\\\"")
                    '\n' -> append("\\n")
                    '\r' -> append("\\r")
                    '\t' -> append("\\t")
                    else -> if (c < ' ' || c == '\u007f') {
                        append("\\x").append(c.code.toString(16).padStart(2, '0'))
                    } else append(c)
                }
            }
            append('"')
        }
    }
}
