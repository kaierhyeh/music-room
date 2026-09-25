#!/usr/bin/env bash

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND="$(dirname "$HERE")"
TOOLS="$(dirname "$BACKEND")/.tools"
ENV_FILE="$BACKEND/.env"
PORT="${PORT:-8080}"

env_value() {
  [[ -f "$ENV_FILE" ]] || return 0
  grep -E "^$1=" "$ENV_FILE" | tail -1 | cut -d= -f2- | sed -E 's/^["'\'']//; s/["'\'']$//'
}

NGROK_AUTHTOKEN="${NGROK_AUTHTOKEN:-$(env_value NGROK_AUTHTOKEN)}"
NGROK_DOMAIN="${NGROK_DOMAIN:-$(env_value NGROK_DOMAIN)}"
GOOGLE_REDIRECT_URL="${GOOGLE_REDIRECT_URL:-$(env_value GOOGLE_REDIRECT_URL)}"

[[ -n "$NGROK_AUTHTOKEN" ]] || { echo "NGROK_AUTHTOKEN is empty in $ENV_FILE (https://dashboard.ngrok.com/get-started/your-authtoken)"; exit 1; }
[[ -n "$NGROK_DOMAIN" ]]    || { echo "NGROK_DOMAIN is empty in $ENV_FILE (https://dashboard.ngrok.com/domains)"; exit 1; }
NGROK_DOMAIN="${NGROK_DOMAIN#https://}"
NGROK_DOMAIN="${NGROK_DOMAIN%/}"

want="https://$NGROK_DOMAIN/auth/google/callback"
if [[ "$GOOGLE_REDIRECT_URL" != "$want" ]]; then
  echo "!! GOOGLE_REDIRECT_URL=$GOOGLE_REDIRECT_URL"
  echo "!! expected: $want — fix .env, then restart the backend (make docker-up)"
fi

if ! curl -s -o /dev/null -m 3 "http://localhost:$PORT/health"; then
  echo "!! nothing answers on localhost:$PORT — start the backend (make docker-up)"
fi

NGROK="$(command -v ngrok || true)"
if [[ -z "$NGROK" ]]; then
  NGROK="$TOOLS/ngrok"
  if [[ ! -x "$NGROK" ]]; then
    [[ "$(uname -s)" == Linux ]] || { echo "Automatic download is Linux only: install ngrok by hand"; exit 1; }
    case "$(uname -m)" in
      x86_64)        arch=amd64 ;;
      aarch64|arm64) arch=arm64 ;;
      *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
    echo "==> Downloading ngrok ($arch) into $TOOLS"
    mkdir -p "$TOOLS"
    curl -fsSL "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-$arch.tgz" | tar -xz -C "$TOOLS"
  fi
fi
echo "==> $("$NGROK" version)"

echo "==> Tunnel https://$NGROK_DOMAIN -> localhost:$PORT"
export NGROK_AUTHTOKEN
exec "$NGROK" http "$PORT" --url "https://$NGROK_DOMAIN" "$@"
