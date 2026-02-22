#!/usr/bin/env bash
# gen_json_logs.sh — Sends NDJSON logs to TCP port simulating pino, zerolog/zap, and winston formats.
# Usage: ./tests/gen_json_logs.sh
# Env vars: PORT (default 4000), RATE (logs/sec, default 5), DURATION (seconds, default infinite)

set -euo pipefail

PORT="${PORT:-4000}"
RATE="${RATE:-5}"
DURATION="${DURATION:-0}"

ADJECTIVES=(blazing silent cosmic rapid frozen amber golden crimson velvet nimble hollow drifting molten steady rustic polished)
NOUNS=(falcon reef nebula piston glacier orchid beacon vortex condor sprocket tundra ember lotus cedar anvil)

HTTP_METHODS=(GET POST PUT DELETE PATCH)
HTTP_PATHS=(/api/users /api/orders /api/products /api/auth/login /api/auth/refresh /api/health /api/search /api/payments /api/notifications /api/settings)
HTTP_CODES=(200 200 200 200 201 204 301 400 401 403 404 404 500 502 503)
SERVICES=(gateway auth-service user-service order-service payment-service notification-service search-service inventory-service)

INFO_MESSAGES=(
  "Request completed successfully"
  "Cache hit for key"
  "Database query executed"
  "Connection pool status: healthy"
  "Background job enqueued"
  "Configuration reloaded"
  "Health check passed"
  "Session validated"
  "Rate limit check passed"
  "Metrics flushed"
  "Scheduled task completed"
  "Message published to queue"
)

WARN_MESSAGES=(
  "Slow query detected"
  "Connection pool near capacity"
  "Rate limit approaching threshold"
  "Deprecated API endpoint called"
  "Cache miss ratio above threshold"
  "Retry attempt"
  "Response time exceeded SLA"
  "Memory usage above 80%"
  "Disk usage warning"
  "Certificate expiring soon"
)

ERROR_MESSAGES=(
  "Failed to connect to database"
  "Request timeout exceeded"
  "Authentication token expired"
  "Upstream service returned 503"
  "Failed to serialize response"
  "Permission denied for resource"
  "Circuit breaker tripped"
  "Message queue connection lost"
  "TLS handshake failed"
  "Out of memory error"
)

FATAL_MESSAGES=(
  "Unrecoverable database corruption detected"
  "Failed to bind to port, shutting down"
  "Panic: nil pointer dereference"
  "Configuration file missing, cannot start"
)

# Pick 3-5 random app names
NUM_APPS=$((RANDOM % 3 + 3))
APPS=()
for ((i = 0; i < NUM_APPS; i++)); do
  adj="${ADJECTIVES[$((RANDOM % ${#ADJECTIVES[@]}))]}"
  noun="${NOUNS[$((RANDOM % ${#NOUNS[@]}))]}"
  APPS+=("${adj}-${noun}")
done

echo "Sending JSON logs to localhost:${PORT}"
echo "Apps: ${APPS[*]}"
echo "Rate: ${RATE}/sec | Duration: ${DURATION}s (0=infinite)"
echo "Press Ctrl+C to stop"

cleanup() {
  echo ""
  echo "Stopped."
  exit 0
}
trap cleanup SIGINT SIGTERM

rand_uuid() {
  printf '%04x%04x-%04x-%04x-%04x-%04x%04x%04x' \
    $((RANDOM)) $((RANDOM)) $((RANDOM)) \
    $(((RANDOM & 0x0fff) | 0x4000)) \
    $(((RANDOM & 0x3fff) | 0x8000)) \
    $((RANDOM)) $((RANDOM)) $((RANDOM))
}

SLEEP_INTERVAL=$(awk "BEGIN {printf \"%.4f\", 1.0 / ${RATE}}")
START_TIME=$(date +%s)
COUNT=0

# Use a persistent TCP connection via a named pipe (FIFO)
FIFO=$(mktemp -u /tmp/lotus_tcp_XXXXXX)
mkfifo "$FIFO"

cleanup_fifo() {
  rm -f "$FIFO"
}
trap 'cleanup_fifo; cleanup' EXIT

