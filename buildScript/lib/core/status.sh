#!/bin/bash
# Content hash of everything that influences libcore.aar (used for CI cache keys).
# Works without initialised submodules: pins come from the git tree.
set -euo pipefail
cd "$(dirname "$0")/../../.."
{
  git ls-files -z libcore | xargs -0 -I{} sh -c '[ -f "{}" ] && cat "{}"; true'
  git ls-tree HEAD libcore/third_party
} | sha1sum
