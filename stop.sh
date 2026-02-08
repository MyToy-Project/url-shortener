#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-$HOME/url-shortener}"
COMPOSE_BIN="${COMPOSE_BIN:-docker compose}"

echo "▶ Stopping url-shortener stack..."

cd "$APP_DIR"

if ! command -v docker >/dev/null 2>&1; then
  echo "ℹ️  Docker is not installed; nothing to stop"
  exit 0
fi

# shellcheck disable=SC2086
$COMPOSE_BIN down

echo "✅ Containers stopped"
