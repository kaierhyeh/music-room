#!/usr/bin/env bash
# Installs k6 into <repo>/.tools from the official prebuilt release, for
# machines with neither root nor Go (`make tools` compiles it with Go).
#
# Pinned to the version the benchmark report was measured with; set K6_VERSION
# to override (e.g. K6_VERSION=v2.3.0). The archive is checked against the
# release's own checksums file before anything is extracted.
#
# Usage: ./scripts/install-k6-without-sudo.sh
#        then .tools/k6 run benchmarks/auth.js

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOLS="$(dirname "$(dirname "$HERE")")/.tools"
K6_VERSION="${K6_VERSION:-v1.8.1}"
K6_VERSION="v${K6_VERSION#v}"

[[ "$(uname -s)" == Linux ]] || { echo "Linux only: on macOS use 'brew install k6'"; exit 1; }
case "$(uname -m)" in
  x86_64)        arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

if [[ -x "$TOOLS/k6" ]] && "$TOOLS/k6" version 2>/dev/null | grep -q "$K6_VERSION"; then
  echo "==> k6 $K6_VERSION already in $TOOLS"
  exit 0
fi

name="k6-$K6_VERSION-linux-$arch"
base="https://github.com/grafana/k6/releases/download/$K6_VERSION"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "==> Downloading $name.tar.gz"
curl -fsSL -o "$tmp/$name.tar.gz" "$base/$name.tar.gz"
curl -fsSL -o "$tmp/checksums.txt" "$base/k6-$K6_VERSION-checksums.txt"

echo "==> Verifying checksum"
(cd "$tmp" && grep " $name.tar.gz\$" checksums.txt | sha256sum -c --quiet -) \
  || { echo "Checksum mismatch: archive discarded"; exit 1; }

mkdir -p "$TOOLS"
tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
mv "$tmp/$name/k6" "$TOOLS/k6"
chmod +x "$TOOLS/k6"

echo "==> $("$TOOLS/k6" version)"
echo "Run the benchmarks with: $TOOLS/k6 run benchmarks/auth.js"
echo "(or add it to PATH: export PATH=\"$TOOLS:\$PATH\")"
