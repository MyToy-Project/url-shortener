#!/usr/bin/env bash
set -e

APP_NAME="url-shortener"
PID_FILE="$HOME/pid/${APP_NAME}.pid"

echo "▶ Stopping $APP_NAME..."

if [ ! -f "$PID_FILE" ]; then
  echo "ℹ️  No PID file found"
  exit 0
fi

PID=$(cat "$PID_FILE")

if kill -0 "$PID" 2>/dev/null; then
  kill "$PID" 2>/dev/null || true
  echo "⏳ Waiting for process to stop..."

  for i in {1..10}; do
    if ! kill -0 "$PID" 2>/dev/null; then
      rm -f "$PID_FILE"
      echo "✅ $APP_NAME stopped"
      exit 0
    fi
    sleep 1
  done

  echo "⚠️  Force killing $APP_NAME"
  kill -9 "$PID" 2>/dev/null || true
  rm -f "$PID_FILE"
  echo "✅ $APP_NAME force stopped"
else
  echo "ℹ️  Process not running, cleaning up PID file"
  rm -f "$PID_FILE"
fi
