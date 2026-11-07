#!/bin/bash
set -e


if [ -d .build ]; then
    chmod -R 777 .build
fi
rm -rf .build 2>/dev/null

if [ -z "$GOPATH" ]; then
    GOPATH=$(go env GOPATH)
fi

# gomobile is a git submodule (libcore/third_party/gomobile); go.mod replaces
# golang.org/x/mobile with it. Pinned commit: see `git submodule status`.
pushd third_party/gomobile
# Install the matching gomobile tools once per GOPATH.
if [ ! -f "$GOPATH/bin/gomobile-matsuri" ]; then
    pushd cmd/gomobile
    go install -v
    popd
    pushd cmd/gobind
    go install -v
    popd
    mv "$GOPATH/bin/gomobile" "$GOPATH/bin/gomobile-matsuri"
    mv "$GOPATH/bin/gobind" "$GOPATH/bin/gobind-matsuri"
fi
popd

GOBIND="$GOPATH/bin/gobind-matsuri" "$GOPATH/bin/gomobile-matsuri" init
