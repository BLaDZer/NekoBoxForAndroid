# Building NekoBox for Android

This document describes how to build **NekoBox for Android** from source.

The build consists of two stages:

1. **Native core** — build `libcore.aar` from the Go sources in `libcore/`.
2. **Android app** — build the APK with Gradle.

The repository does **not** include `app/libs/libcore.aar`, so running
`./gradlew assemble...` directly will fail until the native core has been built.

> All commands below assume the repository root:
>
> ```bash
> cd /home/blad/Загрузки/build/NekoBoxForAndroid
> ```

---

## 1. Requirements

| Component | Required version |
|---|---|
| JDK | 17 or 21 (JDK 17 recommended) |
| Android SDK Platform | `android-35` |
| Android Build Tools | `35.0.1` (pinned exactly) |
| Android NDK | `29.0.14206865` |
| Go | 1.25.5 minimum (`libcore/go.mod` declares toolchain `go1.25.11`) |
| Gradle | provided by wrapper (`8.10.2`) |
| Other tools | `git`, `curl`, `xz`, `unzip` |

The project uses:

- Gradle `8.10.2` (`gradle/wrapper/gradle-wrapper.properties`)
- Android Gradle Plugin `8.8.1` (`buildSrc/build.gradle.kts`)
- Kotlin `2.0.21`
- `compileSdk = 35`, `minSdk = 21`, `targetSdk = 35`
- `buildToolsVersion = "35.0.1"`, configured in
  `buildSrc/src/main/kotlin/Helpers.kt`

These values are hard-coded, so installing the exact Build Tools and NDK
versions avoids build errors.

---

## 2. Current environment status

This is the state of the machine where this guide was written. Use it as a
checklist — it will become stale over time.

| Requirement | Status |
|---|---|
| JDK 17/21 | Missing — only Java 27 installed; Gradle 8.10.2 / AGP 8.8.1 will not run on it |
| Android SDK | Present at `~/Android/Sdk` |
| `platforms;android-35` | Installed |
| `build-tools;35.0.1` | Missing |
| `ndk;29.0.14206865` | Missing (only NDK `27.1.12297006` installed) |
| Go | Present: `go1.27.1` (project is tested with Go 1.25) |
| `local.properties` | Missing |
| `app/libs/libcore.aar` | Missing |
| `app/src/main/assets/sing-box` (geoip/geosite) | Missing |
| Submodules `libcore/third_party/{sing-box,libneko,gomobile}` | Fetched automatically by the core build (or `git submodule update --init`) |

---

## 3. Build steps

### Step 1 — Install JDK 17 and switch to it

Java 27 is too new for Gradle 8.10.2:

```bash
sudo pacman -S --needed jdk17-openjdk

export JAVA_HOME=/usr/lib/jvm/java-17-openjdk
export PATH="$JAVA_HOME/bin:$PATH"
java -version   # should report 17
```

### Step 2 — Install the exact Android SDK components

```bash
export ANDROID_HOME="$HOME/Android/Sdk"

yes | "$ANDROID_HOME/tools/bin/sdkmanager" --sdk_root="$ANDROID_HOME" --licenses

"$ANDROID_HOME/tools/bin/sdkmanager" --sdk_root="$ANDROID_HOME" \
  "platforms;android-35" \
  "build-tools;35.0.1" \
  "ndk;29.0.14206865" \
  "platform-tools"
```

### Step 3 — Create `local.properties`

`local.properties` is git-ignored and tells Gradle where the SDK and NDK are.

```bash
cat > local.properties <<EOF
sdk.dir=$HOME/Android/Sdk
ndk.dir=$HOME/Android/Sdk/ndk/29.0.14206865
EOF
```

For a **signed release** build, append the signing properties:

```properties
KEYSTORE_PASS=your_keystore_password
ALIAS_NAME=your_key_alias
ALIAS_PASS=your_key_password
```

The signing configuration is read in
`buildSrc/src/main/kotlin/Helpers.kt` (`setupAppCommon()`). It uses
`release.keystore` from the repository root. If these properties are absent,
the release APK is produced **unsigned**. Debug builds are signed with the
standard Android debug key.

### Step 4 — Put Go binaries on `PATH`

