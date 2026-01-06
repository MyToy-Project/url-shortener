#!/usr/bin/env bash
set -e

APP_NAME="url-shortener"
APP_DIR="$HOME/url-shortener"

PID_DIR="$HOME/pid"
PID_FILE="$PID_DIR/${APP_NAME}.pid"

LOG_DIR="$HOME/logs"
LOG_FILE="$LOG_DIR/${APP_NAME}.log"

BIN_DIR="$APP_DIR/bin"
BIN="$BIN_DIR/${APP_NAME}"

echo "▶ Starting $APP_NAME..."

cd "$APP_DIR"
mkdir -p "$PID_DIR" "$LOG_DIR" "$BIN_DIR"

# 이미 실행 중인지 확인
if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  echo "❌ $APP_NAME is already running (PID $(cat "$PID_FILE"))"
  exit 1
fi

# 빌드
go build -o "$BIN" ./cmd

# 실행
nohup "$BIN" > "$LOG_FILE" 2>&1 &
PID=$!
echo "$PID" > "$PID_FILE"

sleep 0.2
echo "✅ $APP_NAME started successfully (PID $PID)"
echo "📄 Log file: $LOG_FILE"
