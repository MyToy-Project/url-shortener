#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-$HOME/url-shortener}"
COMPOSE_BIN="${COMPOSE_BIN:-docker compose}"

echo "▶ Starting url-shortener stack via Docker Compose..."

cd "$APP_DIR"

if ! command -v docker >/dev/null 2>&1; then
  echo "❌ Docker is not installed or not in PATH"
  exit 1
fi

if ! pgrep -x dockerd >/dev/null 2>&1; then
  echo "ℹ️  Starting Docker service..."
  sudo systemctl start docker
fi

# shellcheck disable=SC2086
$COMPOSE_BIN pull
# shellcheck disable=SC2086
$COMPOSE_BIN up -d --build

echo "✅ Containers are up. Use 'docker compose ps' to check status."
