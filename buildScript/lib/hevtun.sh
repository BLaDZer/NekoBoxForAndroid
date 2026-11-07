#!/bin/bash
# Build libhev-socks5-tunnel.so (JNI) for all Android ABIs into app/src/main/jniLibs.
#
# Source: libcore/third_party/hev-socks5-tunnel (git submodule, initialized when missing).
# The JNI entry points are bound via RegisterNatives to the class named by
# -DPKGNAME/-DCLSNAME below; keep them in sync with moe.matsuri.nb4a.hevtun.HevTunNative.
#
# Usage: ./run lib hevtun
# Env:   HEV_SRC  alternative hev-socks5-tunnel checkout
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/../init/env_ndk.sh"

HEV_SRC="${HEV_SRC:-$REPO_DIR/libcore/third_party/hev-socks5-tunnel}"
ABIS="armeabi-v7a arm64-v8a x86 x86_64"
OUT_DIR="$REPO_DIR/app/src/main/jniLibs"
BUILD_DIR="$REPO_DIR/app/build/hevtun"

if [ ! -f "$HEV_SRC/Android.mk" ] || [ ! -f "$HEV_SRC/third-part/lwip/Android.mk" ]; then
  echo ">> initializing hev-socks5-tunnel submodule"
  git -C "$REPO_DIR" submodule update --init --recursive libcore/third_party/hev-socks5-tunnel
fi

# Persistent obj/libs dirs keep ndk-build incremental.
mkdir -p "$BUILD_DIR/jni"
ln -sfn "$HEV_SRC" "$BUILD_DIR/jni/hev-socks5-tunnel"
echo 'include $(call all-subdir-makefiles)' > "$BUILD_DIR/jni/Android.mk"

(
  cd "$BUILD_DIR"
  "$ANDROID_NDK_HOME/ndk-build" \
    NDK_PROJECT_PATH=. \
    APP_BUILD_SCRIPT=jni/Android.mk \
    "APP_ABI=$ABIS" \
    APP_PLATFORM=android-23 \
    "APP_CFLAGS=-O3 -DPKGNAME=moe/matsuri/nb4a/hevtun -DCLSNAME=HevTunNative" \
    "APP_LDFLAGS=-Wl,--build-id=none -Wl,--hash-style=gnu" \
    NDK_LIBS_OUT="$BUILD_DIR/libs" \
    NDK_OUT="$BUILD_DIR/obj"
)

for abi in $ABIS; do
  mkdir -p "$OUT_DIR/$abi"
  cp -f "$BUILD_DIR/libs/$abi/libhev-socks5-tunnel.so" "$OUT_DIR/$abi/"
  echo ">> installed $OUT_DIR/$abi/libhev-socks5-tunnel.so"
done
