#!/usr/bin/env bash
#
# Builds a portable Linux libtdjson.so inside an old-glibc container (Ubuntu
# 20.04 => glibc 2.31), so the resulting library runs on essentially every
# Linux distro from ~2020 onward — much wider than a library built on a
# bleeding-edge host.
#
# Output: dist/tdlib-build/libtdjson.so* and dist/tdlib-build/VERSION
#
# Usage: scripts/build-tdlib-portable.sh [git-ref]
#   git-ref defaults to "master".
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/dist/tdlib-build"
REF="${1:-master}"
JOBS="${JOBS:-4}"

mkdir -p "$OUT"

docker run --rm \
  -v "$OUT:/out" \
  -e REF="$REF" -e JOBS="$JOBS" \
  ubuntu:20.04 bash -euo pipefail -c '
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq git cmake g++ make zlib1g-dev libssl-dev gperf ca-certificates >/dev/null
    cd /tmp
    git clone --depth 1 --branch "$REF" https://github.com/tdlib/td.git 2>/dev/null \
      || git clone https://github.com/tdlib/td.git
    cd td
    [ "$REF" != "master" ] && git checkout "$REF" || true
    mkdir -p build && cd build
    cmake -DCMAKE_BUILD_TYPE=Release -DTD_ENABLE_LTO=OFF .. >/dev/null
    cmake --build . --target tdjson -j"$JOBS"
    cp -av libtdjson.so* /out/
    ( cd /tmp/td && git rev-parse HEAD; cat CMakeLists.txt | grep -iE "VERSION [0-9]" | head -1 ) > /out/VERSION || true
    strip /out/libtdjson.so* 2>/dev/null || true
    echo "BUILD_OK"
  '
echo "Portable libtdjson built into: $OUT"
ls -la "$OUT"