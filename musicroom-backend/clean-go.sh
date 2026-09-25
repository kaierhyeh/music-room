#!/usr/bin/env bash
# Remove everything installed by `make tools`:
#   - tool binaries (sqlc, migrate, swag, k6) from GOBIN or GOPATH/bin
#   - the Go module cache (GOMODCACHE, e.g. ~/go/pkg/mod)
#   - the Go build cache (GOCACHE, e.g. ~/.cache/go-build)
#
# Usage: ./clean-go-tools.sh [--dry-run] [--yes]
#   --dry-run  show what would be removed, change nothing
#   --yes      skip the confirmation prompt

set -euo pipefail

DRY_RUN=false
ASSUME_YES=false

for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=true ;;
    --yes|-y)  ASSUME_YES=true ;;
    -h|--help)
      sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "Unknown option: $arg" >&2
      exit 1
      ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  echo "Error: 'go' not found in PATH." >&2
  exit 1
fi

TOOLS=(sqlc migrate swag k6)

GOBIN_DIR="$(go env GOBIN)"
if [[ -z "$GOBIN_DIR" ]]; then
  # GOPATH may contain several entries; binaries go to the first one.
  GOBIN_DIR="$(go env GOPATH | cut -d: -f1)/bin"
fi
MODCACHE="$(go env GOMODCACHE)"
BUILDCACHE="$(go env GOCACHE)"

size_of() {
  if [[ -e "$1" ]]; then
    du -sh "$1" 2>/dev/null | cut -f1
  else
    echo "absent"
  fi
}

echo "== Targets =="
for tool in "${TOOLS[@]}"; do
  path="$GOBIN_DIR/$tool"
  if [[ -e "$path" ]]; then
    echo "  binary       $path ($(size_of "$path"))"
  else
    echo "  binary       $path (absent)"
  fi
done
echo "  module cache $MODCACHE ($(size_of "$MODCACHE"))"
echo "  build cache  $BUILDCACHE ($(size_of "$BUILDCACHE"))"
echo
echo "Note: the module cache is shared by ALL your Go projects;"
echo "their dependencies will be re-downloaded on the next build."
echo

if $DRY_RUN; then
  echo "Dry run: nothing removed."
  exit 0
fi

if ! $ASSUME_YES; then
  read -r -p "Proceed? [y/N] " answer
  [[ "$answer" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }
fi

echo "== Disk usage before =="
df -h "$HOME" | tail -n 1

for tool in "${TOOLS[@]}"; do
  path="$GOBIN_DIR/$tool"
  if [[ -e "$path" ]]; then
    rm -f "$path"
    echo "Removed $path"
  fi
done

# `go clean -modcache` is required: module cache files are read-only,
# so a plain `rm -rf` fails with "permission denied".
echo "Cleaning module cache..."
go clean -modcache

echo "Cleaning build cache..."
go clean -cache

echo "== Disk usage after =="
df -h "$HOME" | tail -n 1
echo "Done."
