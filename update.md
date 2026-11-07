Done. The project is now on sing-box v1.14.4 (latest stable; v1.15.0-alpha.13 is a pre-release).
What changed
Core dependency
- Switched from the broken Matsuri 1.14.x branch to official SagerNet/sing-box at c005f35300ce76427dc285481ec5695f5b179cd2 (v1.14.4).
- One compatibility patch in buildScript/lib/core/patches/ (cache file); boxapi, geo rule-sets and the selector callback live in libcore against stock sing-box; the version string is injected via ldflags.
- sing-box, libneko and gomobile are git submodules in libcore/third_party/; get_source.sh initialises them and applies the patch series to the working tree. See `./run lib core check|update`.
Build toolchain
- NDK bumped to 29.0.14206865 (buildScript/init/env_ndk.sh, CI workflows, build.md).
- Go requirement 1.25.5 / toolchain go1.25.11 (libcore/go.mod).
- gomobile fork pinned to 17d6af34... as submodule libcore/third_party/gomobile (required by the golang.org/x/mobile replace directive).
- Build tags updated to with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,badlinkname,tfogo_checklinkname0; with_conntrack removed; -androidapi 23 and -checklinkname=0 added.
Native core (libcore/)
- New adapter.PlatformInterface implementation and interface monitor bridge (platform_box.go, interface_monitor.go, platform_java.go).
- Updated DNS transport API (Start(stage, scope), Reset, ExchangeAsync), WireGuard moved from outbound registry to endpoint registry, connection reset via Router().ResetNetwork().
- Added tests: dns_box_test.go, interface_monitor_test.go.
Android app
- ConfigBuilder.kt updated to the 1.14 config schema: endpoints, route action/sniff/resolve rules, default_domain_resolver, new DNS server types, fakeip via type = "fakeip".
- WireGuardFmt.kt now builds an Endpoint_WireGuardOptions.
- SingBoxOptions.java regenerated (new endpoint/DNS/rule fields, new TypeAdapter-based serializer).
- NativeInterface.kt implements the new interface monitor and hardened Wi-Fi state.
- Gson bumped to 2.14.0 (needed for Strictness).
- Removed the old "sniff override" option; DataStore.migrateLegacySettings() maps old value 2 → 1.
- Added ACCESS_WIFI_STATE permission and optional Wi-Fi feature.
Verification
- go build ./... and go test ./... pass in libcore.
- All 5 patches apply cleanly to a fresh v1.14.4 checkout via git am --3way; resulting version is 1.14.4-neko-1.
- Selected sing-box tests (boxapi, nekoutils, common/dialer, protocol/group, protocol/vless, route/rule) pass.
- git diff --check is clean.
Caveats
- A full Android Gradle/APK build was not run here (only JDK 27 is installed; JDK 17, NDK 29, local.properties, and libcore.aar are unavailable).
- Behavioral changes from the sing-box migration: WireGuard is now an endpoint, the "sniff override destination" setting is gone, and DNS/routing use the new 1.14 actions.
- build.md was updated with the new versions and build flow.