#!/bin/bash
# Verify that libcore still builds and behaves against the pinned sing-box.
# Needs only Go (+ git); no Android NDK/SDK. Runs every step and prints a summary.
#
#   ./run lib core check              # everything below
#   ./run lib core check --android    # additionally cross-compile for android/arm64 (needs NDK)
#
# Steps: submodules -> patches -> go.mod tidy -> build -> vet -> libcore tests
#        (incl. app-config compatibility fixtures)
set -uo pipefail
source "$(dirname "$0")/common.sh"

android=0
for arg in "$@"; do
  case "$arg" in
    --android) android=1 ;;
    *) core_die "unknown argument: $arg" ;;
  esac
done

export CGO_ENABLED=1
declare -a names results
step() { # step <name> <command...>
  local name=$1; shift
  core_log "$name"
  if "$@"; then results+=(PASS); else results+=(FAIL); fi
  names+=("$name")
}

check_submodules() {
  local out
  out=$(git -C "$REPO_ROOT" submodule status libcore/third_party/*) || return 1
  echo "$out"
  ! grep -q '^-' <<<"$out" || { echo "uninitialised submodule: run ./run lib core get_source" >&2; return 1; }
}

check_patches() { core_patch; }

check_tidy() {
  cd "$LIBCORE_DIR" || return 1
  local tmp; tmp=$(mktemp -d)
  cp go.mod go.sum "$tmp"
  go mod tidy >/dev/null 2>"$tmp/err"; local rc=$?
  if [ $rc -ne 0 ]; then cat "$tmp/err" >&2; cp "$tmp/go.mod" "$tmp/go.sum" .; rm -rf "$tmp"; return 1; fi
  if ! diff -u "$tmp/go.mod" go.mod || ! diff -u "$tmp/go.sum" go.sum; then
    echo "go.mod/go.sum were not tidy (now updated in the working tree; review and commit)" >&2
    rm -rf "$tmp"; return 1
  fi
  rm -rf "$tmp"
}

# -gcflags=-e: report every compile error instead of stopping after 10 per package.
check_build() { cd "$LIBCORE_DIR" && go build -gcflags=-e -tags "$CORE_TAGS" -ldflags=-checklinkname=0 ./...; }
check_vet()   { cd "$LIBCORE_DIR" && go vet -tags "$CORE_TAGS" ./...; }

check_tests() {
  cd "$LIBCORE_DIR" || return 1
  local log; log=$(mktemp)
  go test -count=1 -v -tags "$CORE_TAGS" -ldflags=-checklinkname=0 ./... 2>&1 | tee "$log" | grep -E '^(---|===|FAIL|ok|panic)|_test.go' | grep -v '^=== RUN'
  local rc=${PIPESTATUS[0]}
  echo "--- deprecated sing-box features still used by app configs (migration TODOs):"
  grep -h 'deprecated (migration TODO)' "$log" | sed 's/.*TODO): /    /' | sort -u
  rm -f "$log"
  return $rc
}

check_android() {
  source "$REPO_ROOT/buildScript/init/env_ndk.sh" || return 1
  local cc="$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android23-clang"
  [ -x "$cc" ] || { echo "NDK clang not found: $cc" >&2; return 1; }
  # Type-check the Android-specific files too (assets_android.go, dns_android.go, ...).
  # The gomobile-bound root package needs generated cgo glue, so build it as a library.
  cd "$LIBCORE_DIR" && GOOS=android GOARCH=arm64 CC="$cc" go build -gcflags=-e -tags "$CORE_TAGS" \
    -ldflags=-checklinkname=0 -buildmode=c-archive -o /dev/null . 2>&1 | grep -v 'gomobile' ; return "${PIPESTATUS[0]}"
}

step "submodules initialised" check_submodules
step "sing-box patches applied" check_patches
if [ "${results[1]}" = FAIL ]; then
  echo "Patches do not apply; the remaining steps would test an unpatched core. Fix patches first." >&2
else
  step "go.mod / go.sum tidy" check_tidy
  step "go build (host)" check_build
  step "go vet" check_vet
  step "libcore tests + app config fixtures" check_tests
  [ $android -eq 0 ] || step "go build android/arm64 (NDK)" check_android
fi

echo
echo "== libcore check against sing-box $(git -C "$SING_BOX_DIR" describe --tags --always) =="
fail=0
for i in "${!names[@]}"; do
  printf '  %-4s %s\n' "${results[$i]}" "${names[$i]}"
  [ "${results[$i]}" = PASS ] || fail=1
done
exit $fail