`libcore/init.sh` calls `gomobile-matsuri` by name, so it must be on `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

`init.sh` installs the tools from the pinned `MatsuriDayo/gomobile` fork, which is
the git submodule `libcore/third_party/gomobile`. The checkout must be kept
because `libcore/go.mod` replaces `golang.org/x/mobile` with it.

### Step 5 — Download geoip/geosite assets

```bash
./run init action gradle
```

This executes `buildScript/lib/assets.sh`, which downloads the latest
`geoip.db` and `geosite.db` from the `SagerNet/sing-geoip` and
`SagerNet/sing-geosite` GitHub releases and stores them, xz-compressed, in:

```text
app/src/main/assets/sing-box/
```

### Step 6 — Build the native core

This is the slow part (first run can take 10–20 minutes or more).

```bash
./run lib core
```

This script chain does the following:

1. Initialises the git submodules in `libcore/third_party/`
   (`sing-box` = official `SagerNet/sing-box`, `libneko`, `gomobile`) and applies
   the single compatibility patch in `buildScript/lib/core/patches/` (stops
   `box.New` from forcing the cache file and traffic manager when a platform log
   writer is set) to the sing-box working tree (patches are never committed; the submodule pointer stays on a
   pristine upstream commit).
2. Installs `gomobile-matsuri` / `gobind-matsuri` into `$(go env GOPATH)/bin`.
3. Runs `./run lib core check` (build, vet, tests; see below).
4. Runs `go mod tidy`, then `gomobile bind` with `-androidapi 23`, the build tags
   in `buildScript/lib/core/common.sh` (`CORE_TAGS`) and
   `-ldflags='-s -w -checklinkname=0 -X ...constant.Version=<sing-box tag>-neko'`.
5. Copies the resulting AAR to `app/libs/libcore.aar`.

`libcore/go.mod` points at the submodules:

```go
replace github.com/matsuridayo/libneko => ./third_party/libneko
replace github.com/sagernet/sing-box => ./third_party/sing-box
replace golang.org/x/mobile => ./third_party/gomobile
```

#### Checking the core without the Android toolchain

```bash
./run lib core check              # submodules, patches, go.mod tidy, build, vet, tests
./run lib core check --android    # + cross-compile for android/arm64 (needs NDK)
```

Only Go is needed. The tests include `libcore/config_compat_test.go`, which loads
configs shaped like `ConfigBuilder.kt` output (`libcore/testdata/*.json`) into the
real sing-box core, so removed/renamed config fields fail here instead of at
runtime on a phone. It also lists deprecated features the app still emits. When
`ConfigBuilder.kt` starts emitting something new, add it to a fixture.

#### Updating sing-box

```bash
./run lib core update            # latest stable upstream tag
./run lib core update v1.14.5    # a specific tag/commit
```

This reverts the patches, checks out the new upstream ref, shows what changed in
the API surface libcore implements, re-applies the patches, runs `go mod tidy`
and then the check above. Fix what the check reports (typically
`libcore/platform_box.go`, `box_include.go`, `dns_box.go`), then commit:

```bash
git add libcore/third_party/sing-box libcore/go.mod libcore/go.sum
```

If the patch no longer applies, re-make the one-line change in
`box.go` (`needCacheFile` condition) and regenerate it. Everything else Neko needs
(stats/URL-test client in `libcore/boxapi`, `geoip:`/`geosite:` rule-sets in
`geo_rule_set.go`, selector notification in `box.go`) lives in `libcore` and uses
only exported sing-box APIs.

Verify the core was produced:

```bash
ls -l app/libs/libcore.aar
```

### Step 7 — Build the APK

Release build:

```bash
./gradlew app:assembleOssRelease
```

Debug build (easier to install locally):

```bash
./gradlew app:assembleOssDebug
```

Output locations:

```text
app/build/outputs/apk/oss/release/NekoBox-1.4.2-arm64-v8a.apk
app/build/outputs/apk/oss/debug/NekoBox-1.4.2-arm64-v8a-debug.apk
```

If no signing properties were supplied, the release file is named with an
`-unsigned` suffix instead.

The release variant produces split APKs for four ABIs:

- `armeabi-v7a`
- `arm64-v8a`
- `x86`
- `x86_64`

For normal modern phones, use the **`arm64-v8a`** APK.

---

## 4. Build variants

The project defines these product flavors in
`buildSrc/src/main/kotlin/Helpers.kt`:

| Flavor | Purpose |
|---|---|
| `oss` | Normal open-source build (use this) |
| `fdroid` | F-Droid build |
| `play` | Google Play build |
| `preview` | Preview build using `PRE_VERSION_NAME` |

Examples:

```bash
./gradlew app:assembleOssRelease
./gradlew app:assembleOssDebug
./gradlew app:assembleFdroidRelease
./gradlew app:assemblePlayRelease
./gradlew app:assemblePreviewRelease
```

Version metadata is read from `nb4a.properties` at the repository root:

```properties
PACKAGE_NAME=moe.nb4a
VERSION_NAME=1.4.2
PRE_VERSION_NAME=pre-1.4.2-20260202-1
VERSION_CODE=46
```

Note that `versionCode` is multiplied by 5 in `Helpers.kt`.

---

## 5. How CI builds the project

The GitHub Actions workflow (`.github/workflows/release.yml`) performs the
same steps:

```bash
# Native core
./run lib core

# App
echo "sdk.dir=${ANDROID_HOME}" > local.properties
echo "ndk.dir=${ANDROID_HOME}/ndk/29.0.14206865" >> local.properties
export LOCAL_PROPERTIES="${{ secrets.LOCAL_PROPERTIES }}"
./run init action gradle
./gradlew app:assembleOssRelease
```

CI uses Go `1.25.11` (pinned — Go 1.27 removed the `gotypesalias`
GODEBUG setting used by the pinned gomobile fork), JDK 17, NDK
`29.0.14206865`, and an optional Base64-encoded `LOCAL_PROPERTIES`
environment variable containing signing data. The Android SDK command-line
tools are installed with `android-actions/setup-android`, so the same
workflow also works when run locally through `act`.

---

## 6. Troubleshooting

### `Unsupported class file major version` / Gradle fails immediately

You are still on Java 27. Set `JAVA_HOME` to JDK 17:

```bash
export JAVA_HOME=/usr/lib/jvm/java-17-openjdk
export PATH="$JAVA_HOME/bin:$PATH"
```

### `NDK not found`

Install NDK `29.0.14206865`:

```bash
"$ANDROID_HOME/tools/bin/sdkmanager" --sdk_root="$ANDROID_HOME" \
  "ndk;29.0.14206865"
```

Alternatively export one of:

```bash
export ANDROID_NDK_HOME=/path/to/ndk
# or
export NDK=/path/to/ndk
```

The lookup order is defined in `buildScript/init/env_ndk.sh`:

1. `$ANDROID_HOME/ndk/29.0.14206865`
2. `$ANDROID_NDK_HOME`
3. `$NDK`
4. `$ANDROID_HOME/ndk-bundle`

### `build-tools 35.0.1 not found`

Install the exact version — it is hard-coded in
`buildSrc/src/main/kotlin/Helpers.kt`:

```bash
"$ANDROID_HOME/tools/bin/sdkmanager" --sdk_root="$ANDROID_HOME" \
  "build-tools;35.0.1"
```

### `gomobile` build errors

- Try Go 1.25 if Go 1.27 is incompatible with the pinned
  `gomobile-matsuri` fork.
- Make sure `$(go env GOPATH)/bin` is on `PATH`.
- Remove partial installs and retry:

  ```bash
  git -C libcore/third_party/gomobile clean -xfd
  rm -f "$(go env GOPATH)/bin/gomobile" \
        "$(go env GOPATH)/bin/gobind" \
        "$(go env GOPATH)/bin/gomobile-matsuri" \
        "$(go env GOPATH)/bin/gobind-matsuri"
  ```

### Release build fails on lint or minification

Try a debug build:

```bash
./gradlew app:assembleOssDebug
```

Or disable shrinking/minification:

```bash
nkmr_minify=0 ./gradlew app:assembleOssRelease
```

`nkmr_minify=0` is handled in `Helpers.kt` and disables both
`isMinifyEnabled` and `isShrinkResources` for the release build type.

### You already have a prebuilt `libcore.aar`

Skip Step 6 and place the file at:

```text
app/libs/libcore.aar
```

### Assets download fails

`./run init action gradle` requires network access to the GitHub API.
If the API is rate-limited, download `geoip.db` and `geosite.db` manually
from the latest `SagerNet/sing-geoip` and `SagerNet/sing-geosite`
releases, compress them with `xz -9`, and place them in
`app/src/main/assets/sing-box/`.

---

## 7. Quick reference

Complete sequence from a clean checkout:

```bash
cd /home/blad/Загрузки/build/NekoBoxForAndroid

# JDK
export JAVA_HOME=/usr/lib/jvm/java-17-openjdk
export PATH="$JAVA_HOME/bin:$PATH"

# Android SDK
export ANDROID_HOME="$HOME/Android/Sdk"
yes | "$ANDROID_HOME/tools/bin/sdkmanager" --sdk_root="$ANDROID_HOME" --licenses
"$ANDROID_HOME/tools/bin/sdkmanager" --sdk_root="$ANDROID_HOME" \
  "platforms;android-35" "build-tools;35.0.1" "ndk;29.0.14206865" "platform-tools"

# Go
export PATH="$PATH:$(go env GOPATH)/bin"

# Project configuration
cat > local.properties <<EOF
sdk.dir=$HOME/Android/Sdk
ndk.dir=$HOME/Android/Sdk/ndk/29.0.14206865
EOF

# Assets
./run init action gradle

# Native core
./run lib core

# APK
./gradlew app:assembleOssRelease
```

Result:

```text
app/build/outputs/apk/oss/release/NekoBox-1.4.2-arm64-v8a.apk
```
