#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

fresh=0
if [ "${1:-}" = "--fresh" ]; then
	fresh=1
	shift
fi

available=$(docker compose config --services)

for svc in "$@"; do
	if ! grep -qx -- "$svc" <<<"$available"; then
		echo "restart.sh: unknown service '$svc'" >&2
		echo "available: $(tr '\n' ' ' <<<"$available")" >&2
		exit 1
	fi
done

if [ "$fresh" -eq 1 ]; then
	echo "==> rebuilding ${*:-every service} from scratch"
	docker compose build --no-cache "$@"
	docker compose up -d --force-recreate "$@"
else
	echo "==> rebuilding ${*:-every service}"
	docker compose up -d --build "$@"
fi

if [ $# -eq 0 ] || printf '%s\n' "$@" | grep -qx -- app; then
	printf '==> waiting for the API on :8080'
	for _ in $(seq 1 60); do
		if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
			echo " -- up"
			exit 0
		fi
		printf '.'
		sleep 1
	done
	echo " -- no answer after 60s" >&2
	echo "    docker compose logs --tail=40 app" >&2
	exit 1
fi
