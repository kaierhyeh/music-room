#!/usr/bin/env bash

set -euo pipefail

PORT="${PORT:-8081}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOLS="$(dirname "$(dirname "$HERE")")/.tools"

CLOUDFLARED="$(command -v cloudflared || true)"
if [[ -z "$CLOUDFLARED" ]]; then
  CLOUDFLARED="$TOOLS/cloudflared"
  if [[ ! -x "$CLOUDFLARED" ]]; then
    [[ "$(uname -s)" == Linux ]] || { echo "Automatic download is Linux only: install cloudflared by hand"; exit 1; }
    case "$(uname -m)" in
      x86_64)        arch=amd64 ;;
      aarch64|arm64) arch=arm64 ;;
      *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
    echo "==> Downloading cloudflared ($arch) into $TOOLS"
    mkdir -p "$TOOLS"
    curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-$arch" \
      -o "$CLOUDFLARED.part"
    chmod +x "$CLOUDFLARED.part"
    mv "$CLOUDFLARED.part" "$CLOUDFLARED"
  fi
fi

LOG="$(mktemp -t cf-tunnel-XXXXXX.log)"

cleanup() { [[ -n "${CF_PID:-}" ]] && kill "$CF_PID" 2>/dev/null || true; rm -f "$LOG"; }
trap cleanup EXIT INT TERM

echo "==> Opening a Cloudflare tunnel to localhost:$PORT"
"$CLOUDFLARED" tunnel --url "http://localhost:$PORT" --no-autoupdate > "$LOG" 2>&1 &
CF_PID=$!

URL=""
for _ in $(seq 1 60); do
  URL=$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' "$LOG" | head -1 || true)
  [[ -n "$URL" ]] && break
  kill -0 "$CF_PID" 2>/dev/null || { echo "cloudflared exited:"; cat "$LOG"; exit 1; }
  sleep 0.5
done

[[ -n "$URL" ]] || { echo "No tunnel URL after 30s:"; cat "$LOG"; exit 1; }

echo "==> Tunnel up: $URL"
echo "==> Starting Expo (URLs and QR code will point at the tunnel)"
echo

if [[ -z "${EXPO_PUBLIC_API_URL:-}" ]]; then
  BACKEND_ENV="$(dirname "$(dirname "$HERE")")/musicroom-backend/.env"
  NGROK_DOMAIN="$(grep -E '^NGROK_DOMAIN=' "$BACKEND_ENV" 2>/dev/null | tail -1 | cut -d= -f2- | sed -E 's/^["'\'']//; s/["'\'']$//; s#^https?://##; s#/$##' || true)"
  if [[ -n "$NGROK_DOMAIN" ]]; then
    export EXPO_PUBLIC_API_URL="https://$NGROK_DOMAIN"
    echo "==> API: $EXPO_PUBLIC_API_URL (run musicroom-backend/scripts/start-ngrok.sh too)"
  else
    echo "!! NGROK_DOMAIN not found in $BACKEND_ENV: phones will not reach the API"
  fi
fi

export EXPO_PACKAGER_PROXY_URL="$URL"
exec npx expo start --port "$PORT" "$@"
