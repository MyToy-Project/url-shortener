#!/usr/bin/env bash
set -e

APP_NAME="url-shortener"
APP_DIR="$HOME/url-shortener"

PID_DIR="$HOME/pid"
PID_FILE="$PID_DIR/${APP_NAME}.pid"

LOG_DIR="$HOME/logs"
LOG_FILE="$LOG_DIR/${APP_NAME}.log"

echo "▶ Starting $APP_NAME..."

cd "$APP_DIR"

mkdir -p "$PID_DIR"
mkdir -p "$LOG_DIR"

# 이미 실행 중인지 확인
if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  echo "❌ $APP_NAME is already running (PID $(cat "$PID_FILE"))"
  exit 1
fi

nohup go run ./cmd/main.go \
  > "$LOG_FILE" 2>&1 &

PID=$!
echo "$PID" > "$PID_FILE"

sleep 1

if kill -0 "$PID" 2>/dev/null; then
  echo "✅ $APP_NAME started successfully (PID $PID)"
  echo "📄 Log file: $LOG_FILE"
else
  echo "❌ Failed to start $APP_NAME"
  exit 1
fi
