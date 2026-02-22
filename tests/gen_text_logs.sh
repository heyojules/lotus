#!/usr/bin/env bash
# gen_text_logs.sh — Sends plain text logs (syslog, bracketed, simple) to TCP port.
# Usage: ./tests/gen_text_logs.sh
# Env vars: PORT (default 4000), RATE (logs/sec, default 5), DURATION (seconds, default infinite)

set -euo pipefail

PORT="${PORT:-4000}"
RATE="${RATE:-5}"
DURATION="${DURATION:-0}"

ADJECTIVES=(blazing silent cosmic rapid frozen amber golden crimson velvet nimble hollow drifting molten steady rustic polished)
NOUNS=(falcon reef nebula piston glacier orchid beacon vortex condor sprocket tundra ember lotus cedar anvil)

HOSTNAMES=(web-01 web-02 api-01 api-02 db-01 worker-01 worker-02 cache-01 proxy-01)

MESSAGES=(
  "Request processed successfully"
  "New connection accepted from 10.0.2.$((RANDOM % 255))"
  "Closed idle connection after timeout"
  "Starting background worker"
  "Worker finished processing batch of $((RANDOM % 500 + 10)) items"
  "Listening on 0.0.0.0:$((RANDOM % 9000 + 1000))"
  "Received SIGHUP, reloading configuration"
  "SSL certificate loaded successfully"
  "Accepted connection from peer"
  "Flushing write-ahead log"
  "Compaction completed in $((RANDOM % 3000 + 100))ms"
  "Snapshot saved to disk"
  "Query plan optimized, $((RANDOM % 50 + 1)) rows scanned"
  "Index rebuild completed"
  "Health check endpoint responded 200"
)

WARN_MESSAGES=(
  "Connection pool utilization at $((RANDOM % 20 + 80))%"
  "Slow query detected: $((RANDOM % 5000 + 1000))ms"
  "Retrying failed request, attempt $((RANDOM % 3 + 2))"
  "Disk usage at $((RANDOM % 15 + 80))%"
  "Response time SLA exceeded"
  "Client disconnected unexpectedly"
)

ERROR_MESSAGES=(
  "Failed to open file: permission denied"
  "Connection refused to upstream 10.0.1.$((RANDOM % 255)):8080"
  "Segmentation fault in module worker"
  "Out of file descriptors"
  "Database connection timeout after 30s"
  "TLS handshake error: certificate expired"
  "Kernel: OOM killer invoked for process"
)

FATAL_MESSAGES=(
  "PANIC: unrecoverable state, shutting down"
  "Cannot bind to port: address already in use"
  "Fatal: corrupted write-ahead log detected"
)

# Pick 3-5 random app names
NUM_APPS=$((RANDOM % 3 + 3))
APPS=()
for ((i = 0; i < NUM_APPS; i++)); do
  adj="${ADJECTIVES[$((RANDOM % ${#ADJECTIVES[@]}))]}"
  noun="${NOUNS[$((RANDOM % ${#NOUNS[@]}))]}"
  APPS+=("${adj}-${noun}")
done

echo "Sending text logs to localhost:${PORT}"
echo "Apps: ${APPS[*]}"
echo "Rate: ${RATE}/sec | Duration: ${DURATION}s (0=infinite)"
echo "Press Ctrl+C to stop"

FIFO=$(mktemp -u /tmp/lotus_tcp_text_XXXXXX)
mkfifo "$FIFO"

cleanup_fifo() {
  rm -f "$FIFO"
}

cleanup() {
  echo ""
  echo "Stopped."
  exit 0
}
trap 'cleanup_fifo; cleanup' EXIT
trap cleanup SIGINT SIGTERM

# Start a persistent nc connection
nc localhost "$PORT" < "$FIFO" &
NC_PID=$!
exec 3>"$FIFO"

# Verify connection
sleep 0.2
if ! kill -0 "$NC_PID" 2>/dev/null; then
  echo "Failed to connect to localhost:${PORT}. Is Lotus running?" >&2
  exit 1
fi

SLEEP_INTERVAL=$(awk "BEGIN {printf \"%.4f\", 1.0 / ${RATE}}")
START_TIME=$(date +%s)
COUNT=0

while true; do
  if [[ "$DURATION" -gt 0 ]]; then
    NOW=$(date +%s)
    if (( NOW - START_TIME >= DURATION )); then
      echo "Duration reached (${DURATION}s). Sent ${COUNT} logs."
      break
    fi
  fi

  # Check if nc is still running
  if ! kill -0 "$NC_PID" 2>/dev/null; then
    echo "Connection lost to localhost:${PORT}. Reconnecting..." >&2
    nc localhost "$PORT" < "$FIFO" &
    NC_PID=$!
    sleep 0.2
    if ! kill -0 "$NC_PID" 2>/dev/null; then
      echo "Failed to reconnect. Is Lotus running?" >&2
      sleep 2
      continue
    fi
  fi

  APP="${APPS[$((RANDOM % ${#APPS[@]}))]}"
  HOST="${HOSTNAMES[$((RANDOM % ${#HOSTNAMES[@]}))]}"
  PID=$((RANDOM % 30000 + 1000))

  # Weighted severity
  ROLL=$((RANDOM % 100))
  if (( ROLL < 70 )); then
    SEV="INFO"
    MSG="${MESSAGES[$((RANDOM % ${#MESSAGES[@]}))]}"
  elif (( ROLL < 85 )); then
    SEV="WARN"
    MSG="${WARN_MESSAGES[$((RANDOM % ${#WARN_MESSAGES[@]}))]}"
  elif (( ROLL < 95 )); then
    SEV="ERROR"
    MSG="${ERROR_MESSAGES[$((RANDOM % ${#ERROR_MESSAGES[@]}))]}"
  elif (( ROLL < 98 )); then
    SEV="DEBUG"
    MSG="Debug: state=$((RANDOM % 5)) queue_len=$((RANDOM % 100))"
  else
    SEV="FATAL"
    MSG="${FATAL_MESSAGES[$((RANDOM % ${#FATAL_MESSAGES[@]}))]}"
  fi

  TS_ISO=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  TS_SYSLOG=$(date +"%b %d %H:%M:%S")
  TS_BRACKET=$(date -u +"%Y-%m-%d %H:%M:%S").$((RANDOM % 999))

  # Rotate between 3 text formats
  FORMAT=$((RANDOM % 3))

  case $FORMAT in
    0)
      # Syslog-style
      LINE="${TS_SYSLOG} ${HOST} ${APP}[${PID}]: ${SEV}: ${MSG}"
      ;;
    1)
      # Bracketed
      LINE="[${TS_BRACKET}] ${SEV} [${APP}] ${MSG}"
      ;;
    2)
      # Simple ISO
      LINE="${TS_ISO} ${SEV} ${APP} - ${MSG}"
      ;;
  esac

  echo "$LINE" >&3

  COUNT=$((COUNT + 1))
  sleep "$SLEEP_INTERVAL"
done
