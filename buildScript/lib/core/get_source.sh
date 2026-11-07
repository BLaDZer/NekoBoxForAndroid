#!/bin/bash
# Fetch the core sources (git submodules) and apply the NekoBox patches to sing-box.
set -euo pipefail
source "$(dirname "$0")/common.sh"

core_log "init submodules"
core_init_submodules
core_log "patch sing-box ($(git -C "$SING_BOX_DIR" describe --tags --always))"
core_patch || core_die "some patches do not apply, see ./run lib core update"
