#!/usr/bin/env bash
#
# Packages TgCli into ready-to-run, downloadable bundles under dist/:
#   - TgCli-linux-x64.tar.gz : tg + libtdjson.so side by side (just run ./tg)
#   - TgCli-win64.zip        : tg.exe + bin/*.dll
#
# The Linux library to bundle is found automatically (~/.local/bin or /usr/lib)
# or can be pointed at explicitly with LIBTDJSON=/path/to/libtdjson.so.
#
# NOTE: the bundled libtdjson.so is dynamically linked; it requires the host to
# have a glibc/OpenSSL at least as new as the machine it was built on. Build on
# an old baseline (e.g. an older distro / container) for the widest reach.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

DIST="$ROOT/dist"
rm -rf "$DIST"
mkdir -p "$DIST"

# --- locate the Linux TDLib library to bundle ---
LIBTDJSON="${LIBTDJSON:-}"
if [[ -z "$LIBTDJSON" ]]; then
  for candidate in "$HOME"/.local/bin/libtdjson.so.* /usr/lib/libtdjson.so.* /usr/local/lib/libtdjson.so.*; do
    if [[ -e "$candidate" && ! -L "$candidate" ]]; then
      LIBTDJSON="$candidate"
      break
    fi
  done
fi
if [[ -z "$LIBTDJSON" || ! -e "$LIBTDJSON" ]]; then
  echo "error: could not find libtdjson.so to bundle; set LIBTDJSON=/path/to/libtdjson.so" >&2
  exit 1
fi
echo "Bundling libtdjson: $LIBTDJSON"

# --- linux bundle ---
LINUX_DIR="$DIST/TgCli-linux-x64"
mkdir -p "$LINUX_DIR"
echo "Building Linux binary..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -o "$LINUX_DIR/tg" .
soname="$(basename "$LIBTDJSON")"
cp "$LIBTDJSON" "$LINUX_DIR/$soname"
( cd "$LINUX_DIR" && ln -sf "$soname" libtdjson.so )
cp README.md LICENSE "$LINUX_DIR/" 2>/dev/null || true
( cd "$DIST" && tar czf TgCli-linux-x64.tar.gz TgCli-linux-x64 )

# --- windows bundle ---
WIN_DIR="$DIST/TgCli-win64"
mkdir -p "$WIN_DIR/bin"
echo "Building Windows binary..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o "$WIN_DIR/tg.exe" .
cp bin/*.dll "$WIN_DIR/bin/"
cp README.md LICENSE "$WIN_DIR/" 2>/dev/null || true
if command -v zip >/dev/null 2>&1; then
  ( cd "$DIST" && zip -qr TgCli-win64.zip TgCli-win64 )
elif command -v python3 >/dev/null 2>&1; then
  ( cd "$DIST" && python3 -c "import shutil; shutil.make_archive('TgCli-win64', 'zip', '.', 'TgCli-win64')" )
else
  echo "warning: no 'zip' or python3; falling back to tar.gz for Windows" >&2
  ( cd "$DIST" && tar czf TgCli-win64.tar.gz TgCli-win64 )
fi

echo "Done. Artifacts:"
ls -la "$DIST"/*.tar.gz "$DIST"/*.zip 2>/dev/null
