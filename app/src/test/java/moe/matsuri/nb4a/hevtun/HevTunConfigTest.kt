package moe.matsuri.nb4a.hevtun

import moe.matsuri.nb4a.hevtun.HevTunConfig.Companion.yamlString
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.yaml.snakeyaml.Yaml

class HevTunConfigTest {

    private val base = HevTunConfig(
        mtu = 9000,
        ipv4 = "172.19.0.1",
        socksAddress = "127.0.0.1",
        socksPort = 2080,
    )

    @Suppress("UNCHECKED_CAST")
    private fun parse(c: HevTunConfig) = Yaml().load<Map<String, Map<String, Any?>>>(c.toYaml())

    @Test
    fun minimalConfig() {
        val y = parse(base)
        assertEquals(9000, y["tunnel"]!!["mtu"])
        assertEquals("172.19.0.1", y["tunnel"]!!["ipv4"])
        assertFalse(y["tunnel"]!!.containsKey("ipv6"))
        assertEquals("127.0.0.1", y["socks5"]!!["address"])
        assertEquals(2080, y["socks5"]!!["port"])
        assertEquals("udp", y["socks5"]!!["udp"])
        assertEquals(true, y["socks5"]!!["pipeline"])
        assertEquals("warn", y["misc"]!!["log-level"])
        assertFalse(y.containsKey("mapdns"))
        assertFalse(y["socks5"]!!.containsKey("username"))
    }

    @Test
    fun ipv6IsWrittenOnlyWhenSet() {
        val y = parse(base.copy(ipv6 = "fdfe:dcba:9876::1"))
        assertEquals("fdfe:dcba:9876::1", y["tunnel"]!!["ipv6"])
    }

    @Test
    fun mapDnsSection() {
        val y = parse(
            base.copy(
                mapDns = HevTunConfig.MapDns(
                    address = "172.19.0.2", network = "100.64.0.0", netmask = "255.192.0.0",
                )
            )
        )
        val m = y["mapdns"]!!
        assertEquals("172.19.0.2", m["address"])
        assertEquals(53, m["port"])
        assertEquals("100.64.0.0", m["network"])
        assertEquals("255.192.0.0", m["netmask"])
        assertEquals(10000, m["cache-size"])
    }

    @Test
    fun credentialsNeedBothFields() {
        assertFalse(parse(base.copy(username = "u"))["socks5"]!!.containsKey("username"))
        assertFalse(parse(base.copy(password = "p"))["socks5"]!!.containsKey("password"))
        val s = parse(base.copy(username = "u", password = "p"))["socks5"]!!
        assertEquals("u", s["username"])
        assertEquals("p", s["password"])
    }

    @Test
    fun specialCharactersSurviveRoundTrip() {
        val tricky = "it's \"q\" \\ back\nline\ttab: #not-comment {x}"
        val s = parse(base.copy(username = tricky, password = tricky))["socks5"]!!
        assertEquals(tricky, s["username"])
        assertEquals(tricky, s["password"])
    }

    @Test
    fun yamlStringEscaping() {
        assertEquals("\"plain\"", "plain".yamlString())
        assertEquals("\"a\\\"b\"", "a\"b".yamlString())
        assertEquals("\"a\\\\b\"", "a\\b".yamlString())
        assertEquals("\"a\\nb\"", "a\nb".yamlString())
        assertEquals("\"\\x01\"", "\u0001".yamlString())
        assertEquals("\"\u4e2d\u6587\"", "\u4e2d\u6587".yamlString())
    }

    @Test
    fun outputIsLineTerminated() {
        assertTrue(base.toYaml().endsWith("\n"))
    }
}