# Start a persistent nc connection in the background, reading from the FIFO.
# Use a file descriptor to keep the FIFO open for writing.
nc localhost "$PORT" < "$FIFO" &
NC_PID=$!
exec 3>"$FIFO"

# Verify connection
sleep 0.2
if ! kill -0 "$NC_PID" 2>/dev/null; then
  echo "Failed to connect to localhost:${PORT}. Is Lotus running?" >&2
  exit 1
fi

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
  SERVICE="${SERVICES[$((RANDOM % ${#SERVICES[@]}))]}"
  REQUEST_ID=$(rand_uuid)
  USER_ID=$((RANDOM % 9000 + 1000))
  DURATION_MS=$((RANDOM % 2000 + 5))
  METHOD="${HTTP_METHODS[$((RANDOM % ${#HTTP_METHODS[@]}))]}"
  PATH_VAL="${HTTP_PATHS[$((RANDOM % ${#HTTP_PATHS[@]}))]}"
  STATUS="${HTTP_CODES[$((RANDOM % ${#HTTP_CODES[@]}))]}"
  TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  UNIX_TS=$(date +%s)

  # Weighted severity: ~70% INFO, ~15% WARN, ~10% ERROR, ~3% DEBUG, ~2% FATAL
  ROLL=$((RANDOM % 100))
  if (( ROLL < 70 )); then
    SEV="INFO"
    MSG="${INFO_MESSAGES[$((RANDOM % ${#INFO_MESSAGES[@]}))]}"
  elif (( ROLL < 85 )); then
    SEV="WARN"
    MSG="${WARN_MESSAGES[$((RANDOM % ${#WARN_MESSAGES[@]}))]}"
  elif (( ROLL < 95 )); then
    SEV="ERROR"
    MSG="${ERROR_MESSAGES[$((RANDOM % ${#ERROR_MESSAGES[@]}))]}"
  elif (( ROLL < 98 )); then
    SEV="DEBUG"
    MSG="Debug checkpoint: processing step $((RANDOM % 20 + 1))"
  else
    SEV="FATAL"
    MSG="${FATAL_MESSAGES[$((RANDOM % ${#FATAL_MESSAGES[@]}))]}"
  fi

  # Rotate between 3 logger formats
  FORMAT=$((RANDOM % 3))

  case $FORMAT in
    0)
      # Pino-style: numeric level, "msg", "time" as unix ms
      case $SEV in
        DEBUG) LVL=20 ;; INFO) LVL=30 ;; WARN) LVL=40 ;; ERROR) LVL=50 ;; FATAL) LVL=60 ;; *) LVL=30 ;;
      esac
      LINE="{\"level\":${LVL},\"time\":${UNIX_TS}000,\"msg\":\"${MSG}\",\"_app\":\"${APP}\",\"requestId\":\"${REQUEST_ID}\",\"userId\":${USER_ID},\"duration\":${DURATION_MS},\"method\":\"${METHOD}\",\"path\":\"${PATH_VAL}\",\"statusCode\":${STATUS}}"
      ;;
    1)
      # Zerolog/Zap-style: string level, "msg", "ts" as ISO
      LVL_STR=$(echo "$SEV" | tr '[:upper:]' '[:lower:]')
      LINE="{\"level\":\"${LVL_STR}\",\"ts\":\"${TS}\",\"msg\":\"${MSG}\",\"_app\":\"${APP}\",\"service.name\":\"${SERVICE}\",\"requestId\":\"${REQUEST_ID}\",\"duration\":${DURATION_MS},\"method\":\"${METHOD}\",\"path\":\"${PATH_VAL}\",\"statusCode\":${STATUS}}"
      ;;
    2)
      # Winston-style: "message" field, "timestamp"
      LINE="{\"level\":\"${SEV}\",\"timestamp\":\"${TS}\",\"message\":\"${MSG}\",\"_app\":\"${APP}\",\"service.name\":\"${SERVICE}\",\"userId\":${USER_ID},\"method\":\"${METHOD}\",\"path\":\"${PATH_VAL}\",\"statusCode\":${STATUS}}"
      ;;
  esac

  echo "$LINE" >&3

  COUNT=$((COUNT + 1))
  sleep "$SLEEP_INTERVAL"
done
