#!/bin/bash
# Move sing-box to another upstream tag/commit and re-run the checks.
#
#   ./run lib core update v1.14.5        # specific tag
#   ./run lib core update                # latest stable tag (no -alpha/-beta/-rc)
#
# Afterwards fix whatever the check reports, then commit the new pointer:
#   git add libcore/third_party/sing-box libcore/go.mod libcore/go.sum
set -euo pipefail
source "$(dirname "$0")/common.sh"

core_init_submodules   # also reverts patches
cd "$SING_BOX_DIR"
old=$(git rev-parse HEAD)
git fetch --tags --force origin

ref=${1:-}
if [ -z "$ref" ]; then
  ref=$(git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n1)
  [ -n "$ref" ] || core_die "no stable tag found"
fi
core_log "checkout $ref"
git checkout -q "$ref"
new=$(git rev-parse HEAD)
[ "$old" != "$new" ] || echo "(already at $ref)"

core_log "API surface changed between $(git describe --tags --always "$old") and $(git describe --tags --always "$new") (what libcore implements/uses):"
git diff --stat "$old" "$new" -- adapter option dns/transport route/network.go common/dialer constant | tail -n 25

core_log "apply NekoBox patches"
if ! core_patch; then
  cat >&2 <<MSG

A patch no longer applies. The patched files are small; re-create it by hand:
  cd libcore/third_party/sing-box
  git apply --3way ../../../buildScript/lib/core/patches/<patch>   # resolve conflicts
  git diff > /tmp/x.patch   # or: git diff -- <files> > buildScript/lib/core/patches/<patch>
MSG
  exit 1
fi

# keep libcore's dependency versions in line with the new sing-box go.mod
core_log "sync libcore/go.mod with sing-box go.mod (go mod tidy)"
(cd "$LIBCORE_DIR" && go mod tidy) || echo "go mod tidy failed; fix go.mod requirements for the new sing-box" >&2

exec "$REPO_ROOT/buildScript/lib/core/check.sh"
