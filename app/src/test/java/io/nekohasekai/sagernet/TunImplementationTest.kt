package io.nekohasekai.sagernet

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File
import javax.xml.parsers.DocumentBuilderFactory

class TunImplementationTest {

    @Test
    fun onlyHevBypassesSingBoxTun() {
        assertTrue(TunImplementation.usesSingBoxTun(TunImplementation.GVISOR))
        assertTrue(TunImplementation.usesSingBoxTun(TunImplementation.SYSTEM))
        assertTrue(TunImplementation.usesSingBoxTun(TunImplementation.MIXED))
        assertFalse(TunImplementation.usesSingBoxTun(TunImplementation.HEV))
    }

    @Test
    fun valuesAreDistinctAndStable() {
        // persisted in user settings: must never be renumbered
        assertEquals(listOf(0, 1, 2, 3), listOf(
            TunImplementation.GVISOR, TunImplementation.SYSTEM, TunImplementation.MIXED, TunImplementation.HEV,
        ))
    }

    // --- resources: the settings dropdown must offer every implementation ---

    private fun res(path: String) = File("src/main/res/$path").also { assertTrue("$it missing", it.exists()) }

    private fun items(file: File, arrayName: String): List<String> {
        val doc = DocumentBuilderFactory.newInstance().newDocumentBuilder().parse(file)
        val arrays = doc.getElementsByTagName("string-array")
        for (i in 0 until arrays.length) {
            val a = arrays.item(i)
            if (a.attributes.getNamedItem("name").nodeValue == arrayName) {
                val kids = (a as org.w3c.dom.Element).getElementsByTagName("item")
                return (0 until kids.length).map { kids.item(it).textContent }
            }
        }
        throw AssertionError("array $arrayName not found")
    }

    @Test
    fun settingsDropdownListsHev() {
        val arrays = res("values/arrays.xml")
        val labels = items(arrays, "tun_implementation")
        assertEquals(4, labels.size)
        assertTrue(labels[TunImplementation.HEV].contains("Hev"))

        val prefs = res("xml/global_preferences.xml").readText()
        val pref = Regex("""<[^>]*app:key="tunImplementation"[^>]*>""").find(prefs)!!.value
        val valuesArray = Regex("""@array/(\w+)""").findAll(pref).map { it.groupValues[1] }.toList()
            .first { it != "tun_implementation" }
        assertEquals(
            (0 until labels.size).map { it.toString() },
            items(arrays, valuesArray)
        )
    }
}
