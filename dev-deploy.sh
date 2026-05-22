#!/usr/bin/env bash
#
# dev-deploy.sh — fast local dev loop for the v2 demo stack.
#
# Builds the frontend + Go binary ON THE MAC, stuffs the assets, then
# `docker cp`s the binary into the ALREADY-RUNNING container and restarts
# just the process. It deliberately does NOT run `docker compose build`,
# because every image rebuild writes fresh layers into colima's VM disk
# (which only reclaims on a restart+prune) — that churn is what kept
# filling the host disk and crashing the VM.
#
# Use this for iterating. Only fall back to `docker compose -f
# docker-compose.local.yml up -d --build app` when you change the
# Dockerfile, base image, or Go dependencies.
#
# Usage:
#   ./dev-deploy.sh            # build main app + backend, swap, restart
#   ./dev-deploy.sh --widget   # also rebuild the chat widget bundle
#
set -euo pipefail
cd "$(dirname "$0")"

CONTAINER=libredesk_v2_app
COMPOSE_FILE=docker-compose.local.yml
STUFFBIN="${STUFFBIN:-$(go env GOPATH)/bin/stuffbin}"
BUILD_DIR=.dev
BUILD_WIDGET=0
[ "${1:-}" = "--widget" ] && BUILD_WIDGET=1

if [ ! -x "$STUFFBIN" ]; then
  echo "→ stuffbin not found at $STUFFBIN — installing…"
  go install github.com/knadh/stuffbin/...@latest
fi

echo "→ [1/4] Building frontend (main$([ $BUILD_WIDGET = 1 ] && echo ' + widget'))…"
( cd frontend && pnpm build:main )
if [ $BUILD_WIDGET = 1 ]; then ( cd frontend && pnpm build:widget ); fi

echo "→ [2/4] Cross-compiling Go binary for the linux/arm64 VM…"
VERSION="$(git describe --tags --always 2>/dev/null || echo dev)"
mkdir -p "$BUILD_DIR"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
  -ldflags "-s -w -X 'main.buildString=${VERSION}' -X 'main.versionString=${VERSION}'" \
  -o "$BUILD_DIR/libredesk" ./cmd/

echo "→ [3/4] Stuffing static assets into the binary…"
"$STUFFBIN" -a stuff -in "$BUILD_DIR/libredesk" -out "$BUILD_DIR/libredesk" \
  frontend/dist i18n schema.sql static

echo "→ [4/4] Swapping binary into $CONTAINER and restarting…"
docker cp "$BUILD_DIR/libredesk" "$CONTAINER:/libredesk/libredesk"
docker compose -f "$COMPOSE_FILE" restart app >/dev/null

# Wait for health.
echo -n "→ Waiting for app…"
until curl -fs -o /dev/null http://localhost:9001/api/v1/lang/en; do
  echo -n "."
  sleep 2
done
echo " ready: http://localhost:9001  (build ${VERSION})"
