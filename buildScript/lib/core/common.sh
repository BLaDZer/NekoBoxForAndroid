# Shared settings for the libcore (sing-box) build and check scripts.
# Source this file; it does not run anything by itself.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LIBCORE_DIR="$REPO_ROOT/libcore"
THIRD_PARTY="$LIBCORE_DIR/third_party"
SING_BOX_DIR="$THIRD_PARTY/sing-box"
PATCH_DIR="$REPO_ROOT/buildScript/lib/core/patches"

# Must stay in sync with the tags sing-box needs for the features NekoBox exposes.
CORE_TAGS='with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,badlinkname,tfogo_checklinkname0'

core_log() { printf '>> %s\n' "$*"; }
core_die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

# "1.14.4-neko" (tag based) or "1.14.4-5-gabcdef0-neko" for untagged commits.
core_version() {
  local v
  v=$(git -C "$SING_BOX_DIR" describe --tags --always 2>/dev/null) || v=unknown
  printf '%s-neko' "${v#v}"
}

core_ldflags() {
  printf -- "-s -w -checklinkname=0 -X github.com/sagernet/sing-box/constant.Version=%s" "$(core_version)"
}

core_init_submodules() {
  # `git submodule update` refuses to move a patched checkout; patches are
  # re-applied afterwards by core_patch.
  if [ -e "$SING_BOX_DIR/.git" ]; then core_unpatch >/dev/null; fi
  git -C "$REPO_ROOT" submodule update --init libcore/third_party/sing-box \
    libcore/third_party/libneko libcore/third_party/gomobile
}

# Patches are applied to the working tree only (never committed), so the
# submodule pointer always stays on a pristine upstream commit.
# Returns 0 if every patch is applied (already or now), 1 otherwise.
core_patch() {
  local failed=0 patch
  for patch in "$PATCH_DIR"/*.patch; do
    if git -C "$SING_BOX_DIR" apply --reverse --check "$patch" 2>/dev/null; then
      echo "   already applied: $(basename "$patch")"
    elif git -C "$SING_BOX_DIR" apply --check "$patch" 2>/dev/null; then
      git -C "$SING_BOX_DIR" apply "$patch" && echo "   applied:         $(basename "$patch")"
    else
      echo "   DOES NOT APPLY:  $(basename "$patch")" >&2
      failed=1
    fi
  done
  return $failed
}

core_unpatch() {
  local patch
  # reverse order: later patches may touch the same files
  for patch in $(ls "$PATCH_DIR"/*.patch | sort -r); do
    if git -C "$SING_BOX_DIR" apply --reverse --check "$patch" 2>/dev/null; then
      git -C "$SING_BOX_DIR" apply --reverse "$patch" && echo "   reverted: $(basename "$patch")"
    fi
  done
}
